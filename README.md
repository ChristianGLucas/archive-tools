# archive-tools

Composable [Axiom](https://axiomide.com) node package for deterministic, stateless
inspection and manipulation of archive **container** formats (tar, zip, and
their compressed variants) and the standalone **compression codecs**
(gzip, zlib, xz, zstd, +bzip2 decompress) that wrap them.

Built for the Axiom marketplace under the `christiangeorgelucas` handle.

## Use it from your agent or app

Every node in this package is a **live, auto-scaling API endpoint** on the
[Axiom](https://axiomide.com) marketplace — call it from an AI agent or your own
code, with nothing to self-host.

**📦 See it on the marketplace:**
https://dev.axiomide.com/marketplace/christiangeorgelucas/archive-tools@0.1.0

**Hook it up to an AI agent (MCP).** Add Axiom's hosted MCP server to any MCP
client and every node becomes a typed tool your agent can call — search the
catalog, inspect a schema, and invoke it directly.

```bash
# Claude Code
claude mcp add --transport http axiom https://api.axiomide.com/mcp \
  --header "Authorization: Bearer $AXIOM_API_KEY"
```

Claude Desktop, Cursor, or any config-based client:

```json
{
  "mcpServers": {
    "axiom": {
      "type": "http",
      "url": "https://api.axiomide.com/mcp",
      "headers": { "Authorization": "Bearer YOUR_AXIOM_API_KEY" }
    }
  }
}
```

**Call it from the CLI.**

```bash
axiom invoke christiangeorgelucas/archive-tools/DetectFormat --input '{ ... }'
```

**Call it over HTTP.**

```bash
curl -X POST https://api.axiomide.com/invocations/v1/nodes/christiangeorgelucas/archive-tools/0.1.0/DetectFormat \
  -H "Authorization: Bearer $AXIOM_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{ ... }'
```

> Input/output schema for each node is on the marketplace page above, or via
> `axiom inspect node christiangeorgelucas/archive-tools/DetectFormat`.

### Get started free

Install the CLI:

```bash
# macOS / Linux — Homebrew
brew install axiomide/tap/axiom

# macOS / Linux — install script
curl -fsSL https://raw.githubusercontent.com/AxiomIDE/axiom-releases/main/install.sh | sh
```

**Windows:** download the `windows/amd64` `.zip` from the
[releases page](https://github.com/AxiomIDE/axiom-releases/releases), unzip it,
and put `axiom.exe` on your `PATH`.

Then `axiom version` to verify, `axiom login` (GitHub or Google) to authenticate,
and create an API key under **Console → API Keys**. Docs and sign-up at
**[axiomide.com](https://axiomide.com)**.

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
