package nodes

import (
	"context"
	"fmt"

	"christiangeorgelucas/archive-tools/axiom"
	gen "christiangeorgelucas/archive-tools/gen"
)

// entriesFromCreateRequest validates and converts a CreateArchiveRequest's
// caller-supplied entries into this package's internal rawEntry form,
// shared by CreateTar and CreateZip. Every entry is caller-supplied, so an
// unsafe path fails the whole request rather than being silently dropped.
// A path ending in "/" is treated as a directory entry (its data, if any,
// is ignored); everything else is a regular file.
func entriesFromCreateRequest(req *gen.CreateArchiveRequest) ([]rawEntry, error) {
	entries := make([]rawEntry, 0, len(req.GetEntries()))
	for _, ce := range req.GetEntries() {
		p := ce.GetPath()
		if !sanitizePath(p) {
			return nil, fmt.Errorf("entry path %q is unsafe (absolute or escapes via \"..\") — refusing to create it", p)
		}
		if isDirPath(p) {
			entries = append(entries, rawEntry{path: p, typ: gen.EntryType_ENTRY_TYPE_DIR, mode: ce.GetMode()})
			continue
		}
		entries = append(entries, rawEntry{
			path: p,
			data: ce.GetData(),
			size: int64(len(ce.GetData())),
			mode: ce.GetMode(),
			typ:  gen.EntryType_ENTRY_TYPE_FILE,
		})
	}
	return entries, nil
}

func isDirPath(p string) bool {
	return len(p) > 0 && p[len(p)-1] == '/'
}

func archiveResult(data []byte, entryCount int) *gen.ArchiveResult {
	return &gen.ArchiveResult{Data: data, EntryCount: int32(entryCount), Size: int64(len(data))}
}

// CreateTar builds a fresh, uncompressed tar archive from a list of
// in-memory {path, data, mode} entries — entirely in memory.
func CreateTar(ctx context.Context, ax axiom.Context, input *gen.CreateArchiveRequest) (*gen.ArchiveResult, error) {
	entries, err := entriesFromCreateRequest(input)
	if err != nil {
		return nil, err
	}
	data, err := writeTar(entries)
	if err != nil {
		return nil, err
	}
	return archiveResult(data, len(entries)), nil
}
