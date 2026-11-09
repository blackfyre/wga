# response-compression Specification

## Purpose

TBD - created by archiving change compress-text-responses. Update Purpose after archive.

## Requirements

### Requirement: Text responses are compressed at the origin

The application SHALL compress a response body of its own routes with gzip or zstd when the request's `Accept-Encoding` allows one of them, either by naming it or through a `*` entry with a non-zero quality, the response has a text media type (`text/*`, `application/json`, `application/javascript`, `application/xml`, `application/manifest+json`, any `+json` or `+xml` type, or `image/svg+xml`), the body is at least 1 KiB, and the response sets no `Content-Encoding` or `Content-Range`. A compressed response SHALL carry `Content-Encoding` naming the encoding and `Vary: Accept-Encoding`, and SHALL NOT carry `Content-Length` from the uncompressed body. The PocketBase administration dashboard under `/_/` is not an application route: it keeps PocketBase's built-in compression and is outside this capability.

#### Scenario: Page document is compressed

- **WHEN** a client that sends `Accept-Encoding: gzip` requests a page whose HTML body is at least 1 KiB
- **THEN** the response carries `Content-Encoding: gzip` and `Vary: Accept-Encoding`
- **AND** the decompressed body equals the uncompressed response

#### Scenario: Client without compression support

- **WHEN** a client sends no `Accept-Encoding` header
- **THEN** the response body is sent uncompressed and carries no `Content-Encoding`

#### Scenario: Small response stays uncompressed

- **WHEN** a response body is smaller than 1 KiB, such as the `GET /health` body `ok`
- **THEN** the response is sent uncompressed

### Requirement: Compressed media and partial content are not recompressed

The application SHALL NOT compress responses whose media type is not a text type listed above, including JPEG, PNG, WebP, AVIF, GIF, audio, video, and archives, and SHALL NOT compress a partial-content response that carries `Content-Range`.

#### Scenario: Artwork image is served as stored

- **WHEN** a client that accepts gzip requests an artwork image file
- **THEN** the response carries no `Content-Encoding` and the body is the stored image bytes

#### Scenario: Range request stays byte-exact

- **WHEN** a client requests a byte range of a stored file
- **THEN** the `206 Partial Content` response is not compressed

### Requirement: Validators distinguish encodings

A compressed response that carries an `ETag` SHALL carry a tag that differs from the uncompressed representation's tag by an encoding-specific suffix. Conditional requests for generated agent content SHALL answer `304 Not Modified` when `If-None-Match` carries either the uncompressed tag or its encoding-suffixed form, and the `304` SHALL carry the tag the client revalidated.

#### Scenario: Revalidating a compressed agent document

- **WHEN** a client revalidates generated agent content with the encoding-suffixed tag it received on a compressed response
- **THEN** the response is `304 Not Modified`
- **AND** its `ETag` is the encoding-suffixed tag

#### Scenario: Wildcard encoding preference

- **WHEN** a client sends `Accept-Encoding: *` and requests an HTML page of at least 1 KiB
- **THEN** the response is compressed with gzip
