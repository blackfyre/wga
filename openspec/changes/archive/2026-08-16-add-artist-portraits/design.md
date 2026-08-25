## Context

The source export contains `artists.biography_image_output_path` values such as `artists/<artist-id>/<opaque>.jpg` and a separate asset pipeline owns the corresponding image bytes. WGA currently has no artist image field, its seed reader ignores the source column, and the artist page renders a reserved but empty portrait panel. The embedded synthetic SQLite fixture predates the column and contains no portraits.

## Goals / Non-Goals

**Goals:**
- Model an optional artist portrait in PocketBase.
- Preserve the filename supplied by the source export so separately staged PocketBase assets resolve for the same artist ID.
- Render the portrait consistently on the artist page, in Open Graph metadata, and in `Person.image` JSON-LD.
- Keep image-free source fixtures and artists valid.

**Non-Goals:**
- Uploading, copying, validating, repairing, or synchronising image bytes.
- Backfilling an existing installation's artist records or storage.
- Introducing a reusable media domain, responsive image pipeline, or new storage provider.
- Using artwork images as `Person.image` when a portrait is unavailable.

## Decisions

### Use a parent-owned `portrait` PocketBase FileField

Add one optional, single-select image FileField named `portrait` to the existing `artists` collection. The current baseline schema creates it before its seed migration runs, while a new timestamped migration adds it for existing installations that have already recorded the baseline migration. It is a one-to-one biography attachment rather than independently reusable media, so a general media model adds unnecessary ownership and migration complexity. The field uses the same supported raster types as artwork images and defines a square thumbnail suitable for the portrait panel.

### Treat the source output path as a filename reference, not an upload instruction

The importer reads the optional `biography_image_output_path`, validates it as a relative path using the existing source-path safety rule, and stores its basename in `artists.portrait`. It does not read image bytes or inspect their storage path. This matches the existing external-source convention, where record IDs and filenames identify assets provisioned independently.

The reader detects whether the source column exists. A source database without it imports artists with no portrait, keeping the embedded image-free synthetic fixture usable.

### Resolve one portrait URL for visible and identity metadata

When an artist has a portrait filename, the artist renderer produces a PocketBase file URL for the `artists` record and uses it for the visible image, the page Open Graph image, and `Person.image` JSON-LD. The template uses a square crop while retaining the artist name as alternative text.

When no filename is present, the existing visual placeholder remains. Open Graph metadata retains its existing first-artwork fallback; `Person.image` is omitted because an artwork is not an identity portrait.

## Risks / Trade-offs

- [A filename is present but its separately managed asset is unavailable] → The change deliberately does not validate or repair storage; the owning asset pipeline must stage matching bytes before public use.
- [The source fixture does not contain the new column] → Detect the column and treat all portraits as absent rather than requiring a fixture replacement.
- [Portrait source images are not square] → Generate a square thumbnail and use an intentional crop in the existing square portrait panel.
- [Existing installations receive the schema but not image references] → The migration adds only an optional field; the out-of-scope alternative process remains responsible for later data population.

## Migration Plan

1. Apply the additive PocketBase migration that introduces the optional field for existing installations; fresh installations receive it from the baseline before seed data is imported.
2. Deploy the renderer and source-import compatibility changes; they remain safe while all portrait values are empty.
3. The separate asset pipeline stages image bytes and sets matching filename references for the intended dataset.

The migration is additive and rendering falls back when no value exists. Removing the field would discard image references, so rollback after population is fix-forward rather than automatic removal.

## Open Questions

None. The asset pipeline's delivery guarantees remain an external operational contract.
