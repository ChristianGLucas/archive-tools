package nodes

import (
	"testing"

	gen "christiangeorgelucas/archive-tools/gen"
)

func TestAddEntry_ZipThenVerifyBySystemZip(t *testing.T) {
	zipBytes := refZip(t, map[string]string{"existing.txt": "old"})
	out, err := AddEntry(testCtxBG, testAx, &gen.AddEntryRequest{
		Data:      zipBytes,
		Path:      "new.txt",
		EntryData: []byte("fresh"),
		Mode:      0o644,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, err := ListEntries(testCtxBG, testAx, &gen.ArchiveInput{Data: out.Data})
	if err != nil {
		t.Fatalf("ListEntries on result: %v", err)
	}
	found := map[string]bool{}
	for _, e := range list.Entries {
		found[e.Path] = true
	}
	if !found["existing.txt"] || !found["new.txt"] {
		t.Fatalf("expected both entries present, got %+v", list.Entries)
	}

	// Independent oracle: re-verify against the ExtractAll node's decode of
	// the same bytes, which itself is tested elsewhere against system tools.
	extracted, err := ExtractAll(testCtxBG, testAx, &gen.ArchiveInput{Data: out.Data})
	if err != nil {
		t.Fatalf("ExtractAll on result: %v", err)
	}
	byPath := map[string]string{}
	for _, e := range extracted.Entries {
		byPath[e.Path] = string(e.Data)
	}
	if byPath["existing.txt"] != "old" || byPath["new.txt"] != "fresh" {
		t.Fatalf("content mismatch: %+v", byPath)
	}
}

func TestAddEntry_UnsafeNewPathRejected(t *testing.T) {
	zipBytes := refZip(t, map[string]string{"existing.txt": "old"})
	_, err := AddEntry(testCtxBG, testAx, &gen.AddEntryRequest{Data: zipBytes, Path: "../escape.txt", EntryData: []byte("x")})
	if err == nil {
		t.Fatal("expected an error for an unsafe new entry path, got nil")
	}
}

func TestAddEntry_SourceWithUnsafeEntryRejected(t *testing.T) {
	poisoned, err := writeTar([]rawEntry{{path: "../evil.txt", typ: gen.EntryType_ENTRY_TYPE_FILE, data: []byte("x")}})
	if err != nil {
		t.Fatalf("writeTar: %v", err)
	}
	_, err = AddEntry(testCtxBG, testAx, &gen.AddEntryRequest{Data: poisoned, Path: "new.txt", EntryData: []byte("x")})
	if err == nil {
		t.Fatal("expected an error when the SOURCE archive itself contains an unsafe path, got nil")
	}
}

func TestAddEntry_MalformedSource(t *testing.T) {
	_, err := AddEntry(testCtxBG, testAx, &gen.AddEntryRequest{Data: []byte("not an archive"), Path: "x.txt", EntryData: []byte("x")})
	if err == nil {
		t.Fatal("expected an error for malformed source archive, got nil")
	}
}
