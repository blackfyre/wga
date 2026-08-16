## ADDED Requirements

### Requirement: Visitor can control bionic reading
The system SHALL provide an enhanced shared-footer control named “Bionic reading” that lets a visitor enable or disable bionic reading. The control SHALL be unavailable when JavaScript has not initialised the feature.

#### Scenario: Visitor enables bionic reading
- **WHEN** a visitor activates the off-state Bionic reading control
- **THEN** the control reports the on state and eligible rendered prose is transformed.

#### Scenario: Visitor disables bionic reading
- **WHEN** a visitor activates the on-state Bionic reading control
- **THEN** the control reports the off state and transformed prose is restored to its original text.

#### Scenario: JavaScript is unavailable
- **WHEN** a visitor loads a public page without JavaScript
- **THEN** the Bionic reading control is not presented and prose remains server-rendered and unmodified.

### Requirement: Bionic-reading preference is local and defaults off
The system SHALL persist a visitor's explicit bionic-reading choice in browser local storage. A missing or inaccessible stored choice SHALL result in the off state. The system SHALL NOT send the choice to the server or persist it in a cookie.

#### Scenario: Visitor returns with bionic reading enabled
- **WHEN** a visitor loads a public page with a stored enabled preference
- **THEN** the system applies bionic reading after client-side initialisation and the control reports the on state.

#### Scenario: Browser storage is unavailable
- **WHEN** browser local storage cannot be read or written
- **THEN** the page remains usable with bionic reading off and no error is exposed to the visitor.

### Requirement: Bionic reading preserves and scopes prose content
The system SHALL present bionic reading by visually bolding the opening portion of words in eligible prose text while retaining the same textual content. It SHALL apply only to running-prose paragraphs and explicitly marked bionic-reading regions, and SHALL exclude navigation, footer content, figures, monospace labels, form controls, code, text within existing `b` or `strong` elements, and any `data-bionic="off"` subtree.

#### Scenario: Eligible paragraph is transformed
- **WHEN** bionic reading is enabled for a page containing an eligible paragraph
- **THEN** each eligible word has a visually bold opening portion and the paragraph's textual content is unchanged.

#### Scenario: Existing emphasis is preserved
- **WHEN** an eligible prose region contains excluded navigation, footer, figure, monospace, form-control, code, `b`/`strong`, or `data-bionic="off"` content
- **THEN** that excluded text is not transformed.

#### Scenario: Preference is disabled after transformation
- **WHEN** a visitor disables bionic reading after eligible prose was transformed
- **THEN** every transformed region contains its original text without bionic-reading markup.

### Requirement: Bionic reading follows HTMX content updates
The system SHALL apply the enabled bionic-reading preference to eligible prose inserted by an HTMX content update without reprocessing already transformed content.

#### Scenario: HTMX inserts eligible prose while enabled
- **WHEN** an HTMX update inserts eligible prose while bionic reading is enabled
- **THEN** the inserted prose is transformed and previously transformed prose remains unchanged.

#### Scenario: HTMX inserts eligible prose while disabled
- **WHEN** an HTMX update inserts eligible prose while bionic reading is disabled
- **THEN** the inserted prose remains unmodified.
