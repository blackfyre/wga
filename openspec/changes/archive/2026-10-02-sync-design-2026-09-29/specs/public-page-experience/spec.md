## MODIFIED Requirements

### Requirement: Cookie consent retains Vanilla CookieConsent semantics

The system SHALL display Vanilla CookieConsent using the reference's notice treatment while retaining the existing client-side necessary-consent category, persistence, and preferences behaviour. The system SHALL also offer an optional `preferences` category, off by default, that gates remembered artwork search choices; the notice and the preferences interface SHALL state that this optional storage exists, what it remembers, and that turning it off deletes it. The notice SHALL offer exactly three keyboard-operable actions, in this reading and focus order: `ACCEPT ALL` (every category, including `preferences`), `DENY` (the necessary category only, leaving optional storage off and deleting any stored copy), and `PREFERENCES` (opens the preferences interface). The preferences interface SHALL offer `ACCEPT ALL`, `DENY`, and `SAVE PREFERENCES` alongside a per-category toggle.

#### Scenario: Visitor accepts necessary cookies

- **WHEN** a visitor activates `DENY` in the notice
- **THEN** Vanilla CookieConsent persists consent to the necessary category only, any stored preference copy is deleted, and the initial notice is not shown again according to its existing lifecycle.

#### Scenario: Visitor accepts all categories

- **WHEN** a visitor activates `ACCEPT ALL` in the notice
- **THEN** Vanilla CookieConsent persists consent to every category, including `preferences`.

#### Scenario: Visitor opens cookie preferences

- **WHEN** a visitor selects `PREFERENCES`
- **THEN** Vanilla CookieConsent opens its preferences interface without a server-side consent request, listing the strictly necessary category and the optional preference-storage category.

## ADDED Requirements

### Requirement: Shared footer returns visitors to the top of the page

The system SHALL place a plain `↑ BACK TO TOP` text link in the shared footer's bottom row beside the preferences control. The link SHALL be an ordinary in-page link to the shared page header, SHALL work without JavaScript, and SHALL leave scrolling to the browser's CSS scroll behaviour, including its reduced-motion handling. Apart from the feedback control, the system SHALL not render a floating back-to-top control.

#### Scenario: Visitor returns to the top from the footer

- **WHEN** a visitor at the bottom of a public page activates `↑ BACK TO TOP`
- **THEN** the browser moves to the page header without a scripted interaction.
