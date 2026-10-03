# Spec Delta

## MODIFIED Requirements

### Requirement: Cookie consent retains Vanilla CookieConsent semantics

The system SHALL display Vanilla CookieConsent using the reference's notice treatment while retaining the existing client-side necessary-consent category, persistence, and preferences behaviour. The system SHALL also offer an optional `preferences` category, off by default. That category SHALL gate every remembered preference: the artwork search toolbar choices, the colour scheme, the palette, the bionic-reading aid, and the Study Board. The notice and the preferences interface SHALL state that this optional storage exists and name what it remembers. They SHALL also state that `DENY`, or switching the category off, deletes what it stored.

The notice SHALL offer exactly three keyboard-operable actions, in this reading and focus order:

- `ACCEPT ALL` accepts every category, including `preferences`.
- `DENY` accepts the necessary category only, leaves optional storage off, and deletes any stored copy.
- `PREFERENCES` opens the preferences interface.

The notice SHALL stack its text above its actions at every viewport width. `ACCEPT ALL` SHALL be the primary action, filled with the accent colour. `DENY` and `PREFERENCES` SHALL be outlined secondary actions at the same size as the primary action. Every action SHALL be at least 44px tall. The actions SHALL wrap instead of switching layout at a breakpoint. When the notice is wide enough, all three SHALL share one row. On a narrow notice, `ACCEPT ALL` and `DENY` SHALL share the first row and `PREFERENCES` SHALL take the full width beneath them.

The preferences interface SHALL offer `ACCEPT ALL`, `DENY`, and `SAVE PREFERENCES` alongside a per-category toggle, laid out like the notice's actions. It SHALL open from the notice's `PREFERENCES` action and from the footer's "Cookie settings" link. It SHALL appear as a full-height panel at the right edge of the viewport on desktop and as a full-screen sheet on mobile. While it is open, the rest of the page SHALL be inert, and focus SHALL move to its labelled `CLOSE` control. `Esc` SHALL close it. When the visitor closes it before making any consent choice, the notice SHALL be shown again.

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

#### Scenario: Notice actions wrap on a narrow screen

- **WHEN** the notice is shown at a 390px viewport width
- **THEN** its text sits above its actions, `ACCEPT ALL` and `DENY` share the first row at equal size, `PREFERENCES` spans the full width beneath them, and every action is at least 44px tall.

#### Scenario: Notice actions share one row on desktop

- **WHEN** the notice is shown at a 1440px viewport width
- **THEN** its text sits above its actions, all three actions share one row in the order `ACCEPT ALL`, `DENY`, `PREFERENCES`, and only `ACCEPT ALL` is filled.

#### Scenario: Visitor opens cookie preferences from the footer

- **WHEN** a visitor activates "Cookie settings" in the footer on a desktop viewport
- **THEN** the preferences interface opens as a full-height panel at the right edge, focus is on its `CLOSE` control, and the page behind it cannot receive focus or pointer input.

#### Scenario: Preferences interface on a mobile viewport

- **WHEN** a visitor opens the preferences interface at a 390px viewport width
- **THEN** it covers the full viewport.

#### Scenario: Visitor closes preferences without choosing

- **WHEN** a visitor who has made no consent choice opens the preferences interface from the notice and closes it with `CLOSE` or `Esc`
- **THEN** no consent is recorded and the notice is shown again.

### Requirement: Public preferences are available in the footer

The system SHALL expose one shared-footer trigger whose label states the currently applied palette, light/dark scheme, and reading mode, and SHALL open an accessible preferences panel containing those site-wide choices. The panel SHALL apply initial focus and invoker restoration only when its open state changes, preserving the visitor's focus and scroll position through unrelated preference updates. The system SHALL not present one ever-widening inline footer control per preference or claim a client-only control works when its required script is unavailable.

The panel SHALL end with a storage note that follows the `preferences` consent state. While preference storage is off, the note SHALL say that the choices last for this visit only and can be kept on this device by allowing preference storage in Cookie settings. While it is on, the note SHALL say that the choices are kept on this device. In both states, the note SHALL say that nothing in the panel is sent to the archive. The note SHALL update when consent changes while the page is open.

#### Scenario: Visitor chooses dark appearance

- **WHEN** a visitor explicitly selects DARK
- **THEN** subsequent rendered public pages use the dark half of the selected palette without a light-theme or wrong-palette flash.

#### Scenario: Visitor changes a preference in an open panel

- **WHEN** a visitor changes a preference after scrolling the open preferences panel
- **THEN** the update does not move focus or reset the panel's scroll position.

#### Scenario: Open preferences panel receives an unrelated update

- **WHEN** a client or HTMX lifecycle update leaves the preferences panel open
- **THEN** the update does not move focus or reset the panel's scroll position.

#### Scenario: Storage note without consent

- **WHEN** a visitor without `preferences` consent opens the preferences panel
- **THEN** its note says that the choices last for this visit only.

#### Scenario: Storage note after granting consent

- **WHEN** a visitor accepts `preferences` consent and then opens the preferences panel on the same page
- **THEN** its note says that the choices are kept on this device.

## ADDED Requirements

### Requirement: The privacy policy describes cookie and storage use accurately

The `privacy-policy` static page SHALL describe cookie and browser-storage use in line with the consent model. Essential cookies SHALL be described as covering the session and the consent record. Remembered reading preferences SHALL be described as optional storage on the visitor's device, used only with the visitor's consent and deleted by `DENY` or by switching preference storage off. The page SHALL state that no advertising or cross-site tracking cookies are set. It SHALL NOT claim that cookies record the pages a visitor accessed or customise content by browser type. It SHALL NOT mention a period-music cookie. Every other section of the page SHALL be left unchanged.

#### Scenario: Visitor reads the privacy policy's cookie section

- **WHEN** a visitor opens `/pages/privacy-policy`
- **THEN** its cookie section names the session and consent record as essential, describes reading preferences as optional consent-gated storage on the device, and states that no advertising or cross-site tracking cookies are set.

#### Scenario: Other privacy policy sections are preserved

- **WHEN** the privacy policy's cookie section is updated on an existing database
- **THEN** every other section of the page keeps its existing content.
