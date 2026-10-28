# artist-selections Specification

## Purpose

TBD - created by archiving change adopt-visual-overhaul. Update Purpose after archive.

## Requirements

### Requirement: Artist records distinguish curated selections from catalogue holdings

The system SHALL choose an artist record's selection presentation from its number of published source-backed curated selections:

- **Fewer than two:** the record SHALL retain the artist's ordinary works presentation.
- **Two to nine:** the record SHALL render selection previews.
- **Ten or more (a large holding):** the record SHALL render the form-grouped selection index instead of selection previews.

Each selection preview SHALL state the selection's supplied display title, its selected and catalogued counts, its editorial lede, its representative works, and its dedicated selection route.

For a large holding, the system SHALL derive each selection's form and qualifier from its display title:

- A title of the shape `Form (qualifier)` SHALL yield the text before the final bracketed part as its form and the bracketed text as its qualifier.
- Otherwise, a title of the shape `Form: qualifier` SHALL yield the text before the first colon as its form and the remainder as its qualifier.
- A trailing work count after `·`, a spaced dash or a comma SHALL be removed before the form and qualifier are derived. A work count SHALL be one to three digits, or digits grouped with thousands separators, so a trailing four-digit year such as `Saint Ursula Shrine, 1489` and a year range such as `1435-40` SHALL be kept.
- A title that matches neither shape SHALL become its own form under its full title, with no qualifier.
- A hyphenated year range in a qualifier SHALL be displayed with an en dash.

Selections SHALL be grouped by form, with forms compared case-insensitively. Forms SHALL appear in the order in which each first occurs in the artist's published selection order, and selections SHALL keep that order within each form. A selection's work count SHALL be the number of works it holds.

The main column of a large holding SHALL render one block per form:

- Each block SHALL carry a stable fragment identifier, the form name, and a `N SELECTIONS · N WORKS` header that totals its selections.
- Each block SHALL present a grid of square tiles, one per selection.
- Each tile SHALL show the selection's qualifier, or its full display title when it has none, and its work count. Each tile SHALL link to that selection's stable route.
- A tile's cover SHALL be a representative work from that selection. When no cover image resolves, the tile SHALL show the form name in place of the image.

A large-holding record SHALL NOT render selection commentary, preview plates, or `OPEN SELECTION` actions; these remain on the selection record.

#### Scenario: Artist has multiple curated selections

- **WHEN** a visitor opens an artist record with between two and nine published source-backed curated selections
- **THEN** each preview states its supplied display title, selected and catalogued counts, editorial lede, representative works, and dedicated selection route.

#### Scenario: Artist has a large holding

- **WHEN** a visitor opens an artist record with ten or more published source-backed curated selections
- **THEN** the record shows one block per form instead of selection previews. Each block has its `N SELECTIONS · N WORKS` header and one tile per selection, and each tile shows the selection's qualifier and work count and links to the dedicated selection route.

#### Scenario: Selections share a form

- **WHEN** a large holding's published selections include `Drawings (1484-1499)`, `Woodcuts (1489-1500)` and `drawings (1500-1505)`, in that order
- **THEN** the record shows a `Drawings` block before a `Woodcuts` block, and the `Drawings` block holds tiles labelled `1484–1499` and `1500–1505`.

#### Scenario: A selection title does not parse

- **WHEN** a large holding has a published selection titled `The Navicella mosaic`, with no bracketed or colon qualifier
- **THEN** that selection forms its own block under its full title, and the block's tile shows the full title and the selection's work count.

#### Scenario: A selection has no cover image

- **WHEN** no representative work of a large holding's selection resolves an image
- **THEN** its tile shows the form name in place of an image and remains a working link to the selection route.

### Requirement: A selection is a citable editorial record

The system SHALL render each selection at a stable artist-and-selection route derived from the producer's deterministic selection identity, with reference section number `21`, its lede, commentary, selected works in the reference responsive two-column/four-column work-card grid, links to the artist's other selections, a route to the wider holding, and a citation for the editorial text.

#### Scenario: Scholar opens a selection

- **WHEN** a scholar follows a selection route
- **THEN** the page distinguishes the selected argument from the full catalogue holding, presents section `21` and the responsive two-column/four-column work-card grid, and provides a copyable citation for that selection.

### Requirement: Missing editorial commentary is represented honestly

The system SHALL not generate or imply selection commentary where none exists.

#### Scenario: Selection has no commentary

- **WHEN** a visitor opens a curated selection without supplied commentary
- **THEN** the page states that the commentary is unavailable and does not substitute generated prose.

### Requirement: Curated artist records provide an optional stable in-page index

The system SHALL render `ON THIS PAGE` only when an artist record has more than one published source-backed curated selection. Its entries SHALL be ordinary fragment links. The index SHALL begin with Biography and end with Cite This Record. Between them it SHALL list:

- for selection previews, every rendered selection form, each reporting the number of works in that selection rather than the number its preview shows, as `1 work` or `N works`;
- for a large holding, one row per form.

A large holding's form rows SHALL follow these rules:

- **Row content:** each row SHALL show the form name, its selection count, and its total work count, for example `DRAWINGS · 7 SEL · 84`, and SHALL link to that form's block.
- **Folding:** a form with more than one selection SHALL be foldable. A form with exactly one selection SHALL NOT fold.
- **Open list:** an open form SHALL list its selections in two columns. Each entry SHALL show the selection's qualifier, or its full display title when it has none, and SHALL link to that selection's tile.
- **Control:** a `+`/`−` control SHALL open or close any foldable form, and SHALL report the form's state through `aria-expanded`.
- **Default state with JavaScript:** only the form containing the reader's current scroll position SHALL be open by default.
- **Default state without JavaScript:** every foldable form SHALL be rendered open, so that every selection link remains reachable.
- **Scroll highlighting:** scroll-position highlighting SHALL track forms, and SHALL NOT treat individual tiles as separate sections.

Scroll-position highlighting MAY enhance the links but SHALL NOT be required for navigation.

The optional period-music player and the index SHALL share one rail. From the reference medium breakpoint, the rail SHALL become sticky, subtract the measured bottom stack from its available viewport height, and scroll internally when necessary. Below that breakpoint the rail SHALL remain in normal document flow.

An artist with fewer than two published source-backed curated selections SHALL retain the ordinary works presentation. Its record SHALL omit selection previews, selection forms, the form index, and the one-section TOC. When the artist has holdings beyond those shown on the page, its artwork-search action SHALL instead appear after the citation as the final onward link.

#### Scenario: Visitor follows the artist record index

- **WHEN** a visitor activates an `ON THIS PAGE` entry, form row or selection entry with a pointer or keyboard
- **THEN** browser-native fragment navigation reaches the labelled section, form block or selection tile, and remains functional without JavaScript.

#### Scenario: Visitor folds and unfolds a form

- **WHEN** a visitor with JavaScript activates the `+` control of a closed form in a large holding that holds several selections
- **THEN** that form lists its selections in two columns, and its control shows `−` with `aria-expanded="true"`
- **AND** activating the control again closes the list and sets `aria-expanded="false"`.

#### Scenario: Reader scrolls into a form

- **WHEN** a visitor with JavaScript scrolls a large holding from one form block into the next
- **THEN** the index highlights the new form, opens it if it folds, and closes the previously open form
- **AND** tiles in the same row do not change the highlight among themselves.

#### Scenario: Form holds a single selection

- **WHEN** a form in a large holding holds exactly one selection
- **THEN** its index row has no fold control and links straight to that form's block.

#### Scenario: Index without JavaScript

- **WHEN** a visitor without JavaScript opens a large-holding artist record
- **THEN** every foldable form is listed open, each of its selection entries is linked, and no non-working fold control is offered.

#### Scenario: Artist has no curated selection

- **WHEN** an artist record contains biography, holdings and citation but no curated selection
- **THEN** no `ON THIS PAGE` navigation is rendered, and the rail contains no decorative navigation for a single argument
- **AND** any eligible `FIND MORE BY … IN THE ARTWORK SEARCH →` action follows the citation.

#### Scenario: Artist has exactly one curated selection

- **WHEN** an artist record has exactly one published source-backed curated selection
- **THEN** the record retains its ordinary works presentation, renders no selection preview, selection form or form index, and omits `ON THIS PAGE`
- **AND** any eligible `FIND MORE BY … IN THE ARTWORK SEARCH →` action follows the citation.

#### Scenario: Sticky artist rail shares space with fixed workspaces

- **WHEN** a curated artist record has period music or a TOC, and a Study Board or itinerary tray is visible
- **THEN** the combined rail remains within the viewport above the measured bottom stack, and its own content scrolls without obscuring its links or player.

#### Scenario: A selection holds more works than its preview shows

- **WHEN** an artist record shows a preview of four works for a selection of seven, alongside a selection of one work
- **THEN** the seven-work selection's `ON THIS PAGE` entry reads `· 7 works`, and the one-work selection's entry reads `· 1 work`.

#### Scenario: Form totals count held works

- **WHEN** a form in a large holding holds two selections of 16 and 11 works
- **THEN** its index row reads `· 2 SEL · 27`, its block header reads `2 SELECTIONS · 27 WORKS`, and its tiles read `16 works` and `11 works`.
