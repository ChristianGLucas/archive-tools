package nodes

import (
	"bytes"
	"testing"

	gen "christiangeorgelucas/archive-tools/gen"
)

func TestDecompressStream_GzipFromSystemGzip(t *testing.T) {
	gz := cliGzip(t, samplePlaintext)
	out, err := DecompressStream(testCtxBG, testAx, &gen.DecompressRequest{Data: gz, Codec: "gzip"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(out.Data, samplePlaintext) {
		t.Fatalf("decompressed data does not match original")
	}
}

func TestDecompressStream_Bzip2FromSystemBzip2(t *testing.T) {
	bz := cliBzip2(t, samplePlaintext)
	out, err := DecompressStream(testCtxBG, testAx, &gen.DecompressRequest{Data: bz, Codec: "bzip2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(out.Data, samplePlaintext) {
		t.Fatalf("decompressed data does not match original")
	}
}

func TestDecompressStream_XzFromSystemXz(t *testing.T) {
	xz := cliXz(t, samplePlaintext)
	out, err := DecompressStream(testCtxBG, testAx, &gen.DecompressRequest{Data: xz, Codec: "xz"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(out.Data, samplePlaintext) {
		t.Fatalf("decompressed data does not match original")
	}
}

func TestDecompressStream_ZstdFromSystemZstd(t *testing.T) {
	zst := cliZstd(t, samplePlaintext)
	out, err := DecompressStream(testCtxBG, testAx, &gen.DecompressRequest{Data: zst, Codec: "zstd"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(out.Data, samplePlaintext) {
		t.Fatalf("decompressed data does not match original")
	}
}

func TestDecompressStream_CorruptInput(t *testing.T) {
	_, err := DecompressStream(testCtxBG, testAx, &gen.DecompressRequest{Data: []byte("not really gzip"), Codec: "gzip"})
	if err == nil {
		t.Fatal("expected an error for corrupt/invalid gzip input, got nil")
	}
}

func TestDecompressStream_UnknownCodec(t *testing.T) {
	_, err := DecompressStream(testCtxBG, testAx, &gen.DecompressRequest{Data: []byte("x"), Codec: "rot13"})
	if err == nil {
		t.Fatal("expected an error for an unrecognized codec, got nil")
	}
}

func TestDecompressStream_BombCapTruncates(t *testing.T) {
	orig := maxTotalUncompressedBytes
	maxTotalUncompressedBytes = 50
	defer func() { maxTotalUncompressedBytes = orig }()

	plain := make([]byte, 5000) // highly compressible (all zero)
	gz := cliGzip(t, plain)
	out, err := DecompressStream(testCtxBG, testAx, &gen.DecompressRequest{Data: gz, Codec: "gzip"})
	if err != nil {
		t.Fatalf("expected a truncated result, not an error: %v", err)
	}
	if !out.Truncated || len(out.Data) != 50 {
		t.Fatalf("expected truncated=true with exactly 50 bytes, got truncated=%v len=%d", out.Truncated, len(out.Data))
	}
}
