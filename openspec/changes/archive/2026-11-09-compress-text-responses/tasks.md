## 1. Origin compression

- [x] 1.1 Wrap the PocketBase HTTP handler with `gzhttp` once at serve time, restricted to text media types with a 1 KiB minimum and an encoding-suffixed ETag.
- [x] 1.2 Accept encoding-suffixed tags in agent-content conditional requests.
- [x] 1.3 Add tests for compressed HTML, uncompressed clients, small bodies, images, range responses, and suffixed-tag revalidation.
- [x] 1.4 Verify with a running server that `/` is compressed and an artwork image is not.
