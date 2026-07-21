package nodes

import (
	"context"
	"fmt"

	"christiangeorgelucas/archive-tools/axiom"
	gen "christiangeorgelucas/archive-tools/gen"
)

// ConvertContainer re-encodes an archive's entries from one container
// format into another (zip -> tar or tar -> zip; auto-detecting a
// compressed outer wrap on the source), preserving every entry's path,
// bytes, mode, and type. The result is always an uncompressed container in
// the target format — wrap it with CompressStream afterward to reproduce a
// compressed variant like .tar.gz.
func ConvertContainer(ctx context.Context, ax axiom.Context, input *gen.ConvertRequest) (*gen.ArchiveResult, error) {
	if err := checkRawInputSize(input.GetData()); err != nil {
		return nil, err
	}
	target := input.GetTargetFormat()
	if target != "tar" && target != "zip" {
		return nil, fmt.Errorf("unrecognized target_format %q — expected \"tar\" or \"zip\"", target)
	}

	oc, err := openContainer(input.GetData(), input.GetFormatHint())
	if err != nil {
		return nil, err
	}
	entries, _, truncated, err := walkData(oc, true)
	if err != nil {
		return nil, err
	}
	if truncated {
		return nil, fmt.Errorf("source archive exceeds this package's entry-count/size caps — refusing to convert a partially-read archive")
	}

	data, err := writeContainer(target, entries)
	if err != nil {
		return nil, err
	}
	return archiveResult(data, len(entries)), nil
}
