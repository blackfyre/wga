## Context

WGA serves PocketBase files from S3 through `/api/files`, but its fresh baseline declares only legacy thumbnail profiles and handlers mix unsupported thumbnail requests with original-file URLs. The authoritative visual-overhaul reference assigns a proportional rendition to every image surface. The adjacent @wga source pipeline has already generated and uploaded PocketBase-compatible `Wx0` derivative objects, so WGA must only expose and select them.

Deployments reset both the PocketBase database and migration ledger. The current schema baseline is therefore the sole schema migration point; no forward migration or record preservation is required.

## Goals / Non-Goals

**Goals:**
- Declare the visual-overhaul artwork thumbnail ladder in the fresh `artworks.image` field.
- Declare the portrait variants used by authoritative live surfaces in the fresh `artists.portrait` field.
- Give handlers a small semantic rendition contract and pass complete thumbnail URLs to templates.
- Keep card and row renditions out of the image viewer, while allowing an artwork record to display and zoom at separately selected sizes.
- Preserve existing missing-image fallbacks and intentional access to an original file.

**Non-Goals:**
- Generate, upload, delete, or verify S3 derivative objects.
- Preserve data or apply an in-place schema update to an existing PocketBase data directory.
- Port unimplemented visual-overhaul routes, editorial tour models, or Dual Mode size controls.
- Introduce responsive `srcset` ladders, a media domain, or a runtime image-resizing service.

## Decisions

### Treat the visual-overhaul surface table as the application contract

The visual-overhaul reference, rather than the source pipeline's schema, determines which WGA surfaces request each derivative. The existing source pipeline remains the authority for the compatible object layout and confirms that `Wx0` preserves aspect ratio.

The fresh `artworks.image` field exposes all thirteen visual artwork variants: `120x0`, `200x0`, `400x0`, `500x0`, `600x0`, `700x0`, `800x0`, `900x0`, `1000x0`, `1100x0`, `1400x0`, `1600x0`, and `2000x0`. The fresh `artists.portrait` field exposes `500x0` for portrait cards and `600x0` for the artist record. @wga's additional staged `400x0` portrait object is harmless but has no authoritative WGA surface and is not declared.

### Select semantic renditions in handlers, not templates

The URL utility will own named thumbnail variants and build PocketBase `?thumb=Wx0` URLs. A handler chooses the rendition for the surface it is populating; templates receive a finished URL and never append thumbnail query parameters. This prevents legacy literals from drifting from the field schema.

| Surface | Rendition |
| --- | --- |
| Artwork cards: artist works, random inspiration, home recent, catalogue grid | `500x0` |
| Catalogue list row | `200x0` |
| Related artwork card | `400x0` |
| Work of the day | `900x0` |
| Postcard preview and received postcard | `700x0` |
| Artwork record display | `1400x0` |
| Artwork record viewer zoom | `2000x0` |
| Artist index portrait | `500x0` |
| Artist record portrait | `600x0` |

The existing image DTO may carry the selected rendered URL where one rendition is sufficient. Artwork record rendering requires separate display and zoom/source URLs so a visitor never opens a card or plate rendition as a zoom image. The unscaled original remains reserved for an explicit download or metadata use, not a normal rendered surface.

### Align ViewerJS mounting with the reference interaction

Artwork grids are navigational cards and will not be mounted as ViewerJS galleries. The artwork record remains the deliberate viewer entry point and receives its zoom URL explicitly. This avoids enlarging a card rendition and restores the reference distinction between browsing and studying a work.

### Edit the baseline schema only

The current schema migration creates both image fields before the historical portrait migration runs; that later migration already exits when the field exists. Updating the baseline field declarations therefore produces the final schema on every deployment reset without changing historical migration files or adding a migration that is never needed operationally.

## Risks / Trade-offs

- [A handler requests a derivative absent from the field declaration] → Centralise variant names in the URL utility and assert the exact fresh-field lists in schema tests.
- [A source original is smaller than its named rendition] → @wga intentionally omits an upscaled derivative; PocketBase falls back to the original, which is no larger than the requested rendition.
- [A normal page still renders an original through an overlooked image field] → Inventory direct `GenerateFileUrl` consumers and cover each mapped public surface with focused tests.
- [ViewerJS remains mounted on a grid] → Remove grid mounts and test that only the artwork record supplies the deliberate viewer source.
- [A developer starts against a retained data directory] → The change relies on the documented full database-and-ledger reset; local verification must reset the active data directory too.

## Migration Plan

1. Update the baseline FileField thumbnail declarations and their fresh-schema assertions.
2. Add rendition selection and migrate the existing public image consumers to it.
3. Update viewer wiring and generate Templ output where sources change.
4. Rebuild a fresh deployment database and verify representative file responses reference the pre-staged S3 derivatives.

Rollback is a source rollback followed by the same full database reset. Existing S3 derivative objects are derived, public assets and need no rollback action.

## Open Questions

None.
