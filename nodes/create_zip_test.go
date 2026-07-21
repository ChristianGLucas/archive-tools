package nodes

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	gen "christiangeorgelucas/archive-tools/gen"
)

func TestCreateZip_ReadableBySystemUnzip(t *testing.T) {
	requireTool(t, "unzip")
	out, err := CreateZip(testCtxBG, testAx, &gen.CreateArchiveRequest{Entries: []*gen.CreateEntry{
		{Path: "a.txt", Data: []byte("hello zip"), Mode: 0o644},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dir := t.TempDir()
	zipPath := filepath.Join(dir, "out.zip")
	if err := os.WriteFile(zipPath, out.Data, 0o644); err != nil {
		t.Fatalf("write zip: %v", err)
	}
	cmd := exec.Command("unzip", "-p", zipPath, "a.txt")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("system unzip failed: %v: %s", err, stderr.String())
	}
	if stdout.String() != "hello zip" {
		t.Fatalf("system unzip -p produced %q, want %q", stdout.String(), "hello zip")
	}
}

func TestCreateZip_UnsafePathRejected(t *testing.T) {
	_, err := CreateZip(testCtxBG, testAx, &gen.CreateArchiveRequest{Entries: []*gen.CreateEntry{
		{Path: "/etc/passwd", Data: []byte("x")},
	}})
	if err == nil {
		t.Fatal("expected an error for an absolute entry path, got nil")
	}
}

func TestCreateZip_EntryCountCap(t *testing.T) {
	entries := make([]*gen.CreateEntry, maxEntries+1)
	for i := range entries {
		entries[i] = &gen.CreateEntry{Path: pad(i)}
	}
	_, err := CreateZip(testCtxBG, testAx, &gen.CreateArchiveRequest{Entries: entries})
	if err == nil {
		t.Fatal("expected an error for a request exceeding the entry-count cap, got nil")
	}
}
