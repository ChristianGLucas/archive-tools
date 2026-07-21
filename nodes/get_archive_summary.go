package nodes

import (
	"context"

	"christiangeorgelucas/archive-tools/axiom"
	gen "christiangeorgelucas/archive-tools/gen"
)

// GetArchiveSummary reports archive-level totals (entry count, sizes,
// whether it contains directories/symlinks) without returning a per-entry
// list. Like ListEntries, it reads entry headers only, never entry data.
func GetArchiveSummary(ctx context.Context, ax axiom.Context, input *gen.ArchiveInput) (*gen.ArchiveSummary, error) {
	if err := checkRawInputSize(input.GetData()); err != nil {
		return nil, err
	}
	oc, err := openContainer(input.GetData(), input.GetFormatHint())
	if err != nil {
		return nil, err
	}
	raws, truncated, err := walkHeaders(oc)
	if err != nil {
		return nil, err
	}

	out := &gen.ArchiveSummary{
		ContainerFormat: oc.kind,
		Compression:     oc.compression,
		EntryCount:      int32(len(raws)),
		Truncated:       truncated,
	}
	for _, re := range raws {
		switch re.typ {
		case gen.EntryType_ENTRY_TYPE_DIR:
			out.HasDirs = true
		case gen.EntryType_ENTRY_TYPE_SYMLINK:
			out.HasSymlinks = true
		case gen.EntryType_ENTRY_TYPE_FILE:
			out.TotalUncompressedSize += re.size
			out.TotalCompressedSize += re.compressed
		}
	}
	return out, nil
}
