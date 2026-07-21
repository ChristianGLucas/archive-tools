package nodes

import (
	"testing"

	gen "christiangeorgelucas/archive-tools/gen"
)

func TestReadEntry_ZipAgainstSystemZip(t *testing.T) {
	zipBytes := refZip(t, map[string]string{"one.txt": "the quick brown fox", "sub/two.txt": "jumps"})
	out, err := ReadEntry(testCtxBG, testAx, &gen.ReadEntryRequest{Data: zipBytes, Path: "sub/two.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out.Data) != "jumps" {
		t.Fatalf("got %q, want %q", out.Data, "jumps")
	}
	if out.Size != 5 {
		t.Fatalf("size = %d, want 5", out.Size)
	}
}

func TestReadEntry_TarAgainstSystemTar(t *testing.T) {
	tarBytes := refTar(t, map[string]string{"a.bin": "\x00\x01binary\xff"})
	out, err := ReadEntry(testCtxBG, testAx, &gen.ReadEntryRequest{Data: tarBytes, Path: "a.bin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out.Data) != "\x00\x01binary\xff" {
		t.Fatalf("got %q", out.Data)
	}
}

func TestReadEntry_NotFound(t *testing.T) {
	zipBytes := refZip(t, map[string]string{"one.txt": "x"})
	_, err := ReadEntry(testCtxBG, testAx, &gen.ReadEntryRequest{Data: zipBytes, Path: "does-not-exist.txt"})
	if err == nil {
		t.Fatal("expected an error for a missing entry, got nil")
	}
}

func TestReadEntry_UnsafePathRejectedImmediately(t *testing.T) {
	zipBytes := refZip(t, map[string]string{"one.txt": "x"})
	_, err := ReadEntry(testCtxBG, testAx, &gen.ReadEntryRequest{Data: zipBytes, Path: "../../etc/passwd"})
	if err == nil {
		t.Fatal("expected an error for an unsafe path, got nil")
	}
}

func TestReadEntry_DirectoryPathErrors(t *testing.T) {
	tarBytes := refTarWithSymlink(t, "f.txt", "x", "adir", "alink", "f.txt")
	// "adir/" (or "adir") names a directory, not a file.
	_, err1 := ReadEntry(testCtxBG, testAx, &gen.ReadEntryRequest{Data: tarBytes, Path: "adir/"})
	_, err2 := ReadEntry(testCtxBG, testAx, &gen.ReadEntryRequest{Data: tarBytes, Path: "adir"})
	if err1 == nil && err2 == nil {
		t.Fatal("expected an error reading a directory entry as a file, got nil for both name forms")
	}
}
