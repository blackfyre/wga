## MODIFIED Requirements

### Requirement: Catalogue result views and paging

The system SHALL let visitors select the reference grid or list result presentation and navigate all result pages without losing active filters or the selected presentation. Artwork search SHALL present one view control that names the current view as `VIEW: GRID` or `VIEW: LIST` and links to the other view. When artwork search results span more than one page, the system SHALL render the reference pagination row below the results: `PAGE n OF m` on the left, and `← PREV` and `NEXT →` on the right. Each available direction SHALL be an ordinary link with an `href` that works without JavaScript, enhanced by HTMX to replace only the result fragment. An unavailable direction SHALL render as non-link text marked `aria-disabled`, and the row SHALL contain no link without an `href`.

#### Scenario: Visitor selects list view

- **WHEN** a visitor viewing the grid activates the `VIEW: GRID` control
- **THEN** matching works render in the list presentation, the control reads `VIEW: LIST`, and its accessible name states that it switches to the other view.

#### Scenario: Visitor visits another result page

- **WHEN** a visitor selects next or previous pagination
- **THEN** the requested page renders with the active filters and selected result view retained.

#### Scenario: Visitor reads the pagination row

- **WHEN** artwork search renders page `n` of `m` result pages, with `m` greater than one
- **THEN** the row reads `PAGE n OF m`, `← PREV` is a link only when `n` is greater than one, `NEXT →` is a link only when `n` is less than `m`, and each unavailable direction is non-link text marked `aria-disabled`.

#### Scenario: Visitor pages without JavaScript

- **WHEN** a visitor without JavaScript follows `NEXT →` or `← PREV`
- **THEN** the browser loads the full artwork-search page for the requested page at a shareable `/artworks` URL with the active filters and selected view retained.

### Requirement: Artwork search provides the release filter and sort contract

The system SHALL support artwork filtering by school, period, form, technique, current collection, and an inline year range; grid and dense-list results; URL-addressable filter state; and sorting by title, artist, or date. School and form SHALL accept repeated query values with OR matching within the same facet and AND matching between different facets. The selected collection SHALL use `venue`, collection-name search SHALL use `venue_q`, and tone-keyword filtering remains deferred until an authoritative source dataset is available. Sorting SHALL use `sort=title|artist|date` and `dir=asc|desc`, default to title ascending, and treat obsolete catalogue-order or invalid sort input as title ascending. The facets SHALL appear in the reference order TITLE OR ARTIST, COLLECTION, SCHOOL, FORM, TYPE, followed by TECHNIQUE and PERIOD, then YEAR RANGE. School and form options SHALL render as text rows that show the option label, a selected marker when picked, and the option count, while each row remains a native checkbox control that submits with the filter form without JavaScript.

#### Scenario: Visitor reviews or searches facets

- **WHEN** the artwork search renders filters
- **THEN** the facets appear in the reference order, each facet is independently collapsible, title and school are initially open, valued facets reopen automatically, collapsed facets state their active summary, the heading reports the number of active filters, and the collection facet lists counted holdings sorted and capped at forty with an honest hidden-holdings note.

#### Scenario: Visitor combines school or form values

- **WHEN** a visitor selects multiple school or form values alongside other active filters
- **THEN** the result URL preserves every selected value as a repeated parameter, values in the same facet match with OR semantics, different facets match with AND semantics, each option count reflects the candidate's yield under all other active filters while ignoring its own facet's selected values, and the active-filter total counts each multi-select facet once regardless of its number of picks.

#### Scenario: Visitor reviews or expands a counted multi-select facet

- **WHEN** school or form has more values than its reference initial cap
- **THEN** the complete roster is ordered by count descending and label ascending for ties, the collapsed facet shows the first eight ranked options plus any selected option outside that cap, unavailable zero-result values remain as disabled stable rows, selected rows expose their selected mark and count without artificial rank promotion, and `SHOW ALL N`/`SHOW FEWER` expose or collapse the bounded full list without changing result filters.

#### Scenario: Visitor picks a school or form row

- **WHEN** a visitor activates a school or form text row with a pointer, or focuses it and presses Space
- **THEN** the row's checkbox state toggles, assistive technology reports it as a checked or unchecked checkbox, the row shows or removes its selected marker after the update, and the visible row draws no native checkbox glyph.

#### Scenario: Visitor clears one multi-select facet

- **WHEN** a visitor activates the school or form facet's `CLEAR` action
- **THEN** only that facet's values are removed and every unrelated query, filter, view, and sort parameter is preserved.

#### Scenario: Scholar changes sort criterion

- **WHEN** a scholar selects a new artwork sort criterion
- **THEN** the result URL records that criterion with ascending direction, the selected control alone names `TITLE A–Z`, `ARTIST A–Z`, or `DATE EARLIEST`, the other two controls remain bare, and deterministic tie-breaking preserves stable pagination.

#### Scenario: Scholar reverses the active sort

- **WHEN** a scholar activates the currently selected TITLE, ARTIST, or DATE control
- **THEN** only its direction reverses, its label changes to `Z–A` or `LATEST` as applicable, and no separate reverse control or catalogue-order option is presented.

#### Scenario: Visitor follows an obsolete or invalid sort URL

- **WHEN** `sort=catalogue`, an unknown sort key, or an invalid direction reaches artwork search
- **THEN** the canonical result state and controls fall back to `sort=title&dir=asc` rather than exposing archive storage order.

## ADDED Requirements

### Requirement: Artwork search explains an empty filtered result

When active filters match no artworks, the system SHALL render the reference empty state: the heading `No works match these filters.`, the guidance `Widen the year range or clear the school filter — the collection is uneven across periods.`, and a `RESET FILTERS →` action that links to the unfiltered artwork search.

#### Scenario: Active filters match nothing

- **WHEN** a visitor's active artwork-search filters match no works
- **THEN** the result area shows the reference heading, guidance and `RESET FILTERS →` link, and the result count reads `0 WORKS MATCH`.
