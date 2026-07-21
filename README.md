# archive-tools

Composable [Axiom](https://axiom.co) node package for deterministic, stateless
inspection and manipulation of archive **container** formats (tar, zip, and
their compressed variants) and the standalone **compression codecs**
(gzip, zlib, xz, zstd, +bzip2 decompress) that wrap them.

Built for the Axiom marketplace under the `christiangeorgelucas` handle.

## What it wraps

- The Go standard library: `archive/tar`, `archive/zip`, `compress/gzip`,
  `compress/bzip2`, `compress/zlib`.
- [`github.com/klauspost/compress`](https://github.com/klauspost/compress)
  (BSD-3-Clause/MIT, zero dependencies) for zstd.
- [`github.com/ulikunitz/xz`](https://github.com/ulikunitz/xz)
  (BSD-3-Clause-style, zero dependencies) for xz.

No cgo, no native library, no filesystem or network access — every node is a
pure bytes-in/bytes-out transform over caller-supplied data.

## Nodes

| Node | What it does |
|---|---|
| `DetectFormat` | Identify container (tar/zip) + compression (gzip/bzip2/xz/zstd/none) from magic bytes |
| `ListEntries` | List every entry's metadata (path, size, mode, mtime, type, symlink target) |
| `GetArchiveSummary` | Archive-level totals without a per-entry list |
| `ReadEntry` | Read one named entry's bytes |
| `ExtractAll` | Extract every file entry into an in-memory `{path, data}` list |
| `CreateTar` | Build a fresh tar archive from in-memory entries |
| `CreateZip` | Build a fresh zip archive from in-memory entries |
| `AddEntry` | Append an entry to an existing archive |
| `RemoveEntry` | Remove an entry from an existing archive |
| `ConvertContainer` | Re-encode entries from one container format into another (zip ⇄ tar) |
| `CompressStream` | Compress raw bytes with gzip/zlib/xz/zstd |
| `DecompressStream` | Decompress a gzip/zlib/xz/zstd/bzip2 stream |

## Safety

Every archive-opening node guards against zip-slip / path traversal (an unsafe
entry path is rejected, never silently honored) and bounds total decompressed
bytes and entry count against decompression/zip bombs, checked against the raw
input as bytes are actually consumed. See `messages/messages.proto` and
`nodes/helper.go` for the full safety model.

## License

MIT — Copyright (c) 2026 Christian George Lucas.
