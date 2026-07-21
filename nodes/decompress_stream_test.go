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

// TestDecompressStream_RealisticSizeUnderRealDefaultCap is a regression
// test for the CRITICAL finding that this package's caps (formerly 512
// MiB/256 MiB) were never actually reachable: the Axiom node transport
// caps a single message at ~4 MiB, so a completely ordinary multi-hundred-
// KB decompressed payload used to fail with an opaque transport-level
// error despite being far under the package's own advertised limit. This
// test exercises the REAL, un-overridden default maxTotalUncompressedBytes
// (3 MiB) with a realistic ~1.5 MiB payload and confirms it succeeds
// whole, untruncated — proving the shipped cap is actually usable, not
// just theoretical.
func TestDecompressStream_RealisticSizeUnderRealDefaultCap(t *testing.T) {
	plain := bytes.Repeat([]byte("some realistic file content, not all zeros. "), 34000) // ~1.5 MiB
	gz := cliGzip(t, plain)
	out, err := DecompressStream(testCtxBG, testAx, &gen.DecompressRequest{Data: gz, Codec: "gzip"})
	if err != nil {
		t.Fatalf("unexpected error decompressing a realistic ~1.5MiB payload: %v", err)
	}
	if out.Truncated {
		t.Fatalf("unexpectedly truncated a payload well under the real default cap")
	}
	if !bytes.Equal(out.Data, plain) {
		t.Fatalf("decompressed data does not match original")
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
