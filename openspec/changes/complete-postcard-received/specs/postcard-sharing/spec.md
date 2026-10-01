## ADDED Requirements

### Requirement: Received postcard leads back into the collection

The recipient postcard page SHALL link the selected work's title and a `VIEW IN GALLERY →` action to the work's canonical public artwork record. It SHALL offer `SEND YOUR OWN →`, which opens the postcard composer for the same work, alongside `BROWSE THE ARCHIVE →`. These links SHALL be ordinary navigations that work without JavaScript. The card SHALL show the work's dimensions and holding location when the artwork record provides them, preferring the record's current holding location, and SHALL omit each detail that the record does not provide.

#### Scenario: Recipient opens the work from a postcard

- **WHEN** a recipient opens a valid postcard URL and follows the artwork title or `VIEW IN GALLERY →`
- **THEN** the application SHALL navigate to the canonical public record of the postcard's work.

#### Scenario: Recipient starts their own postcard

- **WHEN** a recipient follows `SEND YOUR OWN →`
- **THEN** the application SHALL open the postcard composer with the same work selected and no sender, recipient or message values carried over.

#### Scenario: Recipient navigates without JavaScript

- **WHEN** JavaScript is unavailable on the recipient page
- **THEN** every link on the card and below it SHALL still navigate to its destination.

#### Scenario: Artwork record lacks dimensions or location

- **WHEN** the selected work's record provides no dimensions or no holding location
- **THEN** the card SHALL omit the missing detail without rendering a placeholder or an empty line.
