## ADDED Requirements

### Requirement: Artwork search remembers presentation choices

The system SHALL remember an artwork search visitor's sort key, sort direction, result view, and result-actions setting on that visitor's device across page loads, using the `wga-aw-prefs` browser storage entry and the `wga_aw_prefs` first-party cookie. These choices are presentation state, not part of the query. Remembering is optional storage: the system SHALL read or write either copy only while the visitor has accepted the optional `preferences` cookie-consent category, SHALL delete both when that consent is absent or withdrawn, and without it SHALL apply the choices to the current page only. A visit to `/artworks` with no query parameters SHALL render with the remembered sort, direction, and view without a redirect. Any explicit query parameter SHALL take precedence over every remembered sort, direction, and view, so a shared or bookmarked search URL renders as addressed. Resetting filters SHALL leave the remembered choices unchanged. A missing, malformed, or invalid remembered value SHALL be ignored in favour of the ordinary defaults. Sort and view controls SHALL remain ordinary links that work without JavaScript.

#### Scenario: Visitor returns to artwork search

- **WHEN** a visitor who last chose date-descending sort in list view opens `/artworks` with no query parameters
- **THEN** the results render sorted by date descending in the list view, the controls report that state, and the canonical result URL records it.

#### Scenario: Visitor has not accepted preference storage

- **WHEN** a visitor without `preferences` consent changes the view or shows result actions and then reloads `/artworks`
- **THEN** the change applied to the page they were on, no `wga_aw_prefs` cookie or `wga-aw-prefs` entry exists, and the reloaded page uses the defaults.

#### Scenario: Visitor withdraws preference storage

- **WHEN** a visitor who accepted `preferences` consent withdraws it in the cookie preferences
- **THEN** the `wga_aw_prefs` cookie and the `wga-aw-prefs` entry are deleted, the current page keeps its state, and later visits use the defaults.

#### Scenario: Visitor opens a shared search link

- **WHEN** a visitor with remembered choices opens an artwork search URL that carries any query parameter
- **THEN** the results render the state that URL addresses and the remembered sort, direction, and view are not applied.

#### Scenario: Visitor resets filters

- **WHEN** a visitor with remembered choices activates `RESET`
- **THEN** every result filter is cleared while the remembered sort, direction, view, and actions setting remain in effect.

#### Scenario: Remembered state is unusable

- **WHEN** the remembered cookie is absent, malformed, or names an unknown sort, direction, or view
- **THEN** artwork search renders the title-ascending grid defaults for the unusable values and does not fail the request.

#### Scenario: Visitor browses without JavaScript

- **WHEN** a visitor without JavaScript activates a sort or view control
- **THEN** the browser follows the control's ordinary link to the addressed result state.

## MODIFIED Requirements

### Requirement: Catalogue result views and paging

The system SHALL let visitors select the reference grid or list result presentation and navigate all result pages without losing active filters or the selected presentation. Artwork search SHALL present one view control that names the current view as `VIEW: GRID` or `VIEW: LIST` and links to the other view.

#### Scenario: Visitor selects list view

- **WHEN** a visitor viewing the grid activates the `VIEW: GRID` control
- **THEN** matching works render in the list presentation, the control reads `VIEW: LIST`, and its accessible name states that it switches to the other view.

#### Scenario: Visitor visits another result page

- **WHEN** a visitor selects next or previous pagination
- **THEN** the requested page renders with the active filters and selected result view retained.

### Requirement: Artwork search cards expose the reference metadata

The system SHALL project and render each artwork search result's filing-form artist name, date, school, form, and type without substituting technique for those fields. A grid card SHALL show the title, `ARTIST · DATE` on its first metadata line, and `SCHOOL · TYPE` on its second, followed, when result actions are shown, by a full-width vertical itinerary/Study Board control block with consistent spacing. A dense-list row SHALL show `ARTIST · DATE` beneath its title, SHALL show separate School, Form, and Type columns at the reference large breakpoint, and, when result actions are shown, SHALL place the same actions in a vertically stacked trailing block; the three repeated metadata columns SHALL be hidden below that breakpoint. Both presentations SHALL provide separate, state-aware `ADD TO ITINERARY +` and `ADD TO STUDY BOARD +` controls that do not activate the artwork-record link. Result actions SHALL be hidden by default and SHALL be shown only while the visitor's actions setting is on. The sort/view toolbar SHALL provide an `ACTIONS +` toggle that shows the actions on every grid card and dense row and reads `ACTIONS ✓` with a pressed state while on; the toggle SHALL not be offered as working when its script is unavailable. Available controls SHALL strengthen their border and gain the appropriate faint tint on hover, while already-added or full controls SHALL remain visually inert without allowing activation to fall through to the record.

#### Scenario: Visitor compares grid and dense-list results

- **WHEN** the same artwork is rendered in grid and dense-list views
- **THEN** both views identify its title, filing-form artist, and date, the grid includes its school and type, and the large dense row includes its school, form, and type without displaying technique in place of a required value.

#### Scenario: Visitor shows and hides result actions

- **WHEN** a visitor activates `ACTIONS +` and later activates `ACTIONS ✓`
- **THEN** the itinerary and Study Board controls appear on every grid card and dense row, the toggle reports its pressed state, and the second activation hides the controls again without reloading the results.

#### Scenario: Visitor adds a search result to a workspace

- **WHEN** a visitor with result actions shown activates the itinerary or Study Board control on a grid card or dense row
- **THEN** only the selected workspace changes, the control reports its resulting present or capacity state, the artwork-record link is not followed, and repeated activation does not add a duplicate.
