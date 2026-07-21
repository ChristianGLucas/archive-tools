package nodes

import (
	"context"

	"christiangeorgelucas/archive-tools/axiom"
	gen "christiangeorgelucas/archive-tools/gen"
)

// CreateZip builds a fresh zip archive (DEFLATE-compressed) from a list of
// in-memory {path, data, mode} entries — entirely in memory. Same
// path-safety validation and directory ("/"-suffixed path) convention as
// CreateTar.
func CreateZip(ctx context.Context, ax axiom.Context, input *gen.CreateArchiveRequest) (*gen.ArchiveResult, error) {
	entries, err := entriesFromCreateRequest(input)
	if err != nil {
		return nil, err
	}
	data, err := writeZip(entries)
	if err != nil {
		return nil, err
	}
	return archiveResult(data, len(entries)), nil
}
