package nodes

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"fmt"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"

	"christiangeorgelucas/archive-tools/axiom"
	gen "christiangeorgelucas/archive-tools/gen"
)

// CompressStream compresses raw bytes with a standalone codec: "gzip",
// "zlib", "xz", or "zstd". (bzip2 has no compressor in this package's
// pure-Go, zero-cgo dependency set — Go's standard library bzip2 package
// is decode-only; see DecompressStream.) level is an optional
// codec-specific compression level (meaningful for gzip/zlib/zstd; ignored
// for xz, which has no simple integer level knob in the wrapped library);
// 0 uses the codec's default.
func CompressStream(ctx context.Context, ax axiom.Context, input *gen.CompressRequest) (*gen.CompressionResult, error) {
	if err := checkRawInputSize(input.GetData()); err != nil {
		return nil, err
	}
	data := input.GetData()
	var out bytes.Buffer

	switch input.GetCodec() {
	case "gzip":
		level := gzipZlibLevel(input.GetLevel())
		w, err := gzip.NewWriterLevel(&out, level)
		if err != nil {
			return nil, fmt.Errorf("initializing gzip writer at level %d: %w", level, err)
		}
		if _, err := w.Write(data); err != nil {
			return nil, fmt.Errorf("gzip-compressing: %w", err)
		}
		if err := w.Close(); err != nil {
			return nil, fmt.Errorf("finalizing gzip stream: %w", err)
		}
	case "zlib":
		level := gzipZlibLevel(input.GetLevel())
		w, err := zlib.NewWriterLevel(&out, level)
		if err != nil {
			return nil, fmt.Errorf("initializing zlib writer at level %d: %w", level, err)
		}
		if _, err := w.Write(data); err != nil {
			return nil, fmt.Errorf("zlib-compressing: %w", err)
		}
		if err := w.Close(); err != nil {
			return nil, fmt.Errorf("finalizing zlib stream: %w", err)
		}
	case "xz":
		w, err := xz.NewWriter(&out)
		if err != nil {
			return nil, fmt.Errorf("initializing xz writer: %w", err)
		}
		if _, err := w.Write(data); err != nil {
			return nil, fmt.Errorf("xz-compressing: %w", err)
		}
		if err := w.Close(); err != nil {
			return nil, fmt.Errorf("finalizing xz stream: %w", err)
		}
	case "zstd":
		opts := []zstd.EOption{}
		if input.GetLevel() > 0 {
			opts = append(opts, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(int(input.GetLevel()))))
		}
		w, err := zstd.NewWriter(&out, opts...)
		if err != nil {
			return nil, fmt.Errorf("initializing zstd writer: %w", err)
		}
		if _, err := w.Write(data); err != nil {
			return nil, fmt.Errorf("zstd-compressing: %w", err)
		}
		if err := w.Close(); err != nil {
			return nil, fmt.Errorf("finalizing zstd stream: %w", err)
		}
	case "bzip2":
		return nil, fmt.Errorf("bzip2 compression is not supported by this package (its zero-cgo dependency set has no bzip2 encoder) — use gzip, zlib, xz, or zstd instead")
	default:
		return nil, fmt.Errorf("unrecognized codec %q — expected one of gzip, zlib, xz, zstd", input.GetCodec())
	}

	return &gen.CompressionResult{
		Data:       out.Bytes(),
		Codec:      input.GetCodec(),
		InputSize:  int64(len(data)),
		OutputSize: int64(out.Len()),
	}, nil
}

// gzipZlibLevel maps a caller-supplied level (0 = default) into the range
// gzip/zlib accept, clamping rather than erroring on an out-of-range value
// so a slightly-wrong caller input degrades gracefully instead of failing.
func gzipZlibLevel(level int32) int {
	if level == 0 {
		return gzip.DefaultCompression
	}
	if level < gzip.BestSpeed {
		return gzip.BestSpeed
	}
	if level > gzip.BestCompression {
		return gzip.BestCompression
	}
	return int(level)
}
