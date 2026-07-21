package nodes

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	gen "christiangeorgelucas/archive-tools/gen"
)

func TestCreateTar_ReadableBySystemTar(t *testing.T) {
	requireTool(t, "tar")
	out, err := CreateTar(testCtxBG, testAx, &gen.CreateArchiveRequest{Entries: []*gen.CreateEntry{
		{Path: "a.txt", Data: []byte("hello"), Mode: 0o644},
		{Path: "sub/b.txt", Data: []byte("world"), Mode: 0o600},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.EntryCount != 2 {
		t.Fatalf("entry_count = %d, want 2", out.EntryCount)
	}

	dir := t.TempDir()
	tarPath := filepath.Join(dir, "out.tar")
	if err := os.WriteFile(tarPath, out.Data, 0o644); err != nil {
		t.Fatalf("write tar: %v", err)
	}
	cmd := exec.Command("tar", "-tf", tarPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("system tar could not list our archive: %v: %s", err, stderr.String())
	}
	listing := stdout.String()
	if !bytes.Contains([]byte(listing), []byte("a.txt")) || !bytes.Contains([]byte(listing), []byte("sub/b.txt")) {
		t.Fatalf("system tar listing missing expected entries:\n%s", listing)
	}

	// And extract with the system tool to confirm content round-trips.
	extractDir := t.TempDir()
	cmd = exec.Command("tar", "-xf", tarPath, "-C", extractDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("system tar extraction failed: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(extractDir, "a.txt"))
	if err != nil || string(got) != "hello" {
		t.Fatalf("extracted a.txt = %q, err=%v", got, err)
	}
}

func TestCreateTar_UnsafePathRejected(t *testing.T) {
	_, err := CreateTar(testCtxBG, testAx, &gen.CreateArchiveRequest{Entries: []*gen.CreateEntry{
		{Path: "../escape.txt", Data: []byte("x")},
	}})
	if err == nil {
		t.Fatal("expected an error for an unsafe entry path, got nil")
	}
}

func TestCreateTar_DirectoryConvention(t *testing.T) {
	out, err := CreateTar(testCtxBG, testAx, &gen.CreateArchiveRequest{Entries: []*gen.CreateEntry{
		{Path: "adir/"},
		{Path: "adir/file.txt", Data: []byte("x")},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, err := ListEntries(testCtxBG, testAx, &gen.ArchiveInput{Data: out.Data})
	if err != nil {
		t.Fatalf("ListEntries on our own output: %v", err)
	}
	var sawDir bool
	for _, e := range list.Entries {
		if e.Path == "adir/" && e.Type == gen.EntryType_ENTRY_TYPE_DIR {
			sawDir = true
		}
	}
	if !sawDir {
		t.Fatalf("expected a directory entry for adir/, got %+v", list.Entries)
	}
}
