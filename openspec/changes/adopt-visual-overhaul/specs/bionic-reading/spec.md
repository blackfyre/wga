## MODIFIED Requirements

### Requirement: Bionic-reading preference is remembered and defaults off
The system SHALL persist a visitor's explicit bionic-reading choice in browser local storage and a `wga_bionic` cookie. A missing or inaccessible stored choice SHALL result in the off state. The cookie SHALL allow the server-rendered footer control to report an existing preference, and the client transform SHALL remain unavailable when JavaScript has not initialised it.

#### Scenario: Visitor returns with bionic reading enabled
- **WHEN** a visitor loads a public page with an enabled bionic-reading preference
- **THEN** the server marks the footer control from the cookie and the client applies the prose transform after initialisation.

#### Scenario: JavaScript is unavailable
- **WHEN** a visitor loads a public page without JavaScript
- **THEN** prose remains server-rendered and unmodified and no unusable bionic-reading control is presented.
