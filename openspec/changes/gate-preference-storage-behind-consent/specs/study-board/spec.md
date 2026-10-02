## MODIFIED Requirements

### Requirement: The Study Board is an anonymous URL-shareable workspace

The system SHALL maintain an ordered, duplicate-free set of at most twelve published artworks on a stable Study Board route. The canonical board URL SHALL encode the ordered artwork identifiers as one comma-separated `board` query value. That URL SHALL be sufficient to share and restore the board without a server-persisted board record.

The system SHALL remember the board in the `wga-study-board` browser local storage entry. It SHALL read or write that entry only while the visitor has accepted the optional `preferences` cookie-consent category. Without that consent, the board SHALL be held for the current visit only, through its URL and in-page state, and any stored copy SHALL be deleted. When consent is first granted, the system SHALL store the board the page currently holds. When consent is withdrawn, the system SHALL delete the entry while the open page keeps its board.

A valid URL board SHALL outrank remembered state for the page that opens it. It SHALL NOT replace the remembered board. Only adding, removing, reordering or clearing works, or the first grant of consent, SHALL write it. Without URL state, the visitor's remembered board SHALL be restored.

Invalid, missing, and unpublished identifiers SHALL be omitted without exposing their metadata. After validation and duplicate removal, only the first twelve published identifiers SHALL be retained, and the URL SHALL canonicalise to that bounded list.

#### Scenario: Recipient opens a shared board

- **WHEN** a recipient opens a board URL containing valid and invalid artwork identifiers
- **THEN** the first twelve published valid works appear once in URL order, invalid entries expose no metadata, the URL canonicalises to those retained identifiers, and no account or stored board record is required.

#### Scenario: Visitor returns without a board URL

- **WHEN** a visitor with `preferences` consent previously maintained a board in the same browser and opens the Study Board without URL state
- **THEN** the remembered local board is restored and its canonical ordered URL is established.

#### Scenario: A shared board does not replace the remembered board

- **WHEN** a visitor with `preferences` consent and a remembered board opens a different `?board=` URL, and later opens the Study Board without URL state
- **THEN** the shared board is shown for that page, and the later visit restores the original remembered board.

#### Scenario: Visitor has not accepted preference storage

- **WHEN** a visitor without `preferences` consent adds works to the board and then reloads a page without board URL state
- **THEN** nothing was written to local storage, and the reloaded page shows no remembered board or shelf.
