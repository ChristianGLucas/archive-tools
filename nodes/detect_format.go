package nodes

import (
	"context"

	"christiangeorgelucas/archive-tools/axiom"
	gen "christiangeorgelucas/archive-tools/gen"
)

// DetectFormat identifies a byte blob's archive container format (tar/zip)
// and outer compression wrap (gzip/bzip2/xz/zstd/none) from its leading
// magic bytes, without fully parsing or decompressing it (only a small
// prefix of any compressed stream is peeked, to check for a tar signature
// inside).
func DetectFormat(ctx context.Context, ax axiom.Context, input *gen.ArchiveInput) (*gen.FormatInfo, error) {
	return detectFormatInfo(input.GetData()), nil
}
