## Why

The visual-overhaul reference defines the image rendition that each public surface needs, and @wga has already staged the matching PocketBase-compatible derivatives in S3. WGA still declares legacy thumbnail profiles and often serves original artwork files or unsupported thumbnail requests, bypassing those prepared assets.

## What Changes

- Align the fresh PocketBase artwork and portrait FileField thumbnail declarations with the authoritative visual-overhaul surface variants.
- Add a semantic thumbnail URL contract so handlers select a finished PocketBase thumbnail URL for the surface they render.
- Replace current direct-original and legacy thumbnail use on existing public artwork, portrait, home, postcard, and catalogue surfaces with their defined rendition.
- Keep originals for intentional download and apply the visual-overhaul rule that grid thumbnails do not open the image viewer.
- Add focused schema, URL, handler, and template-facing coverage for the rendition contract.

## Capabilities

### New Capabilities
- `visual-thumbnail-delivery`: Serve the pre-staged PocketBase image derivatives selected for each authoritative public surface.

### Modified Capabilities
- `database-baseline-bootstrap`: Define the current artwork and portrait thumbnail field contract in the fresh PocketBase baseline.

## Impact

- `internal/migrations/1784808001_current_schema.go` and fresh-baseline schema tests.
- `internal/utils/url` and handlers that construct artwork or portrait image URLs.
- Public Templ image consumers and ViewerJS mounting on artwork grids.
- No S3 upload, derivative generation, record-data migration, or third-party dependency changes; @wga already supplies the required objects.
