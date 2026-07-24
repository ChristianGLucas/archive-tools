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

// TestDecompressStream_RealisticSize proves a completely ordinary
// multi-hundred-KB decompressed payload round-trips whole — this package
// no longer imposes its own decompressed-output-size ceiling; the
// platform's ingress/transport and node sandbox are what actually bound
// payload size and contain a runaway decompression.
func TestDecompressStream_RealisticSize(t *testing.T) {
	plain := bytes.Repeat([]byte("some realistic file content, not all zeros. "), 34000) // ~1.5 MiB
	gz := cliGzip(t, plain)
	out, err := DecompressStream(testCtxBG, testAx, &gen.DecompressRequest{Data: gz, Codec: "gzip"})
	if err != nil {
		t.Fatalf("unexpected error decompressing a realistic ~1.5MiB payload: %v", err)
	}
	if !bytes.Equal(out.Data, plain) {
		t.Fatalf("decompressed data does not match original")
	}
}
