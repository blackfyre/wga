# WGA Artist Directory — PocketBase + HTMX Implementation Spec

> Target: Rebuild the legacy WGA alphabetical artist index and per-letter listing with modern ergonomics, while preserving content parity. Backend: **PocketBase**. Frontend: **HTMX + htmx.org idioms**, vanilla HTML/CSS, optional Tailwind. Image service can be static or via reverse proxy.

---

## 1) Scope Overview

**User-facing:**

- Alphabet index (A–Z) landing
- Per-letter artist listing with search/filter and pagination
- Artist detail with bio + works
- Artwork detail with zoomable image viewer
- Global search (artists + artworks)
- Breadcrumbs, back-to-position navigation, accessible markup

**Admin-facing:**

- CRUD for artists, artworks, sources, images
- Import pipeline from legacy HTML/XML/CSV
- Merge/duplicate tooling
- Image integrity checks

---

## 2) Data Model (PocketBase Collections)

### 2.1 `artists`

- `slug: text` (unique, indexed; e.g. `altdorfer-albrecht`)
- `displayName: text` (e.g. `Altdorfer, Albrecht`)
- `sortName: text` (normalized for sorting/search, e.g. `altdorfer albrecht`)
- `letter: text` (single char A–Z derived from surname initial)
- `birthYear: number?`
- `deathYear: number?`
- `nationality: text?` (ISO or free text)
- `bio: text?` (HTML/Markdown)
- `sources: relation[] -> sources`
- `artworks: relation[] -> artworks` (optional; also maintained from `artworks.artist` ref)
- `aka: json[]?` (alternate names/transliterations)
- `meta: json?` (arbitrary metadata, flags like `needsReview`)

**Indexes:**

- unique(`slug`), index(`letter`), index(`sortName`), composite index(`displayName`, `birthYear`).

### 2.2 `artworks`

- `artist: relation -> artists` (required)
- `slug: text` (unique; `artist-slug/title-year` pattern suggested)
- `title: text`
- `titleAlt: text?`
- `yearStart: number?`
- `yearEnd: number?`
- `medium: text?` (e.g. `oil on panel`)
- `genre: text?`
- `dimensions: text?`
- `location: text?` (museum/collection)
- `city: text?` `country: text?`
- `image: file?` (PocketBase file, primary image)
- `images: file[]?` (additional)
- `sourceRef: relation[] -> sources`
- `meta: json?`

**Indexes:**

- unique(`slug`), index(`artist`), index(`title`).

### 2.3 `sources`

- `label: text` (e.g. `WGA`, `RKD`, `Grove`)
- `url: url?`
- `citation: text?`
- `license: text?`

### 2.4 `imports`

- `kind: text` (`wga-html`, `csv`, `xml`)
- `payload: file/json` (raw import blob or file)
- `status: text` (`queued`, `processing`, `done`, `error`)
- `log: text?`
- `stats: json?`

### 2.5 `duplicates`

- `entityType: text` (`artist`, `artwork`)
- `candidates: relation[]` (list of suspected dupes)
- `reason: text?`
- `resolution: text?` (`merged`, `dismissed`)

---

## 3) Routing & Endpoints (PocketBase API)

PocketBase REST is used via collection endpoints + filters.

### 3.1 Public Endpoints (read-only)

- `GET /api/collections/artists/records?filter=letter="A"&sort=sortName&perPage=50&page=1`
- `GET /api/collections/artists/records/{id or slug}`
- `GET /api/collections/artworks/records?filter=artist.slug="altdorfer-albrecht"`
- Global search: use text-indexed fields with `~` filter:  
  `GET /api/collections/artists/records?filter=displayName~"altdorf"`

### 3.2 Admin Endpoints

- CRUD for all collections
- Import upload: `POST /api/collections/imports/records` with file/json payload
- Import runner: server-side script/worker (see §7)

**Auth:** public read-only rules for `artists`, `artworks`, `sources`. Admin via PocketBase auth dashboard or JWT.

---

## 4) Frontend (HTMX) Architecture

### 4.1 Pages

- `/artists` — Alphabet landing
- `/artists/{letter}` — Letter listing
- `/artist/{slug}` — Artist detail
- `/artwork/{slug}` — Artwork detail
- `/search` — Global search results

### 4.2 HTMX Patterns

- **Letter navigation:** partial swaps for listing pane
  ```html
  <a hx-get="/artists/A" hx-target="#list" hx-push-url="true">A</a>
  ```
- **In-letter search:**
  ```html
  <input
    name="q"
    hx-get="/artists/A"
    hx-trigger="keyup changed delay:300ms"
    hx-target="#list"
  />
  ```
- **Pagination:**
  ```html
  <a hx-get="/artists/A?page=2" hx-target="#list" hx-push-url="true" rel="next"
    >Next</a
  >
  ```
- **Back to position:** use `id` anchors per row or store `scrollY` in `history.state` on click; restore on popstate.

### 4.3 Components

- Alphabet bar
- Artist row (name, dates, nationality, link)
- Filters: mini search within letter
- Breadcrumb
- Artwork card grid
- Image viewer panel (progressive image, zoom lib hook)

---

## 5) Import Pipeline

### 5.1 Steps

1. **Extract** legacy HTML/XML to structured JSON (artist, works, metadata, image paths).
2. **Normalize** names (displayName, sortName, slug), compute `letter`.
3. **De-duplicate** via fuzzy matching (Levenshtein on `sortName`, dates proximity).
4. **Upsert** artists, then artworks (idempotent by `slug`).
5. **Link** images: copy to PB storage or reference existing static path.
6. **Audit** missing fields; flag `meta.needsReview = true`.

### 5.2 CLI Script (pseudo)

```bash
# parse legacy → json
wga-import extract --input ./legacy/ --out ./build/extract.json

# load to PB
wga-import load --pb http://localhost:8090 --token $PB_TOKEN --file ./build/extract.json
```

### 5.3 Error Handling

- Write per-record status into `imports.log`
- Retry transient failures; halt on schema mismatch
- Produce duplicate candidates into `duplicates`

---

## 6) Pages: Requirements & Acceptance Criteria

### 6.1 Alphabet Index `/artists`

**Requirements**

- Show A–Z letters, disabled state for empty letters
- Show counts per letter (optional)
- Keyboard navigation support

**Acceptance**

- [ ] 26 letters render with correct enabled/disabled state
- [ ] Clicking “A” updates URL to `/artists/A` without full reload
- [ ] Focus ring and arrow-key nav cycle through letters
- [ ] Loading states visible during fetch

### 6.2 Letter Listing `/artists/{letter}`

**Requirements**

- List entries: displayName, years `(1500–1558)`, nationality
- Per-letter search box filtering current dataset
- Pagination or infinite scroll
- Stable anchors for back-to-position

**Acceptance**

- [ ] Sorted strictly by `sortName`
- [ ] Search `q` filters by `displayName` and `aka`
- [ ] Page size configurable; nav buttons update URL with `?page=`
- [ ] Returning from an artist detail restores scroll position

### 6.3 Artist Detail `/artist/{slug}`

**Requirements**

- Bio block with source attribution
- Facts: nationality, dates, alt names
- Artworks grid with filters (year range, medium, genre)
- Breadcrumbs: `Home › Artists › A › Altdorfer, Albrecht`

**Acceptance**

- [ ] Bio loads under 1s from cached API on local env
- [ ] Grid shows at least title + year and a thumbnail
- [ ] Filters update grid via HTMX without full reload
- [ ] Broken images replaced by placeholder and logged

### 6.4 Artwork Detail `/artwork/{slug}`

**Requirements**

- Large image with zoom/pan
- Metadata sidebar: title, year, medium, dimensions, location
- Back to artist link preserves prior scroll

**Acceptance**

- [ ] Image loads with responsive sizing and native zoom lib
- [ ] Metadata completeness ≥ 90% across migrated set
- [ ] Keyboard shortcuts: `Esc` returns to artist page

### 6.5 Global Search `/search`

**Requirements**

- Single input for artists + artworks
- Tabs or grouped results
- Debounced queries

**Acceptance**

- [ ] Query returns grouped results with counts
- [ ] Selecting a result navigates correctly
- [ ] Empty state is explicit and accessible

---

## 7) Backend Services & Jobs

- **Importer Worker:** consumes `imports` records and performs upsert
- **Image Inspector:** verifies referenced files, checks dimensions, generates thumbnails
- **Dedup Worker:** scheduled task producing `duplicates` candidates
- **Sitemap Builder:** periodic regeneration of `/sitemap.xml` for SEO

**Acceptance**

- [ ] Import 10k artists under 10 minutes locally
- [ ] Thumbnail generation for new images within 30 seconds of upload
- [ ] Duplicate report lists precision/recall metrics on sample set

---

## 8) Security, Perf, Ops

### Security

- Public read-only rules; write restricted to admins
- Rate limiting at reverse proxy
- Sanitization of HTML fields in `bio` (allowlist tags)

### Performance

- HTTP caching for list endpoints (ETag + Cache-Control)
- Precomputed `letter` counts for alphabet bar
- Lightweight cards, lazy-loaded images

### Observability

- Access logs with request IDs
- Import pipeline logs stored with `imports` record
- Error reporting to stderr and PB logs

---

## 9) Testing Strategy

- **Unit:** name normalization, slug generation, letter derivation
- **Integration:** import → query parity checks
- **E2E:** Cypress/Playwright happy paths for A–Z, search, deep link
- **Visual:** Percy-style snapshots for artist and artwork pages

**Acceptance**

- [ ] Green test suite in CI
- [ ] Sample regression list of 50 legacy artists matches legacy links 1:1

---

## 10) Task Breakdown (Work Items)

### Epic A: Data Foundations

- [ ] Define PocketBase collections and indexes
- [ ] Implement normalization utilities
- [ ] Seed with 100-sample dataset

### Epic B: Importer

- [ ] Legacy parser (HTML/XML → JSON)
- [ ] Loader to PB with idempotent upsert
- [ ] Duplicate detector + merge tooling
- [ ] Import report dashboard

### Epic C: Public UI

- [ ] Alphabet bar component
- [ ] Letter listing with HTMX pagination + search
- [ ] Artist detail + artwork grid + filters
- [ ] Artwork detail viewer + zoom
- [ ] Breadcrumbs + back-to-position

### Epic D: Search

- [ ] API filters for artists/artworks
- [ ] Global search endpoint aggregation
- [ ] UI with grouped results and tabs

### Epic E: Images

- [ ] Storage strategy & CDN/proxy
- [ ] Thumbnail generation
- [ ] Integrity checker job

### Epic F: Ops

- [ ] Reverse proxy, caching, compression
- [ ] Access logs, metrics, error reporting
- [ ] Backup and restore runbooks

---

## 11) Definition of Done (Global)

- [ ] A–Z navigation, per-letter lists, artist and artwork pages live
- [ ] Import pipeline can ingest legacy dump end-to-end
- [ ] Search returns relevant cross-entity results
- [ ] Accessibility checks pass (labels, contrast, keyboard nav)
- [ ] Observability in place and documented
- [ ] Smoke tests and basic E2E run in CI

---

## 12) Nice-to-Have

- Quick keyboard jump by letter: press `A` to focus `/artists/A`
- URL aliases for alternative spellings
- Print-friendly artist dossier page
- Dark mode toggle
- WebP/AVIF variants for thumbnails
