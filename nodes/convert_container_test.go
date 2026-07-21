package nodes

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	gen "christiangeorgelucas/archive-tools/gen"
)

func TestConvertContainer_ZipToTarVerifiedBySystemTar(t *testing.T) {
	requireTool(t, "tar")
	zipBytes := refZip(t, map[string]string{"a.txt": "hello", "sub/b.txt": "world"})
	out, err := ConvertContainer(testCtxBG, testAx, &gen.ConvertRequest{Data: zipBytes, TargetFormat: "tar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dir := t.TempDir()
	tarPath := filepath.Join(dir, "out.tar")
	if err := os.WriteFile(tarPath, out.Data, 0o644); err != nil {
		t.Fatalf("write tar: %v", err)
	}
	extractDir := t.TempDir()
	cmd := exec.Command("tar", "-xf", tarPath, "-C", extractDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("system tar extraction failed: %v: %s", err, stderr.String())
	}
	got, err := os.ReadFile(filepath.Join(extractDir, "a.txt"))
	if err != nil || string(got) != "hello" {
		t.Fatalf("extracted a.txt = %q, err=%v", got, err)
	}
	got2, err := os.ReadFile(filepath.Join(extractDir, "sub/b.txt"))
	if err != nil || string(got2) != "world" {
		t.Fatalf("extracted sub/b.txt = %q, err=%v", got2, err)
	}
}

func TestConvertContainer_TarToZipVerifiedBySystemUnzip(t *testing.T) {
	requireTool(t, "unzip")
	tarBytes := refTar(t, map[string]string{"a.txt": "hi there"})
	out, err := ConvertContainer(testCtxBG, testAx, &gen.ConvertRequest{Data: tarBytes, TargetFormat: "zip"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "out.zip")
	if err := os.WriteFile(zipPath, out.Data, 0o644); err != nil {
		t.Fatalf("write zip: %v", err)
	}
	cmd := exec.Command("unzip", "-p", zipPath, "a.txt")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("system unzip failed: %v", err)
	}
	if stdout.String() != "hi there" {
		t.Fatalf("got %q, want %q", stdout.String(), "hi there")
	}
}

func TestConvertContainer_InvalidTargetFormat(t *testing.T) {
	zipBytes := refZip(t, map[string]string{"a.txt": "x"})
	_, err := ConvertContainer(testCtxBG, testAx, &gen.ConvertRequest{Data: zipBytes, TargetFormat: "7z"})
	if err == nil {
		t.Fatal("expected an error for an unsupported target_format, got nil")
	}
}
