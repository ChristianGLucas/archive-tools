package nodes

import (
	"context"
	"fmt"
	"io"

	"christiangeorgelucas/archive-tools/axiom"
	gen "christiangeorgelucas/archive-tools/gen"
)

// DecompressStream decompresses a standalone compressed stream: "gzip",
// "zlib", "xz", "zstd", or "bzip2".
func DecompressStream(ctx context.Context, ax axiom.Context, input *gen.DecompressRequest) (*gen.CompressionResult, error) {
	codec := input.GetCodec()
	data := input.GetData()

	r, c, err := newDecompressReader(data, codec)
	if err != nil {
		return nil, fmt.Errorf("initializing %s decompressor: %w", codec, err)
	}
	if c != nil {
		defer c.Close()
	}

	out, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("decompressing %s stream: %w", codec, err)
	}

	return &gen.CompressionResult{
		Data:       out,
		Codec:      codec,
		InputSize:  int64(len(data)),
		OutputSize: int64(len(out)),
	}, nil
}
