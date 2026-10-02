## MODIFIED Requirements

### Requirement: Fixed bottom surfaces share one measured stack

The system SHALL coordinate the Study Board tray, itinerary tray, cookie notice, toast container, keyboard hint bar, and page-content reservation through one measured bottom-stack contract. When multiple fixed surfaces are visible, they SHALL not overlap one another or obscure page content or focused controls. The keyboard hint bar, when displayed, SHALL be the lowest item of the stack, with the itinerary tray and then the Study Board tray above it. The FEEDBACK control and the toast container SHALL sit one shared constant gap above the total height of the stack: 16px below a 720px viewport width and 24px from 720px.

#### Scenario: Cookie notice and itinerary tray are visible together

- **WHEN** both fixed surfaces are rendered and a toast is raised
- **THEN** the Study Board shelf and itinerary tray remain adjacent in their reference order, the notice and toast clear the combined stack, and the page retains sufficient bottom space to reach its final content.

#### Scenario: Floating controls clear the docked surfaces

- **WHEN** a public page shows any combination of the keyboard hint bar, the itinerary tray, and the Study Board tray, including none of them
- **THEN** the bottom edge of FEEDBACK and of the toast container is exactly the shared gap above the top of the highest docked surface, or above the viewport's bottom edge when nothing is docked.

#### Scenario: Desktop page reserves the hint bar

- **WHEN** a desktop-pointer visitor scrolls to the end of a public page
- **THEN** the shared footer's final row remains reachable above the keyboard hint bar and any visible trays.
