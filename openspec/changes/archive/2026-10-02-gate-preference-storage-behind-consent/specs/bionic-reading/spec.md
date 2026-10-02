## MODIFIED Requirements

### Requirement: Bionic-reading preference is local and defaults off

The system SHALL keep a visitor's explicit bionic-reading choice in the `wga-bionic` browser local storage entry. It SHALL read or write that entry only while the visitor has accepted the optional `preferences` cookie-consent category. Without that consent, the choice SHALL apply to the current page, including its HTMX updates, and SHALL not be stored. Any stored copy SHALL be deleted.

When consent is first granted, the system SHALL store the state the page shows. When consent is withdrawn, the system SHALL delete the entry while the page keeps its state. A missing, inaccessible or unconsented stored choice SHALL result in the off state.

The system SHALL NOT send the choice to the server or persist it in a cookie. The legacy `wga_bionic` cookie SHALL be expired.

#### Scenario: Visitor returns with bionic reading enabled

- **WHEN** a visitor with `preferences` consent loads a public page with a stored enabled preference
- **THEN** the system applies bionic reading after client-side initialisation and the control reports the on state.

#### Scenario: Visitor has not accepted preference storage

- **WHEN** a visitor without `preferences` consent enables bionic reading and then reloads the page
- **THEN** bionic reading applied before the reload, nothing was written to local storage, and the reloaded page starts with bionic reading off.

#### Scenario: Visitor grants preference storage with bionic reading on

- **WHEN** a visitor enables bionic reading and then accepts `preferences` consent
- **THEN** the enabled state is stored and survives a reload.

#### Scenario: Browser storage is unavailable

- **WHEN** browser local storage cannot be read or written
- **THEN** the page remains usable with bionic reading off and no error is exposed to the visitor.
