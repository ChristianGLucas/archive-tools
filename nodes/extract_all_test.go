package nodes

import (
	"testing"

	gen "christiangeorgelucas/archive-tools/gen"
)

func TestExtractAll_ZipAgainstSystemZip(t *testing.T) {
	files := map[string]string{"one.txt": "111", "sub/two.txt": "22222", "empty.txt": ""}
	zipBytes := refZip(t, files)
	out, err := ExtractAll(testCtxBG, testAx, &gen.ArchiveInput{Data: zipBytes})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := map[string]string{}
	for _, e := range out.Entries {
		got[e.Path] = string(e.Data)
	}
	for path, want := range files {
		if got[path] != want {
			t.Fatalf("entry %q = %q, want %q", path, got[path], want)
		}
	}
	if len(out.SkippedUnsafePaths) != 0 {
		t.Fatalf("unexpected skipped paths: %v", out.SkippedUnsafePaths)
	}
}

func TestExtractAll_UnsafeEntrySkippedAndReported(t *testing.T) {
	data, err := writeTar([]rawEntry{
		{path: "good.txt", typ: gen.EntryType_ENTRY_TYPE_FILE, data: []byte("ok")},
		{path: "../evil.txt", typ: gen.EntryType_ENTRY_TYPE_FILE, data: []byte("bad")},
	})
	if err != nil {
		t.Fatalf("writeTar: %v", err)
	}
	out, err := ExtractAll(testCtxBG, testAx, &gen.ArchiveInput{Data: data})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Entries) != 1 || out.Entries[0].Path != "good.txt" {
		t.Fatalf("expected only good.txt extracted, got %+v", out.Entries)
	}
	if len(out.SkippedUnsafePaths) != 1 || out.SkippedUnsafePaths[0] != "../evil.txt" {
		t.Fatalf("expected ../evil.txt reported as skipped, got %v", out.SkippedUnsafePaths)
	}
}

func TestExtractAll_DirsAndSymlinksOmittedFromData(t *testing.T) {
	tarBytes := refTarWithSymlink(t, "f.txt", "content", "adir", "alink", "f.txt")
	out, err := ExtractAll(testCtxBG, testAx, &gen.ArchiveInput{Data: tarBytes})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Entries) != 1 || out.Entries[0].Path != "f.txt" {
		t.Fatalf("expected only the regular file entry, got %+v", out.Entries)
	}
}

func TestExtractAll_DecompressionBombCapTruncates(t *testing.T) {
	// Shrink the budget for this test so we can prove the cap fires without
	// allocating a real half-gigabyte fixture — a small entry (1000 bytes)
	// against a tiny budget (100 bytes) exercises exactly the same code
	// path a real gzip-bomb (small compressed input, huge decompressed
	// output) would hit.
	orig := maxTotalUncompressedBytes
	maxTotalUncompressedBytes = 100
	defer func() { maxTotalUncompressedBytes = orig }()

	data := make([]byte, 1000)
	tarBytes, err := writeTar([]rawEntry{{path: "bomb.bin", typ: gen.EntryType_ENTRY_TYPE_FILE, data: data, mode: 0o644}})
	if err != nil {
		t.Fatalf("writeTar: %v", err)
	}
	out, err := ExtractAll(testCtxBG, testAx, &gen.ArchiveInput{Data: tarBytes})
	if err != nil {
		t.Fatalf("expected a truncated result, not an error: %v", err)
	}
	if !out.Truncated {
		t.Fatalf("expected truncated=true for an archive whose entry exceeds the size cap")
	}
	if len(out.Entries) != 1 || len(out.Entries[0].Data) != 100 {
		t.Fatalf("expected exactly 100 bytes of partial data, got entries=%+v", out.Entries)
	}
}

func TestExtractAll_MalformedInput(t *testing.T) {
	_, err := ExtractAll(testCtxBG, testAx, &gen.ArchiveInput{Data: []byte("garbage")})
	if err == nil {
		t.Fatal("expected an error for malformed input, got nil")
	}
}
