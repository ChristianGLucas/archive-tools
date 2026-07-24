// Package nodes implements christiangeorgelucas/archive-tools — thin,
// stateless wrappers around the Go standard library (archive/tar,
// archive/zip, compress/gzip, compress/bzip2, compress/zlib) plus two
// zero-dependency, permissively-licensed pure-Go libraries:
// github.com/klauspost/compress (zstd) and github.com/ulikunitz/xz (xz).
//
// This file holds logic shared by every node: format sniffing from magic
// bytes, path-traversal (zip-slip) sanitizing, decompression, and the
// common "open an archive, walk its entries" machinery.
//
// SAFETY MODEL (see the package-level proto comment for the caller-facing
// summary):
//   - Payload size, entry-count, and decompressed-output-size limits are
//     NOT enforced here — the platform's ingress/transport already bounds
//     request/response size, and the node sandbox bounds memory/CPU/time
//     for a runaway decompression. This package owns domain correctness
//     (is this a valid archive?) and path safety, not resource limits.
//   - Every entry path is sanitized against zip-slip / traversal. Nothing
//     is ever written to a real filesystem, so there is no "unsafe
//     extraction" in the traditional sense — but an unsafe path is still
//     never silently honored: single-entry / caller-supplied-path
//     operations error immediately, and bulk reads of a possibly-hostile
//     archive skip the unsafe entry and report it rather than returning it.
//   - Symlink entries are surfaced as metadata (type + target string) only.
//     Nothing ever resolves or follows a symlink target.
package nodes

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"

	gen "christiangeorgelucas/archive-tools/gen"
)

// defaultFileMode is used for a created entry that supplies mode=0.
const defaultFileMode = 0o644

// ---------------------------------------------------------------------
// Path safety (zip-slip / traversal)
// ---------------------------------------------------------------------

// sanitizePath reports whether p is safe to treat as a relative path under
// a notional extraction root: not empty, not absolute, contains no NUL
// byte, no backslash (never meaningful here — real tar/zip entries use
// forward slashes; a literal backslash is only ever an attempted traversal
// on this package's Unix-style path handling), no Windows drive letter,
// and does not escape the root via a ".." component once lexically
// cleaned.
//
// A path that cleans to exactly "." (e.g. the literal string ".", or "./",
// or "a/..") is treated as SAFE: it names the container's own root, which
// the ubiquitous `tar -C dir .` idiom records as an explicit "./" entry.
// It carries no traversal risk (there is nothing to escape to) — flagging
// it as unsafe only produced false positives on completely ordinary
// archives without catching any real zip-slip attempt.
func sanitizePath(p string) (safe bool) {
	if p == "" {
		return false
	}
	if strings.ContainsRune(p, 0) {
		return false
	}
	if strings.Contains(p, "\\") {
		return false
	}
	if len(p) >= 2 && p[1] == ':' {
		return false // e.g. "C:/x" — Windows-style absolute
	}
	if strings.HasPrefix(p, "/") {
		return false
	}
	cleaned := path.Clean(p)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return false
	}
	return true
}

// ---------------------------------------------------------------------
// Format sniffing
// ---------------------------------------------------------------------

func looksLikeZip(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	sig := data[:4]
	return bytes.Equal(sig, []byte{'P', 'K', 0x03, 0x04}) || // local file header
		bytes.Equal(sig, []byte{'P', 'K', 0x05, 0x06}) || // empty archive (end of central dir)
		bytes.Equal(sig, []byte{'P', 'K', 0x07, 0x08}) // spanned archive
}

// looksLikeTar checks for the POSIX/GNU ustar magic at byte offset 257,
// the same heuristic widely used to recognize tar without a dedicated
// leading-byte signature (tar has none — this is the standard check). A
// pre-POSIX ("V7") tar lacking this magic is not auto-detected; pass
// format_hint="tar" to force it.
func looksLikeTar(data []byte) bool {
	if len(data) < 262 {
		return false
	}
	return string(data[257:262]) == "ustar"
}

// sniffCompression identifies an outer compression wrap from its magic
// bytes, or "" if none of the four recognized codecs match.
func sniffCompression(data []byte) string {
	switch {
	case len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b:
		return "gzip"
	case len(data) >= 3 && string(data[:3]) == "BZh":
		return "bzip2"
	case len(data) >= 6 && bytes.Equal(data[:6], []byte{0xFD, '7', 'z', 'X', 'Z', 0x00}):
		return "xz"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x28, 0xB5, 0x2F, 0xFD}):
		return "zstd"
	default:
		return ""
	}
}

// detectFormatInfo implements the DetectFormat node's logic: identify the
// container + compression layers from magic bytes, peeking only a small
// prefix of any decompressed stream rather than fully decompressing it.
func detectFormatInfo(data []byte) *gen.FormatInfo {
	if looksLikeZip(data) {
		return &gen.FormatInfo{Recognized: true, ContainerFormat: "zip", Compression: "none", Label: "zip"}
	}
	if comp := sniffCompression(data); comp != "" {
		peek := peekDecompressed(data, comp, 300)
		if looksLikeTar(peek) {
			return &gen.FormatInfo{Recognized: true, ContainerFormat: "tar", Compression: comp, Label: "tar+" + comp}
		}
		return &gen.FormatInfo{Recognized: true, ContainerFormat: "", Compression: comp, Label: comp}
	}
	if looksLikeTar(data) {
		return &gen.FormatInfo{Recognized: true, ContainerFormat: "tar", Compression: "none", Label: "tar"}
	}
	return &gen.FormatInfo{Recognized: false, Compression: "none"}
}

// peekDecompressed decompresses at most n bytes of data under codec,
// tolerating truncation (EOF mid-stream) since only a magic-byte peek is
// wanted, not a full valid stream. Returns nil if the codec's reader
// cannot even be initialized (data doesn't really match the codec despite
// the magic-byte prefix match, or is corrupt).
func peekDecompressed(data []byte, codec string, n int) []byte {
	r, c, err := newDecompressReader(data, codec)
	if err != nil {
		return nil
	}
	if c != nil {
		defer c.Close()
	}
	buf := make([]byte, n)
	read, _ := io.ReadFull(r, buf)
	return buf[:read]
}

type closer interface {
	Close() error
}

// newDecompressReader constructs the appropriate stdlib/pure-Go
// decompressing io.Reader for codec over data. The returned closer (nil
// for codecs without one) must be closed by the caller when non-nil.
func newDecompressReader(data []byte, codec string) (io.Reader, closer, error) {
	switch codec {
	case "gzip":
		gz, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		return gz, gz, nil
	case "bzip2":
		return bzip2.NewReader(bytes.NewReader(data)), nil, nil
	case "xz":
		xr, err := xz.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		return xr, nil, nil
	case "zstd":
		zr, err := zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		return zr, zstdCloser{zr}, nil
	case "zlib":
		zl, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		return zl, zl, nil
	default:
		return nil, nil, fmt.Errorf("unrecognized compression codec %q — expected one of gzip, bzip2, xz, zstd, zlib", codec)
	}
}

// zstdCloser adapts *zstd.Decoder's void Close() to the closer interface.
type zstdCloser struct{ d *zstd.Decoder }

func (z zstdCloser) Close() error { z.d.Close(); return nil }

// ---------------------------------------------------------------------
// Reading (decompressing/materializing full content — no size ceiling
// here; the platform's ingress/transport and node sandbox already bound
// request/response size and contain a runaway decompression)
// ---------------------------------------------------------------------

func decompressAll(data []byte, codec string) ([]byte, error) {
	r, c, err := newDecompressReader(data, codec)
	if err != nil {
		return nil, err
	}
	if c != nil {
		defer c.Close()
	}
	return io.ReadAll(r)
}

// ---------------------------------------------------------------------
// Opening a container (auto-detect or format_hint)
// ---------------------------------------------------------------------

type openedContainer struct {
	kind        string // "tar" or "zip"
	compression string // "none", "gzip", "bzip2", "xz", "zstd"
	// tarBytes holds the fully-decompressed raw tar bytes when kind=="tar".
	tarBytes []byte
	// zipBytes holds the (never outer-compressed) zip bytes when kind=="zip".
	zipBytes []byte
}

var validHints = map[string][2]string{
	"tar":     {"tar", "none"},
	"zip":     {"zip", "none"},
	"tar.gz":  {"tar", "gzip"},
	"tar.bz2": {"tar", "bzip2"},
	"tar.xz":  {"tar", "xz"},
	"tar.zst": {"tar", "zstd"},
}

// openContainer opens data as an archive, honoring an explicit format_hint
// or auto-detecting from magic bytes. It fully materializes the container
// bytes (decompressing any outer wrap) so the tar/zip readers can operate
// on it directly.
func openContainer(data []byte, formatHint string) (*openedContainer, error) {
	if formatHint != "" {
		pair, ok := validHints[formatHint]
		if !ok {
			return nil, fmt.Errorf("unrecognized format_hint %q — expected one of tar, zip, tar.gz, tar.bz2, tar.xz, tar.zst", formatHint)
		}
		kind, compression := pair[0], pair[1]
		if kind == "zip" {
			if !looksLikeZip(data) {
				return nil, fmt.Errorf("format_hint \"zip\" but input does not start with a zip signature")
			}
			return &openedContainer{kind: "zip", compression: "none", zipBytes: data}, nil
		}
		if compression == "none" {
			return &openedContainer{kind: "tar", compression: "none", tarBytes: data}, nil
		}
		raw, err := decompressAll(data, compression)
		if err != nil {
			return nil, fmt.Errorf("decompressing %s outer stream: %w", compression, err)
		}
		return &openedContainer{kind: "tar", compression: compression, tarBytes: raw}, nil
	}

	// Auto-detect.
	if looksLikeZip(data) {
		return &openedContainer{kind: "zip", compression: "none", zipBytes: data}, nil
	}
	if comp := sniffCompression(data); comp != "" {
		raw, err := decompressAll(data, comp)
		if err != nil {
			return nil, fmt.Errorf("decompressing detected %s outer stream: %w", comp, err)
		}
		if !looksLikeTar(raw) {
			return nil, fmt.Errorf("decompressed %s stream does not contain a recognizable tar archive", comp)
		}
		return &openedContainer{kind: "tar", compression: comp, tarBytes: raw}, nil
	}
	if looksLikeTar(data) {
		return &openedContainer{kind: "tar", compression: "none", tarBytes: data}, nil
	}
	return nil, fmt.Errorf("unrecognized archive format: no zip or tar signature found (also checked for a gzip/bzip2/xz/zstd outer wrap around a tar)")
}

// ---------------------------------------------------------------------
// Entry metadata + data model shared across List/Summary/Extract/Read/
// Add/Remove/Convert
// ---------------------------------------------------------------------

// rawEntry is this package's normalized in-memory view of one archive
// entry, used both when reading an existing archive and when writing a
// new/modified one.
type rawEntry struct {
	path          string
	data          []byte // populated for FILE entries only, when actually read
	size          int64  // declared size (headers-only) or len(data) once read
	mode          uint32
	mtimeUnix     int64
	typ           gen.EntryType
	symlinkTarget string
	compressed    int64 // zip only; 0 for tar
	pathSafe      bool
}

// walkHeaders enumerates every entry's metadata without reading any entry
// DATA (tar: header block only; zip: central-directory record only) — the
// cheap path used by ListEntries and GetArchiveSummary.
func walkHeaders(oc *openedContainer) (entries []rawEntry, err error) {
	switch oc.kind {
	case "tar":
		tr := tar.NewReader(bytes.NewReader(oc.tarBytes))
		for {
			hdr, terr := tr.Next()
			if terr == io.EOF {
				return entries, nil
			}
			if terr != nil {
				return nil, fmt.Errorf("reading tar header: %w", terr)
			}
			entries = append(entries, rawEntryFromTarHeader(hdr))
		}
	case "zip":
		zr, zerr := zip.NewReader(bytes.NewReader(oc.zipBytes), int64(len(oc.zipBytes)))
		if zerr != nil {
			return nil, fmt.Errorf("reading zip central directory: %w", zerr)
		}
		for _, f := range zr.File {
			entries = append(entries, rawEntryFromZipFile(f))
		}
		return entries, nil
	default:
		return nil, fmt.Errorf("internal error: unknown container kind %q", oc.kind)
	}
}

// walkData enumerates every entry AND reads FILE entry data. When
// strictPaths is true, an unsafe entry path aborts the whole call with an
// error (used by AddEntry/RemoveEntry/ConvertContainer, which must not
// silently drop or re-emit a source entry). When false, an unsafe entry is
// skipped and its path recorded in skippedUnsafe instead (used by
// ExtractAll).
func walkData(oc *openedContainer, strictPaths bool) (entries []rawEntry, skippedUnsafe []string, err error) {
	// process returns true when the caller should stop iterating because
	// err was just set.
	process := func(re rawEntry, r io.Reader) bool {
		if !re.pathSafe {
			if strictPaths {
				err = fmt.Errorf("archive contains an unsafe entry path %q (absolute or escapes via \"..\") — refusing to carry it forward", re.path)
				return true
			}
			skippedUnsafe = append(skippedUnsafe, re.path)
			return false
		}
		if re.typ == gen.EntryType_ENTRY_TYPE_FILE && r != nil {
			buf, rerr := io.ReadAll(r)
			if rerr != nil {
				err = fmt.Errorf("reading entry %q: %w", re.path, rerr)
				return true
			}
			re.data = buf
			re.size = int64(len(buf))
			entries = append(entries, re)
			return false
		}
		entries = append(entries, re)
		return false
	}

	switch oc.kind {
	case "tar":
		tr := tar.NewReader(bytes.NewReader(oc.tarBytes))
		for {
			hdr, terr := tr.Next()
			if terr == io.EOF {
				break
			}
			if terr != nil {
				err = fmt.Errorf("reading tar header: %w", terr)
				return
			}
			if process(rawEntryFromTarHeader(hdr), tr) {
				return
			}
		}
	case "zip":
		zr, zerr := zip.NewReader(bytes.NewReader(oc.zipBytes), int64(len(oc.zipBytes)))
		if zerr != nil {
			err = fmt.Errorf("reading zip central directory: %w", zerr)
			return
		}
		for _, f := range zr.File {
			re := rawEntryFromZipFile(f)
			switch re.typ {
			case gen.EntryType_ENTRY_TYPE_SYMLINK:
				rc, operr := f.Open()
				if operr != nil {
					err = fmt.Errorf("opening zip entry %q: %w", f.Name, operr)
					return
				}
				target, _ := io.ReadAll(rc)
				rc.Close()
				re.symlinkTarget = string(target)
				if process(re, nil) {
					return
				}
			case gen.EntryType_ENTRY_TYPE_FILE:
				rc, operr := f.Open()
				if operr != nil {
					err = fmt.Errorf("opening zip entry %q: %w", f.Name, operr)
					return
				}
				stop := process(re, rc)
				rc.Close()
				if stop {
					return
				}
			default:
				if process(re, nil) {
					return
				}
			}
		}
	default:
		err = fmt.Errorf("internal error: unknown container kind %q", oc.kind)
	}
	return
}

// findEntry scans for exactly one named entry (used by ReadEntry) and
// reads its full decompressed data.
func findEntry(oc *openedContainer, target string) (*rawEntry, error) {
	switch oc.kind {
	case "tar":
		tr := tar.NewReader(bytes.NewReader(oc.tarBytes))
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				return nil, fmt.Errorf("no entry with path %q found in the archive", target)
			}
			if err != nil {
				return nil, fmt.Errorf("reading tar header: %w", err)
			}
			if hdr.Name != target {
				continue
			}
			re := rawEntryFromTarHeader(hdr)
			if re.typ != gen.EntryType_ENTRY_TYPE_FILE {
				return nil, fmt.Errorf("path %q is not a regular file (type=%s) — nothing to read", target, re.typ.String())
			}
			buf, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("reading entry %q: %w", target, err)
			}
			re.data = buf
			re.size = int64(len(buf))
			return &re, nil
		}
	case "zip":
		zr, err := zip.NewReader(bytes.NewReader(oc.zipBytes), int64(len(oc.zipBytes)))
		if err != nil {
			return nil, fmt.Errorf("reading zip central directory: %w", err)
		}
		for _, f := range zr.File {
			if f.Name != target {
				continue
			}
			re := rawEntryFromZipFile(f)
			if re.typ != gen.EntryType_ENTRY_TYPE_FILE {
				return nil, fmt.Errorf("path %q is not a regular file (type=%s) — nothing to read", target, re.typ.String())
			}
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("opening zip entry %q: %w", target, err)
			}
			defer rc.Close()
			buf, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("reading entry %q: %w", target, err)
			}
			re.data = buf
			re.size = int64(len(buf))
			return &re, nil
		}
		return nil, fmt.Errorf("no entry with path %q found in the archive", target)
	default:
		return nil, fmt.Errorf("internal error: unknown container kind %q", oc.kind)
	}
}

func rawEntryFromTarHeader(hdr *tar.Header) rawEntry {
	safe := sanitizePath(hdr.Name)
	typ := gen.EntryType_ENTRY_TYPE_OTHER
	switch hdr.Typeflag {
	case tar.TypeReg, tar.TypeRegA:
		typ = gen.EntryType_ENTRY_TYPE_FILE
	case tar.TypeDir:
		typ = gen.EntryType_ENTRY_TYPE_DIR
	case tar.TypeSymlink:
		typ = gen.EntryType_ENTRY_TYPE_SYMLINK
	}
	return rawEntry{
		path:          hdr.Name,
		size:          hdr.Size,
		mode:          uint32(hdr.Mode) & 0o7777,
		mtimeUnix:     hdr.ModTime.Unix(),
		typ:           typ,
		symlinkTarget: hdr.Linkname,
		pathSafe:      safe,
	}
}

func rawEntryFromZipFile(f *zip.File) rawEntry {
	safe := sanitizePath(f.Name)
	mode := f.Mode()
	typ := gen.EntryType_ENTRY_TYPE_FILE
	switch {
	case mode.IsDir():
		typ = gen.EntryType_ENTRY_TYPE_DIR
	case mode&os.ModeSymlink != 0:
		typ = gen.EntryType_ENTRY_TYPE_SYMLINK
	}
	mtime := f.Modified
	if mtime.IsZero() {
		mtime = f.ModTime()
	}
	return rawEntry{
		path:       f.Name,
		size:       int64(f.UncompressedSize64),
		mode:       uint32(mode.Perm()),
		mtimeUnix:  mtime.Unix(),
		typ:        typ,
		compressed: int64(f.CompressedSize64),
		pathSafe:   safe,
	}
}

// ---------------------------------------------------------------------
// Writing archives (CreateTar / CreateZip / AddEntry / RemoveEntry /
// ConvertContainer)
// ---------------------------------------------------------------------

func modeOrDefault(m uint32) uint32 {
	if m == 0 {
		return defaultFileMode
	}
	return m
}

// writeTar renders entries into a fresh, uncompressed tar archive.
func writeTar(entries []rawEntry) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		hdr := &tar.Header{
			Name:    e.path,
			Mode:    int64(modeOrDefault(e.mode)),
			ModTime: unixTime(e.mtimeUnix),
		}
		switch e.typ {
		case gen.EntryType_ENTRY_TYPE_DIR:
			hdr.Typeflag = tar.TypeDir
			hdr.Name = strings.TrimSuffix(e.path, "/") + "/"
		case gen.EntryType_ENTRY_TYPE_SYMLINK:
			hdr.Typeflag = tar.TypeSymlink
			hdr.Linkname = e.symlinkTarget
		default:
			hdr.Typeflag = tar.TypeReg
			hdr.Size = int64(len(e.data))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, fmt.Errorf("writing tar header for %q: %w", e.path, err)
		}
		if hdr.Typeflag == tar.TypeReg {
			if _, err := tw.Write(e.data); err != nil {
				return nil, fmt.Errorf("writing tar data for %q: %w", e.path, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("finalizing tar archive: %w", err)
	}
	return buf.Bytes(), nil
}

// writeZip renders entries into a fresh zip archive (DEFLATE-compressed
// file entries).
func writeZip(entries []rawEntry) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		name := e.path
		hdr := &zip.FileHeader{Name: name, Modified: unixTime(e.mtimeUnix)}
		switch e.typ {
		case gen.EntryType_ENTRY_TYPE_DIR:
			hdr.Name = strings.TrimSuffix(name, "/") + "/"
			hdr.SetMode(os.FileMode(modeOrDefault(e.mode)) | os.ModeDir)
			if _, err := zw.CreateHeader(hdr); err != nil {
				return nil, fmt.Errorf("writing zip dir header for %q: %w", e.path, err)
			}
		case gen.EntryType_ENTRY_TYPE_SYMLINK:
			hdr.SetMode(os.FileMode(modeOrDefault(e.mode)) | os.ModeSymlink)
			w, err := zw.CreateHeader(hdr)
			if err != nil {
				return nil, fmt.Errorf("writing zip symlink header for %q: %w", e.path, err)
			}
			if _, err := w.Write([]byte(e.symlinkTarget)); err != nil {
				return nil, fmt.Errorf("writing zip symlink target for %q: %w", e.path, err)
			}
		default:
			hdr.Method = zip.Deflate
			hdr.SetMode(os.FileMode(modeOrDefault(e.mode)))
			w, err := zw.CreateHeader(hdr)
			if err != nil {
				return nil, fmt.Errorf("writing zip header for %q: %w", e.path, err)
			}
			if _, err := w.Write(e.data); err != nil {
				return nil, fmt.Errorf("writing zip data for %q: %w", e.path, err)
			}
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("finalizing zip archive: %w", err)
	}
	return buf.Bytes(), nil
}

// unixTime converts a Unix-epoch-seconds value (0 means "unset/epoch") into
// a UTC time.Time for tar/zip header fields.
func unixTime(sec int64) time.Time {
	return time.Unix(sec, 0).UTC()
}
