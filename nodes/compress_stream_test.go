package nodes

import (
	"bytes"
	"testing"

	gen "christiangeorgelucas/archive-tools/gen"
)

var samplePlaintext = bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. "), 200)

func TestCompressStream_GzipDecodableBySystemGzip(t *testing.T) {
	out, err := CompressStream(testCtxBG, testAx, &gen.CompressRequest{Data: samplePlaintext, Codec: "gzip"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := cliGunzip(t, out.Data)
	if !bytes.Equal(got, samplePlaintext) {
		t.Fatalf("system gzip -d could not recover our compressed output correctly")
	}
	if out.OutputSize >= out.InputSize {
		t.Fatalf("expected compression to shrink highly-repetitive input: in=%d out=%d", out.InputSize, out.OutputSize)
	}
}

func TestCompressStream_ZlibRoundTrip(t *testing.T) {
	out, err := CompressStream(testCtxBG, testAx, &gen.CompressRequest{Data: samplePlaintext, Codec: "zlib"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	back, err := DecompressStream(testCtxBG, testAx, &gen.DecompressRequest{Data: out.Data, Codec: "zlib"})
	if err != nil {
		t.Fatalf("unexpected error decompressing: %v", err)
	}
	if !bytes.Equal(back.Data, samplePlaintext) {
		t.Fatalf("zlib round trip mismatch")
	}
}

func TestCompressStream_XzDecodableBySystemXz(t *testing.T) {
	out, err := CompressStream(testCtxBG, testAx, &gen.CompressRequest{Data: samplePlaintext, Codec: "xz"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := cliUnxz(t, out.Data)
	if !bytes.Equal(got, samplePlaintext) {
		t.Fatalf("system xz -d could not recover our compressed output correctly")
	}
}

func TestCompressStream_ZstdDecodableBySystemZstd(t *testing.T) {
	out, err := CompressStream(testCtxBG, testAx, &gen.CompressRequest{Data: samplePlaintext, Codec: "zstd"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := cliUnzstd(t, out.Data)
	if !bytes.Equal(got, samplePlaintext) {
		t.Fatalf("system zstd -d could not recover our compressed output correctly")
	}
}

func TestCompressStream_Bzip2Unsupported(t *testing.T) {
	_, err := CompressStream(testCtxBG, testAx, &gen.CompressRequest{Data: samplePlaintext, Codec: "bzip2"})
	if err == nil {
		t.Fatal("expected a structured error for bzip2 compression (unsupported), got nil")
	}
}

func TestCompressStream_UnknownCodec(t *testing.T) {
	_, err := CompressStream(testCtxBG, testAx, &gen.CompressRequest{Data: samplePlaintext, Codec: "rot13"})
	if err == nil {
		t.Fatal("expected an error for an unrecognized codec, got nil")
	}
}

func TestCompressStream_RawInputCap(t *testing.T) {
	big := make([]byte, maxRawInputBytes+1)
	_, err := CompressStream(testCtxBG, testAx, &gen.CompressRequest{Data: big, Codec: "gzip"})
	if err == nil {
		t.Fatal("expected an error for input exceeding the raw-input cap, got nil")
	}
}
