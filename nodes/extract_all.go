package nodes

import (
	"context"

	"christiangeorgelucas/archive-tools/axiom"
	gen "christiangeorgelucas/archive-tools/gen"
)

// ExtractAll extracts every FILE-type entry's bytes from a tar or zip
// archive (auto-detecting a compressed outer wrap) into a normalized
// in-memory list — never to a real filesystem. An entry with an unsafe
// path is excluded and reported in skipped_unsafe_paths rather than being
// silently dropped or honored.
func ExtractAll(ctx context.Context, ax axiom.Context, input *gen.ArchiveInput) (*gen.ExtractAllResult, error) {
	oc, err := openContainer(input.GetData(), input.GetFormatHint())
	if err != nil {
		return nil, err
	}
	raws, skipped, err := walkData(oc, false)
	if err != nil {
		return nil, err
	}

	out := &gen.ExtractAllResult{SkippedUnsafePaths: skipped}
	for _, re := range raws {
		if re.typ != gen.EntryType_ENTRY_TYPE_FILE {
			continue
		}
		out.Entries = append(out.Entries, &gen.EntryData{Path: re.path, Data: re.data, Size: re.size})
	}
	out.Count = int32(len(out.Entries))
	return out, nil
}
