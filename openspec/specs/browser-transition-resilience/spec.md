# browser-transition-resilience Specification

## Purpose

Preserve HTMX navigation when a native View Transition animation is skipped, while keeping DOM update failures and unrelated browser errors observable.

## Requirements

### Requirement: Skipped browser animations preserve navigation

When a native View Transition cannot capture its animation and its ready promise
rejects with InvalidStateError, the application SHALL allow the successful DOM
update to complete without reporting that animation rejection as an unhandled
error. It SHALL preserve the response, history update, and subsequent navigation,
without repeating the DOM update or request.

#### Scenario: Artwork search encounters invalid transition capture

- **WHEN** an artwork search response encounters invalid native transition capture
- **THEN** the requested results and URL SHALL update once
- **AND** subsequent searches SHALL continue to work
- **AND** the animation-only InvalidStateError SHALL NOT become an unhandled error

### Requirement: Transition handling preserves unrelated failures and fallback

The application SHALL keep callback failures, other transition rejection types,
and unrelated browser errors observable. Browsers without View Transitions SHALL
continue to support HTMX navigation, and successful transitions SHALL remain enabled.

#### Scenario: An update callback fails

- **WHEN** the transition DOM update fails, including with InvalidStateError
- **THEN** the failure SHALL remain observable

#### Scenario: Another error occurs

- **WHEN** ready rejects with another error or an unrelated browser operation raises InvalidStateError
- **THEN** the failure SHALL remain observable

#### Scenario: The API is unavailable

- **WHEN** a visitor searches artworks without View Transitions support
- **THEN** the results and URL SHALL still update

#### Scenario: The transition succeeds

- **WHEN** a native transition completes successfully
- **THEN** its animation and search update SHALL remain enabled
