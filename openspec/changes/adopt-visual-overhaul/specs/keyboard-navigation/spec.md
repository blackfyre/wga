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
