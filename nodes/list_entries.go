package nodes

import (
	"context"

	"christiangeorgelucas/archive-tools/axiom"
	gen "christiangeorgelucas/archive-tools/gen"
)

// ListEntries lists every entry in a tar or zip archive (auto-detecting a
// compressed outer wrap such as .tar.gz) with its metadata — no entry
// bytes are read or returned.
func ListEntries(ctx context.Context, ax axiom.Context, input *gen.ArchiveInput) (*gen.EntryList, error) {
	oc, err := openContainer(input.GetData(), input.GetFormatHint())
	if err != nil {
		return nil, err
	}
	raws, err := walkHeaders(oc)
	if err != nil {
		return nil, err
	}
	out := &gen.EntryList{Count: int32(len(raws))}
	for _, re := range raws {
		out.Entries = append(out.Entries, toArchiveEntry(re))
	}
	return out, nil
}

func toArchiveEntry(re rawEntry) *gen.ArchiveEntry {
	return &gen.ArchiveEntry{
		Path:           re.path,
		Size:           re.size,
		Mode:           re.mode,
		Mtime:          re.mtimeUnix,
		Type:           re.typ,
		CompressedSize: re.compressed,
		SymlinkTarget:  re.symlinkTarget,
		PathSafe:       re.pathSafe,
	}
}
