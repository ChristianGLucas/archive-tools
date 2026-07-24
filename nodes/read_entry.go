package nodes

import (
	"context"
	"fmt"

	"christiangeorgelucas/archive-tools/axiom"
	gen "christiangeorgelucas/archive-tools/gen"
)

// ReadEntry reads one named entry's full uncompressed bytes out of a tar
// or zip archive (auto-detecting a compressed outer wrap). Errors if the
// path does not exist, names a non-file entry, or is itself unsafe
// (absolute or containing "..") — an unsafe request is never partially
// honored.
func ReadEntry(ctx context.Context, ax axiom.Context, input *gen.ReadEntryRequest) (*gen.EntryData, error) {
	target := input.GetPath()
	if !sanitizePath(target) {
		return nil, fmt.Errorf("path %q is unsafe (absolute or escapes via \"..\") — refusing to read it", target)
	}
	oc, err := openContainer(input.GetData(), input.GetFormatHint())
	if err != nil {
		return nil, err
	}
	re, err := findEntry(oc, target)
	if err != nil {
		return nil, err
	}
	return &gen.EntryData{Path: re.path, Data: re.data, Size: re.size}, nil
}
