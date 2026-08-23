# Database-Managed Head Markup Specification

## Purpose

Allow privileged operators to supply optional trusted markup for the shared HTML document head without rebuilding the WGA application.

## Requirements

### Requirement: Render the configured trusted head markup
For each full public HTML document, the system SHALL read the current `content` of the `strings` record whose name is exactly `scripts:header` and SHALL render non-empty content verbatim once inside the document `<head>`.

#### Scenario: Trusted script markup is configured
- **WHEN** `scripts:header` contains script markup and a full public HTML document is requested
- **THEN** the response contains that markup unchanged once before the closing `</head>` tag

#### Scenario: The configured value changes
- **WHEN** a privileged operator changes `scripts:header` before a subsequent full public HTML document request
- **THEN** the subsequent response uses the changed content without an application rebuild

#### Scenario: An HTMX fragment is requested
- **WHEN** a public handler returns an HTMX fragment rather than a full HTML document
- **THEN** the response does not add `scripts:header` content to the fragment

### Requirement: Header markup remains optional
The system SHALL omit database-managed head markup without failing the requested page when `scripts:header` is missing, empty, or cannot be read.

#### Scenario: The record is missing
- **WHEN** no `strings` record is named `scripts:header`
- **THEN** the system returns the requested page without injected head markup

#### Scenario: The content is empty
- **WHEN** `scripts:header` exists with empty content
- **THEN** the system returns the requested page without an empty placeholder or injected head markup

#### Scenario: The optional lookup fails
- **WHEN** reading `scripts:header` fails for a reason other than absence
- **THEN** the system logs the request-scoped failure without the record content and returns the requested page without injected head markup

### Requirement: Header markup is a privileged execution boundary
The system SHALL treat `scripts:header` as trusted executable operator content, SHALL preserve the collection's superuser-only API rules, and SHALL NOT expose a public write path for the record.

#### Scenario: Trusted active content is rendered
- **WHEN** a privileged operator stores active HTML or JavaScript in `scripts:header`
- **THEN** the system renders it without sanitisation or escaping

#### Scenario: A non-superuser attempts to change the record
- **WHEN** a guest or ordinary authenticated user attempts to create, update, or delete a `strings` record through the PocketBase API
- **THEN** the existing collection rules deny the operation
