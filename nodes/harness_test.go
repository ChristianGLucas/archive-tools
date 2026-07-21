package nodes

// Shared test fixtures and helpers. Several tests here deliberately build
// their reference archives/streams using the SYSTEM tar/zip/gzip/bzip2/xz/
// zstd command-line tools (the reference C implementations) rather than
// this package's own encode nodes — that is what makes them an
// independent-oracle test of our DECODE logic, and separately, feeding our
// own encode-node output back through the system tools' decoders is an
// independent-oracle test of our ENCODE logic. A round trip through only
// our own code would merely show self-consistency, not correctness against
// an external reference.

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"

	"christiangeorgelucas/archive-tools/axiom"
)

// testCtxBG/testAx are the fixed context/AxiomContext values every node
// test passes — none of this package's nodes call any ax method (they are
// pure bytes-in/bytes-out transforms), so the nil interface value is a
// legitimate stand-in, not a shortcut around real behavior.
var testCtxBG = context.Background()

var testAx axiom.Context // nil; never dereferenced by any node in this package

// requireTool skips the test if name is not on PATH — these tests are an
// independent-oracle CHECK layered on top of the golden/unit tests, not
// the only coverage, so an environment missing a system tool degrades to
// "skipped", never a false failure of the code under test.
func requireTool(t *testing.T, name string) string {
	t.Helper()
	p, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("system tool %q not on PATH; skipping independent-oracle test", name)
	}
	return p
}

// refTar builds a tar archive using the SYSTEM `tar` binary over a real
// temp-directory tree, returning its bytes. files maps archive-relative
// path -> content.
func refTar(t *testing.T, files map[string]string) []byte {
	t.Helper()
	requireTool(t, "tar")
	dir := t.TempDir()
	var names []string
	for name, content := range files {
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture file: %v", err)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	args := append([]string{"-cf", "-", "-C", dir}, names...)
	cmd := exec.Command("tar", args...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("system tar failed: %v: %s", err, stderr.String())
	}
	return out.Bytes()
}

// refZip builds a zip archive using the SYSTEM `zip` binary, returning its
// bytes.
func refZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	requireTool(t, "zip")
	dir := t.TempDir()
	var names []string
	for name, content := range files {
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture file: %v", err)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	outPath := filepath.Join(t.TempDir(), "out.zip")
	args := append([]string{"-q", outPath}, names...)
	cmd := exec.Command("zip", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("system zip failed: %v: %s", err, stderr.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read produced zip: %v", err)
	}
	return data
}

// refTarWithSymlink builds, via the SYSTEM `tar` binary, an archive
// containing one regular file, one real directory, and one real symlink —
// used to check this package's directory/symlink metadata handling
// against the reference tar implementation.
func refTarWithSymlink(t *testing.T, filePath, fileContent, dirPath, linkPath, linkTarget string) []byte {
	t.Helper()
	requireTool(t, "tar")
	dir := t.TempDir()
	full := filepath.Join(dir, filePath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(fileContent), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, dirPath), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	linkFull := filepath.Join(dir, linkPath)
	if err := os.MkdirAll(filepath.Dir(linkFull), 0o755); err != nil {
		t.Fatalf("mkdir for symlink: %v", err)
	}
	if err := os.Symlink(linkTarget, linkFull); err != nil {
		t.Fatalf("create fixture symlink: %v", err)
	}
	cmd := exec.Command("tar", "-cf", "-", "-C", dir, filePath, dirPath, linkPath)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("system tar failed: %v: %s", err, stderr.String())
	}
	return out.Bytes()
}

// cliCompress shells out to `tool -c` (reading stdin, writing stdout) to
// compress data with the system's reference implementation of a codec.
func cliCompress(t *testing.T, tool string, args []string, data []byte) []byte {
	t.Helper()
	requireTool(t, tool)
	cmd := exec.Command(tool, args...)
	cmd.Stdin = bytes.NewReader(data)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("system %s failed: %v: %s", tool, err, stderr.String())
	}
	return out.Bytes()
}

func cliGzip(t *testing.T, data []byte) []byte  { return cliCompress(t, "gzip", []string{"-c"}, data) }
func cliBzip2(t *testing.T, data []byte) []byte { return cliCompress(t, "bzip2", []string{"-c"}, data) }
func cliXz(t *testing.T, data []byte) []byte    { return cliCompress(t, "xz", []string{"-c"}, data) }
func cliZstd(t *testing.T, data []byte) []byte  { return cliCompress(t, "zstd", []string{"-c", "-q"}, data) }

func cliGunzip(t *testing.T, data []byte) []byte { return cliCompress(t, "gzip", []string{"-dc"}, data) }
func cliBunzip2(t *testing.T, data []byte) []byte {
	return cliCompress(t, "bzip2", []string{"-dc"}, data)
}
func cliUnxz(t *testing.T, data []byte) []byte  { return cliCompress(t, "xz", []string{"-dc"}, data) }
func cliUnzstd(t *testing.T, data []byte) []byte {
	return cliCompress(t, "zstd", []string{"-dc", "-q"}, data)
}
