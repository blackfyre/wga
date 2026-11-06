## 1. Artist portrait data contract

- [x] 1.1 Add an additive PocketBase migration that defines the optional single-file `artists.portrait` image field and square thumbnail profile.
- [x] 1.2 Extend the migration schema test to verify the `artists.portrait` field contract on a fresh database.
- [x] 1.3 Extend the seed-source artist model and loader to read `biography_image_output_path` when available, safely reduce it to a filename, and remain compatible with fixtures that lack the column.
- [x] 1.4 Set each imported artist's `portrait` filename without reading or uploading image bytes, and cover both portrait-bearing and image-free source schemas with focused seed tests.

## 2. Public artist presentation

- [x] 2.1 Add the resolved portrait URL to the artist view data and use the artist FileField URL when a filename is present.
- [x] 2.2 Replace the artist-page portrait placeholder with the supplied square-cropped image and preserve the labelled fallback when no URL is available.
- [x] 2.3 Update artist Open Graph metadata to prefer the portrait while preserving the existing first-artwork fallback.
- [x] 2.4 Populate `Person.image` JSON-LD from the portrait URL only, and cover present and absent portraits with focused renderer and JSON-LD tests.

## 3. Generated assets and verification

- [x] 3.1 Run `templ generate` after changing the artist template and confirm only generated ignored files are produced.
- [x] 3.2 Run focused seed, migration, artist-handler, and JSON-LD tests; run `go test ./...` if the focused checks pass.
- [x] 3.3 Manually verify an artist with a separately staged portrait shows the image and matching page metadata, while an artist without one shows the fallback.

  Local verification (2026-11-06): `playwright-tests/artist-portraits.spec.ts`
  passed both scenarios on port 8102. Synthetic Artist 02 had its portrait
  filename set in an isolated database and JPEG bytes staged separately in
  PocketBase storage; the browser decoded the image and verified its alternative
  text and matching Open Graph / `Person.image` URLs. Synthetic Artist 01 retained
  the labelled fallback, omitted `Person.image`, and used artwork for Open Graph.
  The staged-image test accepts `WGA_PORTRAIT_ARTIST_PATH` for repeatable checks
  against independently provisioned assets; the default synthetic fixture skips
  that scenario because it contains no portrait references.

  Staging verification (2026-11-06): after the coordinator restored the Railway
  `uat` service, the staged-image browser test passed against
  `https://beta.wga.hu/artists/aachen-hans-von-r49032850f1b20c`, including decoding
  the delivered portrait and matching its Open Graph and `Person.image` URLs.
  This check was read-only; no staging records or files were changed.
