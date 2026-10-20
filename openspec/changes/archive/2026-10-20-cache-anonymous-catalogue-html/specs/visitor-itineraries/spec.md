## MODIFIED Requirements

### Requirement: Visitors can maintain an anonymous bounded itinerary draft

The system SHALL let a visitor add published artworks to one session-owned draft of at most fifteen stops, show the fixed dark-inverted draft tray after the first addition with `CLEAR` and `ARRANGE & NARRATE →` actions, preserve the draft across navigation and reload, and reserve enough bottom space that neither page content nor toast notifications are obscured by the tray. The itinerary session SHALL NOT be created by viewing a catalogue page or by a rejected addition; it SHALL be created by the first successful addition, or by opening an itinerary page whose forms require a session (the builder, the Study Board import, and the publication confirmation). Catalogue documents SHALL NOT embed the session token, tray, or added state; the tray and the added and full state of add controls SHALL load after the document from an uncached session fragment and SHALL be reapplied after HTMX swaps; without JavaScript, the tray is shown on itinerary pages. A first addition without a session SHALL be accepted only from a same-origin request; an addition with a session SHALL require the session's CSRF token, and a no-JavaScript addition without it SHALL receive a confirmation form, and an HTMX addition without it SHALL receive the session tray and a retry notice, rather than a rejection. Add controls SHALL expose the reference `compact`, `row`, and `block` presentations: compact uses short labels, row is a 46px inline full-label action, and block is a 50px full-width action.

#### Scenario: Visitor adds a work

- **WHEN** a visitor adds a published work from an artwork-search grid card or dense row, another supported card, a record, or a dual-mode pane
- **THEN** the itinerary tray shows the draft count, clear and builder actions, the work appears once in the session-owned draft, and any resulting toast is fully visible above the tray.

#### Scenario: Visitor adds a search result without opening its record

- **WHEN** a visitor activates `ADD TO ITINERARY +` on an artwork-search result
- **THEN** the draft mutation completes without following the surrounding artwork link, and the control reports `IN ITINERARY ✓` or `ITINERARY FULL` as applicable.

#### Scenario: Visitor views catalogue pages without adding

- **WHEN** a visitor without an itinerary session browses catalogue pages
- **THEN** no itinerary session cookie is issued and no tray is shown

#### Scenario: Visitor makes a first addition from a cached page

- **WHEN** a visitor without a session adds a work from a same-origin catalogue page
- **THEN** the session is created, the work is added, the tray appears, and the control reports `IN ITINERARY ✓`

#### Scenario: Added state survives a fragment swap

- **WHEN** a visitor with a draft swaps catalogue content through HTMX
- **THEN** add controls in the new content report `IN ITINERARY ✓` or `ITINERARY FULL` as applicable

#### Scenario: Cross-origin first addition

- **WHEN** an add request without a session arrives without a same-origin indication
- **THEN** it is rejected and no session is created

#### Scenario: Visitor without JavaScript adds again

- **WHEN** a visitor with a session and without JavaScript adds a work from a shared-cached page
- **THEN** a confirmation form carrying the session token is shown, and confirming adds the work
