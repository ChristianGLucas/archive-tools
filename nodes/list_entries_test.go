package nodes

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"

	gen "christiangeorgelucas/archive-tools/gen"
)

func TestListEntries_TarAgainstSystemTar(t *testing.T) {
	tarBytes := refTarWithSymlink(t, "file.txt", "hello world", "adir", "alink", "file.txt")
	out, err := ListEntries(testCtxBG, testAx, &gen.ArchiveInput{Data: tarBytes})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	byPath := map[string]*gen.ArchiveEntry{}
	for _, e := range out.Entries {
		byPath[e.Path] = e
	}
	f, ok := byPath["file.txt"]
	if !ok || f.Type != gen.EntryType_ENTRY_TYPE_FILE || f.Size != int64(len("hello world")) || !f.PathSafe {
		t.Fatalf("file.txt entry wrong: %+v", f)
	}
	d, ok := byPath["adir/"]
	if !ok {
		// some tar implementations record the dir without trailing slash
		d, ok = byPath["adir"]
	}
	if !ok || d.Type != gen.EntryType_ENTRY_TYPE_DIR {
		t.Fatalf("adir entry wrong or missing: %+v (all: %+v)", d, byPath)
	}
	l, ok := byPath["alink"]
	if !ok || l.Type != gen.EntryType_ENTRY_TYPE_SYMLINK || l.SymlinkTarget != "file.txt" {
		t.Fatalf("alink entry wrong: %+v", l)
	}
}

func TestListEntries_ZipAgainstSystemZip(t *testing.T) {
	zipBytes := refZip(t, map[string]string{"one.txt": "111", "sub/two.txt": "22"})
	out, err := ListEntries(testCtxBG, testAx, &gen.ArchiveInput{Data: zipBytes})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Count != int32(len(out.Entries)) {
		t.Fatalf("count %d != len(entries) %d", out.Count, len(out.Entries))
	}
	found := map[string]bool{}
	for _, e := range out.Entries {
		found[e.Path] = true
		if e.Path == "one.txt" && e.Size != 3 {
			t.Fatalf("one.txt size = %d, want 3", e.Size)
		}
	}
	if !found["one.txt"] || !found["sub/two.txt"] {
		t.Fatalf("missing expected entries: %+v", out.Entries)
	}
}

func TestListEntries_UnsafePathFlagged(t *testing.T) {
	// Hand-build a minimal tar with a traversal entry name using our own
	// writeTar (already proven against the system tar reader elsewhere) —
	// here we only need SOME tar bytes containing an unsafe name; the
	// system `tar` CLI refuses to create such archives directly, so we use
	// our own writer with strictPaths bypassed via direct struct construction.
	data, err := writeTar([]rawEntry{{path: "../../etc/passwd", typ: gen.EntryType_ENTRY_TYPE_FILE, data: []byte("x"), mode: 0o644}})
	if err != nil {
		t.Fatalf("writeTar: %v", err)
	}
	out, err := ListEntries(testCtxBG, testAx, &gen.ArchiveInput{Data: data})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Entries) != 1 || out.Entries[0].PathSafe {
		t.Fatalf("expected one entry flagged path_safe=false, got %+v", out.Entries)
	}
}

// TestListEntries_RootDotEntryIsSafe is a regression test: sanitizePath
// used to flag a tar's own "./" root-directory entry (produced by the
// extremely common `tar -C dir .` idiom) as unsafe, which put it in
// skipped_unsafe_paths / path_safe=false right alongside genuine zip-slip
// attempts on every ordinary archive built that way — a false positive on
// the package's own headline security signal.
func TestListEntries_RootDotEntryIsSafe(t *testing.T) {
	requireTool(t, "tar")
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/f.txt", []byte("x"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cmd := exec.Command("tar", "-czf", "-", "-C", dir, ".")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("system tar -C dir . failed: %v", err)
	}
	gz := out.Bytes()

	result, err := ListEntries(testCtxBG, testAx, &gen.ArchiveInput{Data: gz})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, e := range result.Entries {
		if (e.Path == "./" || e.Path == ".") && !e.PathSafe {
			t.Fatalf("root entry %q incorrectly flagged unsafe", e.Path)
		}
	}

	extracted, err := ExtractAll(testCtxBG, testAx, &gen.ArchiveInput{Data: gz})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, p := range extracted.SkippedUnsafePaths {
		if p == "./" || p == "." {
			t.Fatalf("root entry %q incorrectly reported as skipped-unsafe", p)
		}
	}
}

func TestListEntries_MalformedInput(t *testing.T) {
	_, err := ListEntries(testCtxBG, testAx, &gen.ArchiveInput{Data: []byte("this is not an archive")})
	if err == nil {
		t.Fatal("expected an error for non-archive input, got nil")
	}
}

func TestListEntries_ManyEntriesReturnedWhole(t *testing.T) {
	// This package no longer imposes its own entry-count ceiling — an
	// archive with many entries should come back whole, not truncated.
	const n = 100
	entries := make([]rawEntry, n)
	for i := range entries {
		entries[i] = rawEntry{path: pad(i), typ: gen.EntryType_ENTRY_TYPE_FILE}
	}
	data, err := writeTar(entries)
	if err != nil {
		t.Fatalf("writeTar: %v", err)
	}
	out, err := ListEntries(testCtxBG, testAx, &gen.ArchiveInput{Data: data})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Entries) != n {
		t.Fatalf("expected exactly %d entries, got %d", n, len(out.Entries))
	}
}

func pad(i int) string {
	return fmt.Sprintf("f%08d", i)
}
