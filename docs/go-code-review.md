# Go Code Review

## Summary

- Routing and rendering logic generally works but several handlers hide subtle bugs and mix unrelated concerns.
- Utility packages have grown organically; duplicated helpers and inconsistent naming make reuse and testing harder.
- A few generators and background tasks rely on console prints and global state, which complicates observability and reproducibility.

## High Priority Issues

- internal/handlers/dual/main.go:74 – `relPath.Query().Add(...)` mutates a copy of the query values, so the composed URL never includes `left`, `right`, or `*_render_to` parameters. Persist the values via `vals := relPath.Query(); vals.Set(...); relPath.RawQuery = vals.Encode()`.
- internal/handlers/dual/main.go:135 – the default left pane assigns `"Left pane"` but immediately overwrites it with `buf.String()`, which is empty. Render a template into the buffer or skip the overwrite to avoid returning a blank column.
- internal/handlers/artworks/main.go:139 – every result row triggers an extra `FindRecordById("artists", ...)`, leading to N+1 queries. Prefetch authors or extend the artwork query to join related artist data.
- internal/utils/seed/images.go:72 – thumbnails are generated even when `artwork.GetString("image")` is empty, producing S3 keys ending in `/`. Add a guard or skip empty images.

## Architectural & Structuring Observations

- internal/handlers/artworks/main.go:63 – the handler bundles query parsing, PocketBase lookups, DTO assembly, pagination, and template rendering. Split these into smaller helpers (query parsing, store access, view model) to improve reuse (e.g. for HTMX endpoints) and unit testing.
- internal/handlers/artists/main.go:18 – similar monolithic flow plus repeated option loading (`getArtFormOptions`, `getArtTypesOptions` etc.) scattered across handlers. Consider a shared data service layer or cached repository package for dropdown data.
- internal/utils – the package mixes template helpers, HTTP error rendering, URL helpers, string utilities, and persistence helpers. Breaking these into focused subpackages (e.g. `internal/utils/template`, `internal/utils/http`, `internal/utils/strings`) would prevent circular dependencies and clarify ownership.
- internal/utils/main.go:334 & internal/utils/url/main.go:94 both expose `GenerateCurrentPageUrl` with different logic. Consolidate into a single implementation to avoid diverging behaviour.
- internal/handlers/contributors/main.go:13 – fetching contributors writes to `contributors.json` on every request. Move this into a background refresh job or cache with TTL to avoid filesystem coupling inside request handlers.

## Code Style & Consistency

- internal/utils/listMusicUrls.go:11 – structs use `Url`/`Id` instead of the Go-preferred `URL`/`ID`, and the file has indentation drift from missing `gofmt`. Run `gofmt` and align naming with Go initialism conventions.
- internal/utils/seed/images.go:20 & internal/utils/listMusicUrls.go:44 – server-side utilities use `fmt.Println` for errors, bypassing the structured logger used elsewhere. Prefer `app.Logger()` or a shared logger so logs stay consistent.
- internal/handlers/dual/main.go:27 – comments still mention the old `echo.Context` API, but the handlers now receive `*core.RequestEvent`. Update documentation to match current APIs.
- internal/handlers/dual/main.go:65 and other functions declare local variables starting with capital letters (`ArtistNameList`, `JsonLd`), which reads like exported identifiers. Keep locals lower-case to match Go style.
- internal/utils/pagination.go:164 – manual concatenation of query strings (`strParam = strParam + "&" + k + "=" + v[0]`) ignores URL encoding and drops multi-valued params. Reuse `url.Values.Encode()` to build compliant URLs.

## Additional Recommendations

- internal/utils/seed/images.go:77 – seeding uses `rand.Intn` without seeding the RNG, producing the same "random" sequence each run. Seed with `rand.New(rand.NewSource(time.Now().UnixNano()))` or switch to crypto/rand when uniqueness matters.
- internal/utils/url.go:11 – `AssetUrl` trusts `WGA_PROTOCOL`/`WGA_HOSTNAME` without defaults, yielding malformed URLs when env vars are missing. Provide fallbacks or error handling for local development.
- internal/handlers/postcards/save.go:54 – on `app.Save` failure the code re-renders the form but does not echo validation errors back to the client. Bubble the specific error to improve UX and debuggability.
- internal/handlers/artists/main.go:84 – counting records by fetching the full result set scales poorly. Use PocketBase’s `TotalItems` (if available) or a lightweight aggregate query.
- internal/utils/listMusicUrls.go:75 – writing `musicUrls.json` to the repository root couples runtime behaviour to the working directory. Accept an output path or stream results instead.
