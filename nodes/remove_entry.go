package nodes

import (
	"context"
	"fmt"

	"christiangeorgelucas/archive-tools/axiom"
	gen "christiangeorgelucas/archive-tools/gen"
)

// RemoveEntry returns a copy of an existing (uncompressed) tar or zip
// archive with every entry matching the exact given path removed. Errors
// if no entry matches — a no-op copy is never silently returned for a
// missing path.
func RemoveEntry(ctx context.Context, ax axiom.Context, input *gen.RemoveEntryRequest) (*gen.ArchiveResult, error) {
	if err := checkRawInputSize(input.GetData()); err != nil {
		return nil, err
	}
	target := input.GetPath()

	oc, err := openContainer(input.GetData(), input.GetFormatHint())
	if err != nil {
		return nil, err
	}
	entries, _, truncated, err := walkData(oc, true)
	if err != nil {
		return nil, err
	}
	if truncated {
		return nil, fmt.Errorf("source archive exceeds this package's entry-count/size caps — refusing to modify a partially-read archive")
	}

	kept := entries[:0:0]
	removed := 0
	for _, e := range entries {
		if e.path == target {
			removed++
			continue
		}
		kept = append(kept, e)
	}
	if removed == 0 {
		return nil, fmt.Errorf("no entry with path %q found in the archive", target)
	}

	data, err := writeContainer(oc.kind, kept)
	if err != nil {
		return nil, err
	}
	return archiveResult(data, len(kept)), nil
}
