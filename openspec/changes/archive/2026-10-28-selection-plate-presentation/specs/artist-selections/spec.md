## MODIFIED Requirements

### Requirement: Artist records distinguish curated selections from catalogue holdings

The system SHALL choose an artist record's selection presentation from its number of published source-backed curated selections:

- **Fewer than two:** the record SHALL retain the artist's ordinary works presentation.
- **Two to nine:** the record SHALL render selection previews.
- **Ten or more (a large holding):** the record SHALL render the form-grouped selection index instead of selection previews.

Each selection preview SHALL state the selection's supplied display title, its selected and catalogued counts, its editorial lede, its representative works, and its dedicated selection route. Each preview's representative works SHALL use the same selection plate grid and selection plates as the dedicated selection page.

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

#### Scenario: Preview plates match selection page plates

- **WHEN** a visitor views a selection preview on the artist record and then the dedicated page for that selection at the same viewport width
- **THEN** the plate image boxes in both places have the same width and height.

### Requirement: A selection is a citable editorial record

The system SHALL render each selection at a stable artist-and-selection route derived from the producer's deterministic selection identity, with reference section number `21`, its lede, commentary, selected works in the selection plate grid, links to the artist's other selections, a route to the wider holding, and a citation for the editorial text. The selection plate grid SHALL show two equal columns below the reference medium breakpoint and, from that breakpoint, fixed-width plate columns that fill the available width, so plate size depends on the viewport width and not on the width of the surrounding page column.

#### Scenario: Scholar opens a selection

- **WHEN** a scholar follows a selection route
- **THEN** the page distinguishes the selected argument from the full catalogue holding, presents section `21` and the selection plate grid, and provides a copyable citation for that selection.

## ADDED Requirements

### Requirement: Selection plates show the stored holding location

A selection plate on the selection page or in an artist record's selection preview SHALL show the work's stored current holding name as a separate text entry after the title and artist; a long name MAY wrap, and it SHALL NOT be truncated. When the work has no stored current holding, or the stored holding cannot be resolved, the plate SHALL omit the entry and SHALL NOT infer a location from other text. Plates outside selections SHALL NOT show the entry.

#### Scenario: Work has a stored holding

- **WHEN** a selection includes a work whose current holding is stored as `Kunsthistorisches Museum, Vienna`
- **THEN** that work's plate shows `Kunsthistorisches Museum, Vienna` as a separate entry below the title and artist.

#### Scenario: Work has no stored holding

- **WHEN** a selection includes a work with no stored current holding
- **THEN** that work's plate shows no location entry and no placeholder.

#### Scenario: Plate outside a selection

- **WHEN** the same work appears on the Inspiration page
- **THEN** its plate shows no location entry.

### Requirement: Plates in one grid align

Public artwork plates in one grid SHALL have identical image boxes, and within one grid row the plates' trailing `VIEW WORK →` actions SHALL share one vertical position, as SHALL their itinerary actions, regardless of title, artist, or location length.

#### Scenario: Titles of different lengths share a row

- **WHEN** a row contains one plate with a one-line title and another with a three-line title
- **THEN** both plates' image boxes have the same size and their `VIEW WORK →` actions start at the same vertical position.
