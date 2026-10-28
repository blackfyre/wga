# Spec Delta

## MODIFIED Requirements

### Requirement: Artwork search remembers presentation choices

The system SHALL remember an artwork search visitor's sort key, sort direction, result view, and result-actions setting on that visitor's device across page loads, using the `wga-aw-prefs` browser storage entry and the `wga_aw_prefs` first-party cookie. These choices are presentation state, not part of the query. Remembering is optional storage: the system SHALL read or write either copy only while the visitor has accepted the optional `preferences` cookie-consent category, SHALL delete both when that consent is absent or withdrawn, and without it SHALL apply the choices to the current page only. A visit to `/artworks` with no query parameters SHALL render with the remembered sort, direction, and view without a redirect. Any explicit query parameter SHALL take precedence over the remembered sort and direction, so a shared or bookmarked search URL renders the sort it addresses. The remembered result view SHALL follow the shared listing-view rule: any artwork search page or result fragment request without a `view` parameter SHALL render the remembered view, and an explicit `view` parameter SHALL take precedence over it. Resetting filters SHALL leave the remembered choices unchanged. A missing, malformed, or invalid remembered value SHALL be ignored in favour of the ordinary defaults. Sort and view controls SHALL remain ordinary links that work without JavaScript. A response rendered with remembered choices applied SHALL be `private, no-store` and SHALL never be served from a shared cache.

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
- **THEN** the results render the filters and sort that URL addresses, the remembered sort and direction are not applied, and the remembered view applies only when the URL carries no `view` parameter.

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

## ADDED Requirements

### Requirement: Listing screens share the remembered result view

Every screen that offers a grid/list result toggle SHALL inherit and record one shared remembered result view. These screens are artwork search (`/artworks` pages and result fragments), the artist index (`/artists`), and each Dual Mode artist-index pane (`l_view` for the left pane, `r_view` for the right). The remembered view SHALL be the `view` field of the `wga_aw_prefs` preference and SHALL be read or written only under the consent rules of the artwork search presentation choices.

When a request for one of these listings carries no view parameter for that listing, the system SHALL render the remembered view if the visitor has `preferences` consent and a valid `grid` or `list` value is stored, and SHALL otherwise render the grid. An explicit view parameter for the listing SHALL take precedence over the remembered view, and an unknown explicit value SHALL render the grid. Canonical and in-page URLs SHALL omit the view parameter for a grid listing unless the visitor's remembered view is list, in which case they SHALL carry the grid parameter explicitly so paging, filtering, and sorting keep the presented view; a list listing SHALL always carry its view parameter.

Activating a view toggle on any of these screens SHALL record the view it addresses in `wga_aw_prefs` before the toggle is followed, SHALL keep every other stored field unchanged, and SHALL record nothing without `preferences` consent. A modified activation that opens another tab or window SHALL record nothing. Opening an explicit URL SHALL NOT change the remembered view. A response whose view depends on the preference cookie SHALL vary on `Cookie`, and a response to a request carrying the `wga_aw_prefs` cookie SHALL be `private, no-store`.

#### Scenario: Visitor's list choice follows them to other listings

- **WHEN** a visitor with `preferences` consent chooses the list view on artwork search and then opens `/artists` and a Dual Mode artist-index pane without view parameters
- **THEN** the artist index and the Dual Mode pane render in the list view and their toggles report list as current.

#### Scenario: Visitor without preference storage sees the grid

- **WHEN** a visitor without `preferences` consent chooses the list view on artwork search and then opens `/artists` and Dual Mode without view parameters
- **THEN** the artist index and both Dual Mode index panes render in the grid view and no `wga_aw_prefs` cookie or `wga-aw-prefs` entry exists.

#### Scenario: Explicit view parameter wins

- **WHEN** a visitor whose remembered view is list opens `/artists?view=grid` or `/dual-mode?l_view=grid`
- **THEN** the addressed listing renders as a grid, its paging, letter, filter, and sort links carry the grid parameter, and the remembered view stays list.

#### Scenario: Remembered view is unusable

- **WHEN** the remembered `view` is missing or is a value other than `grid` or `list`
- **THEN** every in-scope listing without a view parameter renders the grid and the request does not fail.

#### Scenario: Dual Mode panes default to the grid

- **WHEN** a visitor with no remembered view opens `/dual-mode` without `l_view` or `r_view`
- **THEN** both index panes render as grids and the canonical Dual Mode URL omits both view parameters, while a pane in list view adds `l_view=list` or `r_view=list`.

#### Scenario: Toggle on another screen records the view

- **WHEN** a visitor with `preferences` consent activates the LIST toggle on the artist index or in a Dual Mode pane
- **THEN** the `wga-aw-prefs` entry and the `wga_aw_prefs` cookie record `view` as `list` and keep the stored sort, direction, and actions values.

#### Scenario: Remembered view responses are not shared

- **WHEN** a request for `/artists` or `/dual-mode` carries a `wga_aw_prefs` cookie
- **THEN** the response varies on `Cookie` and is `private, no-store`.
