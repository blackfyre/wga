## ADDED Requirements

### Requirement: In-page navigation scrolls smoothly and respects reduced motion

The system SHALL scroll smoothly to the destination of an in-page fragment link in the public document and inside Dual Mode pane scroll regions. When a visitor has requested reduced motion, the system SHALL NOT animate any of these scrolls.

#### Scenario: Visitor follows an in-page anchor

- **WHEN** a visitor without a reduced-motion preference activates an "ON THIS PAGE", table-of-contents, back-to-top or other fragment link to an element on the current page
- **THEN** the viewport scrolls progressively to that element, and the URL fragment updates.

#### Scenario: Reduced motion is requested for an in-page anchor

- **WHEN** a visitor who has requested reduced motion activates the same fragment link
- **THEN** the viewport moves to the element immediately, without intermediate scroll positions.

### Requirement: Page navigations start at the top without disturbing in-place updates

The system SHALL return the viewport to the top, instantly, after an enhanced navigation swaps the main area and pushes or replaces the browser URL. The system SHALL leave the scroll position unchanged for fragment and in-place updates and SHALL retain HTMX history scroll restoration.

#### Scenario: Visitor navigates from a scrolled page

- **WHEN** a visitor scrolled down a page follows a public navigation or record link that replaces the main area
- **THEN** the new page is shown from its top, without a scroll animation.

#### Scenario: Fragment or in-place update occurs

- **WHEN** global search, artwork-search filtering, in-block pagination, a Dual Mode pane, the itinerary tray, the study board shelf, a toast or a dialog updates
- **THEN** the viewport scroll position is unchanged.

#### Scenario: Visitor goes back

- **WHEN** a visitor uses browser Back after an enhanced navigation
- **THEN** the previous page is restored at its previous scroll position.

#### Scenario: Visitor opens a page section link

- **WHEN** a visitor opens a public URL with a fragment that identifies a section
- **THEN** the page lands on that section rather than at the top.
