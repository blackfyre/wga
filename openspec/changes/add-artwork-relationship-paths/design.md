## Context

The PocketBase `artworks` collection currently uses a multi-value `author` relation even though source imports provide one author. Artwork technique and location information are currently display text, so the application cannot safely derive related artworks by those values. The public application also uses `author` for artist pages, artwork URLs, search, and JSON-LD.

Curators need an admin-friendly way to maintain a bounded set of meaningful artwork connections. The design must preserve source text, avoid linking unrelated private collections, and allow location parsing to improve over repeated imports.

## Goals / Non-Goals

**Goals:**

- Represent the agreed relationship paths as explicit PocketBase artwork relations.
- Make primary authorship distinct from co-authorship.
- Present public related artworks with the reason for each shared relation.
- Resolve current museums conservatively while preserving raw location text and exposing unresolved values for review.
- Keep curation workable in the PocketBase dashboard without a custom relationship editor.

**Non-Goals:**

- Build a generic subject-predicate-object graph.
- Infer museum identities from fuzzy matches without curator confirmation.
- Treat private collections as a public location or related-artwork path.
- Model every possible art-historical relationship or automated authority-data reconciliation.
- Implement ordered panels or part-whole hierarchy for altarpieces in this change.

## Decisions

### Use fixed artwork relation fields instead of a generic edge collection

`artworks` will hold an optional single `primary_author` relation and a multi-value `co_authors` relation. `primary_author` is optional at schema level because a legacy record with several authors cannot be assigned one without curator review. The shared-connection fields will be direct relations: school/workshop (existing `school`), subjects, series/altarpieces, original locations (`original_locations`), current museums (existing `current_location_id`), techniques, and art periods (existing `art_period_id`).

This makes all common curation actions available directly in the PocketBase artwork form. A generic relationship collection would permit arbitrary tags, but PocketBase's dashboard does not provide an equivalent concise inline editing experience and a polymorphic edge would weaken relation validation.

The initial fixed authorship qualities are primary author and co-author. Further qualities are schema additions, not unbounded data values, so they remain deliberate collection decisions.

### Canonical records represent shared concepts

New collections will be created for subjects, series, and techniques, plus `location_aliases` for the reviewed alias mapping. Current museums and original locations reuse the existing producer-owned `locations` collection, whose `museum` and `is_public` flags distinguish institutions, other public places, and private collections; no separate museums collection is created. Existing `schools` and `art_periods` records will be reused where their vocabulary is appropriate. The source provides no subjects or series, so those collections start empty and are curator-managed. Multiple artworks become related because they reference the same canonical record; no pairwise "same as" records are stored.

`series` records will include a small type distinction for series and altarpieces. This supports the required related-work label without prematurely introducing panel ordering or whole-to-part semantics.

Original locations and current museums are separate relations, even though both target `locations`. A museum is an institution and a current holding, whereas an original location can be a building or historical site. Conflating the relations would produce incorrect connection labels.

### Derive connection paths at read time

Artwork detail handling will collect published artworks sharing any configured relation, retain every shared relation per result, and render reason labels such as “Same artist”, “Same technique”, or “Same current museum”. Artist matching will consider both primary author and co-author relations.

The result set will be deduplicated by artwork ID after collecting paths. This avoids duplicate cards while retaining multiple reasons for a connection. Private-collection classifications never participate in this query.

### Preserve raw locations and match museums through reviewed aliases

The `wga-src` producer owns location normalisation, private-collection classification, alias review, and the unresolved-location report (its `location-normalization` specification). WGA imports the producer's results: the raw text into `artworks.source_location`, the resolved `current_location_id`, the canonical `locations` with their flags, and the reviewed `location_aliases`. WGA does not duplicate the normaliser or the report. The rules below describe the producer behaviour that WGA relies on.

The import representation will retain raw location text independently of structured fields. Parsing creates a comparison key using safe normalisation such as case folding, whitespace collapse, punctuation handling, and diacritic handling; it does not overwrite the source text.

Private-collection variants are matched by a configured classifier and resolved to a canonical `locations` record with `is_public = false` and `museum = false`; that classification never produces a public related-artwork path. Museum resolution first uses exact comparison-key matches against curator-maintained canonical names and aliases. Edit-distance comparisons produce scored report candidates only. Curators confirm a candidate by adding an alias; the next import then resolves it exactly.

### Migrate author data in stages

The migration will add the replacement fields and collections before changing application reads. Single legacy `author` values copy to `primary_author`. Artworks with multiple legacy authors are reported for manual curation and retain their legacy data until resolved; the migration must not silently choose or discard an artist.

Import, handlers, queries, URLs, and JSON-LD read `primary_author` and `co_authors`. While an artwork has no `primary_author`, every read falls back to the legacy `author` relation, so artworks awaiting multi-author review stay attributed and public. The fallback and the legacy relation are removed only in a later, verified migration step once the review report is clear.

## Risks / Trade-offs

- [A fixed set of authorship fields is less flexible than a typed edge table] → Add a field only when a supported curatorial quality becomes concrete; do not introduce a generic graph pre-emptively.
- [Fuzzy museum matches can misidentify institutions] → Use edit distance for review candidates only and require an explicit alias before linking.
- [Multiple legacy author values cannot prove primary authorship] → Report them and retain the legacy relation until manually resolved.
- [Related-artwork queries can expand as vocabularies grow] → Restrict queries to published works, deduplicate results, and add focused query tests and limits where needed.
- [A generic “Private collection” record would falsely connect unrelated works] → Store only a non-public classification, not a canonical public location relation.

## Migration Plan

1. Add canonical collections and replacement artwork relations through PocketBase migrations.
2. Add raw-location preservation, private-collection classification, museum aliases, and unresolved-location reporting to the import pipeline.
3. Copy single legacy author assignments to `primary_author`; emit a review report for multi-value assignments.
4. Populate initial canonical records and reviewed aliases, then validate imported relationship counts and unresolved-location output.
5. Update application readers, search, URLs, JSON-LD, and artwork detail presentation to use the replacement relations and derived paths.
6. Verify migrated data and public output before removing the legacy `author` relation in a follow-up migration.

Rollback before legacy-field removal consists of reverting application reads to `author` and retaining the new collections for inspection. Database backups are required before applying the data migration; a destructive rollback after legacy removal is not assumed.

## Open Questions

- Which additional authorship qualities, if any, are required beyond primary author and co-author?
- Does an altarpiece need panel order or a parent artwork in a later change?
- Can an artwork have more than one original location, and should buildings receive their own curator-facing record type?
- Which existing art-period vocabulary is authoritative for artwork assignment?
