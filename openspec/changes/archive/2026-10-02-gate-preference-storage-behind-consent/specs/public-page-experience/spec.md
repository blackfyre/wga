## MODIFIED Requirements

### Requirement: Cookie consent retains Vanilla CookieConsent semantics

The system SHALL display Vanilla CookieConsent using the reference's notice treatment while retaining the existing client-side necessary-consent category, persistence, and preferences behaviour. The system SHALL also offer an optional `preferences` category, off by default. That category SHALL gate every remembered preference: the artwork search toolbar choices, the colour scheme, the palette, the bionic-reading aid, and the Study Board. The notice and the preferences interface SHALL state that this optional storage exists and name what it remembers. They SHALL also state that `DENY`, or switching the category off, deletes what it stored.

The notice SHALL offer exactly three keyboard-operable actions, in this reading and focus order:

- `ACCEPT ALL` accepts every category, including `preferences`.
- `DENY` accepts the necessary category only, leaves optional storage off, and deletes any stored copy.
- `PREFERENCES` opens the preferences interface.

The preferences interface SHALL offer `ACCEPT ALL`, `DENY`, and `SAVE PREFERENCES` alongside a per-category toggle.

Without `preferences` consent, each remembered preference SHALL apply to the current page only. The system SHALL neither read nor write its browser storage, and SHALL delete any stored copy. When consent is first granted, the system SHALL store the state the page currently shows. When the consent library confirms consent that already existed, the system SHALL leave the remembered state unchanged. When consent is withdrawn, the system SHALL delete every stored copy while the open page keeps its current state. A grant or withdrawal made in another tab SHALL govern later reads and writes in an already-open tab.

#### Scenario: Visitor accepts necessary cookies

- **WHEN** a visitor activates `DENY` in the notice
- **THEN** Vanilla CookieConsent persists consent to the necessary category only, every stored preference copy is deleted, and the initial notice is not shown again according to its existing lifecycle.

#### Scenario: Visitor accepts all categories

- **WHEN** a visitor activates `ACCEPT ALL` in the notice
- **THEN** Vanilla CookieConsent persists consent to every category, including `preferences`, and the scheme, palette, bionic-reading and Study Board state the page shows is stored.

#### Scenario: Visitor opens cookie preferences

- **WHEN** a visitor selects `PREFERENCES`
- **THEN** Vanilla CookieConsent opens its preferences interface without a server-side consent request, listing the strictly necessary category and the optional preference-storage category with the preferences it covers.

#### Scenario: Visitor withdraws preference storage

- **WHEN** a visitor who accepted `preferences` consent switches it off and saves
- **THEN** the stored scheme, palette, bionic-reading, Study Board and search toolbar copies are deleted, and the open page keeps its current appearance and board.

#### Scenario: Consent changes in another tab

- **WHEN** a visitor grants or withdraws `preferences` consent in one tab while another tab is open
- **THEN** the other tab's next preference change is stored only if consent is now granted, and nothing that tab reads comes from storage that the withdrawal deleted.

### Requirement: Palette and light/dark scheme are independent remembered choices

The system SHALL provide the eleven reference palettes `bone`, `classic`, `verdigris`, `gothic`, `renaissance`, `baroque`, `rococo`, `classical`, `impressionist`, `catppuccin`, and `tokyo`. Each palette SHALL reproduce the complete immutable-reference interface-role, chart-series, and Timeline-lane token set without changing layout.

Palette and light/dark scheme SHALL be stored and resolved independently, in the `wga-palette` and `wga-theme` browser local storage entries only. Either entry SHALL be read or written only while the visitor has accepted the optional `preferences` cookie-consent category. An explicit choice SHALL take precedence over the operating-system scheme, and an unset scheme SHALL continue to follow operating-system changes.

Without that consent, a choice SHALL apply to the current page, including its HTMX updates. The next full page load SHALL use the default palette and the operating-system scheme. When consent is first granted, the system SHALL store the palette shown and any scheme explicitly chosen on the page. No palette or scheme cookie, request-context state, server preference DTO, or server-selected appearance control state SHALL exist. The legacy `wga_theme` and `wga_palette` cookies SHALL be expired.

The inline client head resolver SHALL apply valid local choices before stylesheet rendering to prevent a wrong-palette or wrong-scheme flash, and SHALL consult them only while the consent record accepts `preferences`.

For this change, exact clean-reference token literals control where the reference's prose contrast guidance contradicts those literals. The 53 measured token/ground exceptions SHALL remain explicitly documented rather than hidden as passing contrast checks.

#### Scenario: Visitor changes palette without changing scheme

- **WHEN** a visitor with `preferences` consent selects a different palette while using DARK
- **THEN** the selected palette's dark build is applied and remembered without changing the stored DARK choice.

#### Scenario: Visitor selects a dark-only palette

- **WHEN** a visitor selects `baroque` or `tokyo`
- **THEN** the dark-only build is applied, the LIGHT choice is visibly disabled with a reason, and the visitor's stored scheme remains unchanged for restoration after leaving that palette.

#### Scenario: Visitor has not accepted preference storage

- **WHEN** a visitor without `preferences` consent chooses DARK and a palette, then reloads the page
- **THEN** the choices applied before the reload, nothing was written to local storage, and the reloaded page uses the default palette and the operating-system scheme.

#### Scenario: A stale choice remains without consent

- **WHEN** a page loads without `preferences` consent while `wga-palette` or `wga-theme` still holds a value
- **THEN** the head resolver ignores the value, and the stored entries are deleted.

#### Scenario: Visitor identifies a palette choice

- **WHEN** the preferences panel lists palettes
- **THEN** choices are grouped by provenance and identified by label plus a paper/ink split swatch rather than colour alone.

#### Scenario: Release verifies immutable palette literals

- **WHEN** release verification compares WGA palette roles with clean reference commit `016cf6f0e93e88ce173bff3f9e8a09d854e52d35`
- **THEN** every token matches the immutable reference literal, and the known 53 contrast-floor exceptions are reported as explicit accepted exceptions rather than altering the external reference or substituting undeclared colours.

#### Scenario: JavaScript is unavailable

- **WHEN** a visitor opens a public route without JavaScript
- **THEN** the page follows the operating-system light/dark scheme, core content remains available, and unavailable manual palette or scheme controls are not presented as working.

#### Scenario: Browser storage is unavailable

- **WHEN** local storage cannot be read or written
- **THEN** the page uses the default palette and operating-system scheme without consulting or creating an appearance cookie, remains usable, and does not expose a server-derived preference state.
