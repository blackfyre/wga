## ADDED Requirements

### Requirement: Keyboard navigation covers all release destinations and actions

The system SHALL register every public release destination in one server-rendered registry with its letter shortcut, two-digit section number, label, and route, while reserving J, K, H, and L for movement.

#### Scenario: Visitor uses a release section shortcut

- **WHEN** a visitor enters a registered letter or section number outside an editable control
- **THEN** the keyboard layer navigates to the registry's corresponding public route.

### Requirement: Keyboard help includes on-page release actions

The system SHALL expose the available section jumps, palette, search, list traversal, tour page turns, viewer controls, and Escape dismissal paths through the shortcut help surface.

#### Scenario: Visitor opens keyboard help

- **WHEN** a visitor activates the shortcut help control
- **THEN** the help surface names the available action and its usable key without requiring undocumented knowledge.

### Requirement: Reference navigation widgets have complete keyboard paths

The system SHALL provide native or equivalent keyboard interaction for the artist portrait-comparison carousel and every Help table-of-contents row. Carousel controls SHALL expose their destination and current state, and table-of-contents navigation SHALL use ordinary fragment links or equivalent link semantics.

#### Scenario: Visitor uses portrait comparison without a pointer

- **WHEN** a visitor focuses the portrait-comparison controls and activates the previous or next action
- **THEN** the comparison advances, announces or exposes the resulting state, and retains a visible focus path.

#### Scenario: Visitor follows the Help table of contents

- **WHEN** a visitor activates a table-of-contents row with Enter
- **THEN** the corresponding section receives browser-native fragment navigation without requiring a pointer-specific listener.
