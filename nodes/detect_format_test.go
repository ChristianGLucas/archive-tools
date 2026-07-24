package nodes

import (
	"testing"

	gen "christiangeorgelucas/archive-tools/gen"
)

func TestDetectFormat_TarGz(t *testing.T) {
	tarBytes := refTar(t, map[string]string{"a.txt": "hello"})
	gz := cliGzip(t, tarBytes)
	out, err := DetectFormat(testCtxBG, testAx, &gen.ArchiveInput{Data: gz})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Recognized || out.ContainerFormat != "tar" || out.Compression != "gzip" {
		t.Fatalf("got %+v, want recognized tar+gzip", out)
	}
}

func TestDetectFormat_Zip(t *testing.T) {
	zipBytes := refZip(t, map[string]string{"a.txt": "hello"})
	out, err := DetectFormat(testCtxBG, testAx, &gen.ArchiveInput{Data: zipBytes})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Recognized || out.ContainerFormat != "zip" || out.Compression != "none" {
		t.Fatalf("got %+v, want recognized zip/none", out)
	}
}

func TestDetectFormat_BareXzNoTar(t *testing.T) {
	xzBytes := cliXz(t, []byte("just plain text, not a tar"))
	out, err := DetectFormat(testCtxBG, testAx, &gen.ArchiveInput{Data: xzBytes})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Recognized || out.ContainerFormat != "" || out.Compression != "xz" {
		t.Fatalf("got %+v, want recognized bare xz with no container", out)
	}
}

func TestDetectFormat_Unrecognized(t *testing.T) {
	out, err := DetectFormat(testCtxBG, testAx, &gen.ArchiveInput{Data: []byte("not an archive at all")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Recognized {
		t.Fatalf("got recognized=true for garbage input: %+v", out)
	}
}

func TestDetectFormat_Empty(t *testing.T) {
	out, err := DetectFormat(testCtxBG, testAx, &gen.ArchiveInput{Data: nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Recognized {
		t.Fatalf("got recognized=true for empty input: %+v", out)
	}
}
