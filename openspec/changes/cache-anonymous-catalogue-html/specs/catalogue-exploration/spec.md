## MODIFIED Requirements

### Requirement: Artwork search remembers presentation choices

The system SHALL remember an artwork search visitor's sort key, sort direction, result view, and result-actions setting on that visitor's device across page loads, using the `wga-aw-prefs` browser storage entry and the `wga_aw_prefs` first-party cookie. These choices are presentation state, not part of the query. Remembering is optional storage: the system SHALL read or write either copy only while the visitor has accepted the optional `preferences` cookie-consent category, SHALL delete both when that consent is absent or withdrawn, and without it SHALL apply the choices to the current page only. A visit to `/artworks` with no query parameters SHALL render with the remembered sort, direction, and view without a redirect. Any explicit query parameter SHALL take precedence over every remembered sort, direction, and view, so a shared or bookmarked search URL renders as addressed. Resetting filters SHALL leave the remembered choices unchanged. A missing, malformed, or invalid remembered value SHALL be ignored in favour of the ordinary defaults. Sort and view controls SHALL remain ordinary links that work without JavaScript. A response rendered with remembered choices applied SHALL be `private, no-store` and SHALL never be served from a shared cache.

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

#### Scenario: Remembered choices are not shared

- **WHEN** a visitor with a `wga_aw_prefs` cookie requests `/artworks`
- **THEN** the request bypasses the edge cache and the response is `private, no-store`
