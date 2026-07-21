package nodes

import (
	"testing"

	gen "christiangeorgelucas/archive-tools/gen"
)

func TestRemoveEntry_ZipThenVerifyBySystemZip(t *testing.T) {
	zipBytes := refZip(t, map[string]string{"keep.txt": "k", "drop.txt": "d"})
	out, err := RemoveEntry(testCtxBG, testAx, &gen.RemoveEntryRequest{Data: zipBytes, Path: "drop.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, err := ListEntries(testCtxBG, testAx, &gen.ArchiveInput{Data: out.Data})
	if err != nil {
		t.Fatalf("ListEntries on result: %v", err)
	}
	if len(list.Entries) != 1 || list.Entries[0].Path != "keep.txt" {
		t.Fatalf("expected only keep.txt to remain, got %+v", list.Entries)
	}
}

func TestRemoveEntry_TarRoundTrip(t *testing.T) {
	tarBytes := refTar(t, map[string]string{"keep.txt": "k", "drop.txt": "d"})
	out, err := RemoveEntry(testCtxBG, testAx, &gen.RemoveEntryRequest{Data: tarBytes, Path: "drop.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	extracted, err := ExtractAll(testCtxBG, testAx, &gen.ArchiveInput{Data: out.Data})
	if err != nil {
		t.Fatalf("ExtractAll on result: %v", err)
	}
	if len(extracted.Entries) != 1 || extracted.Entries[0].Path != "keep.txt" || string(extracted.Entries[0].Data) != "k" {
		t.Fatalf("unexpected result: %+v", extracted.Entries)
	}
}

func TestRemoveEntry_NotFound(t *testing.T) {
	zipBytes := refZip(t, map[string]string{"keep.txt": "k"})
	_, err := RemoveEntry(testCtxBG, testAx, &gen.RemoveEntryRequest{Data: zipBytes, Path: "nope.txt"})
	if err == nil {
		t.Fatal("expected an error for a missing path, got nil")
	}
}
