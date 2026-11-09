## Why

WGA sends every dynamic response uncompressed. The edge compresses for visitors, but the hop from Railway to the edge is billed on uncompressed bytes: `/` leaves the origin at 57,794 bytes and reaches the visitor as 11,293 bytes with Brotli. Railway billed 1,420 GB of egress in the period where Cloudflare served 766 GB, which makes uncompressed origin responses the largest avoidable cost.

## What Changes

- Compress text responses (HTML, HTMX fragments, CSS, JavaScript, JSON, XML, Markdown, SVG, plain text) at the origin when the client accepts gzip or zstd.
- Leave already-compressed media (JPEG, PNG, WebP, AVIF, GIF, audio, video, archives), partial-content responses, and small responses untouched.
- Give compressed responses an encoding-specific ETag, and keep conditional requests for generated agent content answering `304 Not Modified` with either form of the tag.

## Capabilities

### New Capabilities

- `response-compression`: Origin compression of text responses, with the media, range, and validator rules that keep it safe.

### Modified Capabilities

- None.

## Impact

- `internal/handlers` server registration wraps the PocketBase HTTP handler once, outside every route middleware.
- `internal/handlers/static` conditional-request matching accepts the encoding-suffixed ETag.
- Uses `github.com/klauspost/compress/gzhttp` from the existing direct dependency; no new module.
