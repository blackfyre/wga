# Spec Delta

## ADDED Requirements

### Requirement: Dual Mode embedded record links honour pane routing

Within a Dual Mode pane, every link in artist biography HTML and selection commentary HTML whose destination is supported pane content (an artist, artwork, or selection record route) SHALL open that destination in the pane selected by the originating pane's link-routing setting, preserving the other pane and all pane state in the resulting `/dual-mode` URL. Such a link SHALL work as an ordinary link without JavaScript. Links to other destinations SHALL remain unchanged. The link text and surrounding biography or commentary content SHALL be otherwise unchanged.

#### Scenario: Biography link opens in the opposite pane

- **WHEN** the left pane shows an artist whose biography links to another artist and the left pane routes links to the right window
- **THEN** activating that link loads the linked artist in the right pane, the left pane keeps its artist, and the URL remains `/dual-mode` with both panes represented

#### Scenario: Biography link opens in the same pane

- **WHEN** the left pane routes links to itself and the visitor activates a biography link to another artist
- **THEN** the left pane shows the linked artist and the right pane is unchanged

#### Scenario: Commentary link follows pane routing

- **WHEN** a selection's commentary shown in a pane links to an artwork record
- **THEN** activating it loads that artwork in the pane chosen by the pane's link-routing setting

#### Scenario: Link works without JavaScript

- **WHEN** JavaScript is disabled and a visitor follows a biography record link in a pane
- **THEN** the browser navigates to the `/dual-mode` URL that shows the linked record in the routed pane

#### Scenario: External link is untouched

- **WHEN** biography HTML contains an external `https://` link
- **THEN** the rendered link keeps its original destination and is not routed to a pane
