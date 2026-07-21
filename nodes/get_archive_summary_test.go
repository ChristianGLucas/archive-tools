package nodes

import (
	"testing"

	gen "christiangeorgelucas/archive-tools/gen"
)

func TestGetArchiveSummary_Zip(t *testing.T) {
	zipBytes := refZip(t, map[string]string{"a.txt": "hello", "b.txt": "world!"})
	out, err := GetArchiveSummary(testCtxBG, testAx, &gen.ArchiveInput{Data: zipBytes})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ContainerFormat != "zip" || out.Compression != "none" {
		t.Fatalf("got format=%s/%s, want zip/none", out.ContainerFormat, out.Compression)
	}
	if out.EntryCount != 2 {
		t.Fatalf("entry_count = %d, want 2", out.EntryCount)
	}
	if out.TotalUncompressedSize != int64(len("hello")+len("world!")) {
		t.Fatalf("total_uncompressed_size = %d, want %d", out.TotalUncompressedSize, len("hello")+len("world!"))
	}
	if out.HasDirs || out.HasSymlinks {
		t.Fatalf("unexpected has_dirs/has_symlinks: %+v", out)
	}
}

func TestGetArchiveSummary_TarGzWithDirAndSymlink(t *testing.T) {
	tarBytes := refTarWithSymlink(t, "f.txt", "12345", "d", "l", "f.txt")
	gz := cliGzip(t, tarBytes)
	out, err := GetArchiveSummary(testCtxBG, testAx, &gen.ArchiveInput{Data: gz})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ContainerFormat != "tar" || out.Compression != "gzip" {
		t.Fatalf("got format=%s/%s, want tar/gzip", out.ContainerFormat, out.Compression)
	}
	if !out.HasDirs {
		t.Fatalf("expected has_dirs=true")
	}
	if !out.HasSymlinks {
		t.Fatalf("expected has_symlinks=true")
	}
	if out.EntryCount != 3 {
		t.Fatalf("entry_count = %d, want 3", out.EntryCount)
	}
}

func TestGetArchiveSummary_MalformedInput(t *testing.T) {
	_, err := GetArchiveSummary(testCtxBG, testAx, &gen.ArchiveInput{Data: []byte{0, 1, 2, 3}})
	if err == nil {
		t.Fatal("expected an error for malformed input, got nil")
	}
}
