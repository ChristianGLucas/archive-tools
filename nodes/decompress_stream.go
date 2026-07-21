package nodes

import (
	"context"
	"fmt"

	"christiangeorgelucas/archive-tools/axiom"
	gen "christiangeorgelucas/archive-tools/gen"
)

// DecompressStream decompresses a standalone compressed stream: "gzip",
// "zlib", "xz", "zstd", or "bzip2". Bounded by a total-output-bytes cap
// (3 MiB) to guard against decompression bombs — a partial result is
// returned with truncated=true if the cap is hit before the stream ends,
// rather than erroring outright (a partial decompressed blob is still a
// safe, useful, well-formed result — see helper.go's package doc comment
// for why this differs from CompressStream/archive-construction nodes,
// which error instead).
func DecompressStream(ctx context.Context, ax axiom.Context, input *gen.DecompressRequest) (*gen.CompressionResult, error) {
	if err := checkRawInputSize(input.GetData()); err != nil {
		return nil, err
	}
	codec := input.GetCodec()
	data := input.GetData()

	r, c, err := newDecompressReader(data, codec)
	if err != nil {
		return nil, fmt.Errorf("initializing %s decompressor: %w", codec, err)
	}
	if c != nil {
		defer c.Close()
	}

	out, _, truncated, err := readBoundedTruncating(r, maxTotalUncompressedBytes)
	if err != nil {
		return nil, fmt.Errorf("decompressing %s stream: %w", codec, err)
	}

	return &gen.CompressionResult{
		Data:       out,
		Codec:      codec,
		InputSize:  int64(len(data)),
		OutputSize: int64(len(out)),
		Truncated:  truncated,
	}, nil
}
