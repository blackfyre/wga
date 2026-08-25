## 1. Fresh PocketBase thumbnail contract

- [x] 1.1 Replace the baseline `artworks.image` and `artists.portrait` thumbnail declarations with the authoritative `Wx0` variants; a newly reset database exposes exactly those field contracts.
- [x] 1.2 Update the fresh-schema migration tests to assert every artwork variant and both portrait variants; the test fails if a legacy profile is restored.

## 2. Public rendition delivery

- [x] 2.1 Add named artwork and portrait rendition values to the URL utility and focused tests; each produces the expected public PocketBase `?thumb=Wx0` URL.
- [x] 2.2 Populate existing catalogue grid and list, artist works, related works, home, inspiration, postcard, and portrait surfaces with their assigned finished rendition URLs; missing filenames retain the existing fallback.
- [x] 2.3 Give artwork record rendering separate display and viewer-zoom URLs, preserving an original URL only for intentional download or metadata use.
- [x] 2.4 Update public artwork and portrait templates so grids are not ViewerJS mounts, the artwork record remains the viewer entry point, and artist-index portraits render when available.
- [x] 2.5 Run `templ generate`; generated ignored files reflect the changed Templ sources without being committed.

## 3. Verification

- [x] 3.1 Add or update focused URL, handler, migration, and template-facing tests for the authoritative surface mapping and ViewerJS boundary.
- [x] 3.2 Run the focused Go tests for modified packages, then `go test ./...`; all tests pass.
- [x] 3.3 Manually verify against a freshly reset data directory that representative card, row, record, zoom, postcard, and portrait requests use their assigned `Wx0` URLs and no artwork grid opens ViewerJS.
