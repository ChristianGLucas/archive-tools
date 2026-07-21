package nodes

import (
	"context"
	"fmt"

	"christiangeorgelucas/archive-tools/axiom"
	gen "christiangeorgelucas/archive-tools/gen"
)

// AddEntry returns a copy of an existing (uncompressed) tar or zip archive
// with one entry appended. If path already exists, the new entry is added
// alongside it (both are present in the output, matching standard
// tar/zip "last entry wins on read" semantics) — this is not an atomic
// replace. Rejects an unsafe path (either the new entry's, or any
// pre-existing entry's) with a structured error rather than carrying it
// forward silently.
func AddEntry(ctx context.Context, ax axiom.Context, input *gen.AddEntryRequest) (*gen.ArchiveResult, error) {
	if err := checkRawInputSize(input.GetData()); err != nil {
		return nil, err
	}
	newPath := input.GetPath()
	if !sanitizePath(newPath) {
		return nil, fmt.Errorf("entry path %q is unsafe (absolute or escapes via \"..\") — refusing to add it", newPath)
	}
	if int64(len(input.GetEntryData())) > maxTotalUncompressedBytes {
		return nil, fmt.Errorf("new entry is %d bytes, exceeding the %d-byte size cap", len(input.GetEntryData()), maxTotalUncompressedBytes)
	}

	oc, err := openContainer(input.GetData(), input.GetFormatHint())
	if err != nil {
		return nil, err
	}
	entries, _, truncated, err := walkData(oc, true)
	if err != nil {
		return nil, err
	}
	if truncated {
		return nil, fmt.Errorf("source archive exceeds this package's entry-count/size caps — refusing to modify a partially-read archive")
	}

	entries = append(entries, rawEntry{
		path: newPath,
		data: input.GetEntryData(),
		size: int64(len(input.GetEntryData())),
		mode: input.GetMode(),
		typ:  gen.EntryType_ENTRY_TYPE_FILE,
	})

	data, err := writeContainer(oc.kind, entries)
	if err != nil {
		return nil, err
	}
	return archiveResult(data, len(entries)), nil
}

func writeContainer(kind string, entries []rawEntry) ([]byte, error) {
	switch kind {
	case "tar":
		return writeTar(entries)
	case "zip":
		return writeZip(entries)
	default:
		return nil, fmt.Errorf("internal error: unknown container kind %q", kind)
	}
}
