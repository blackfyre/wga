# Spec Delta

## ADDED Requirements

### Requirement: Enhanced navigation recovers from edge challenges

When an enhanced `GET` request receives an edge challenge response, identified by the `cf-mitigated: challenge` response header, the system SHALL NOT swap the challenge response into the page and SHALL load the requested URL as a full document, so the visitor can complete the challenge and reach the destination. The system SHALL NOT convert challenged non-`GET` requests into navigations.

#### Scenario: Navigation link is challenged

- **WHEN** a visitor follows an enhanced navigation link and the edge answers the enhanced request with a challenge
- **THEN** the browser loads the link's URL as a full page instead of leaving the current page unchanged

#### Scenario: Enhanced search request is challenged

- **WHEN** an enhanced search or filter `GET` request is answered with a challenge
- **THEN** the browser loads that request's URL, including its query, as a full page

#### Scenario: Ordinary error response

- **WHEN** an enhanced request receives a `403` or other error response without the challenge header
- **THEN** the existing error handling applies and no navigation occurs

#### Scenario: Challenged submission

- **WHEN** an enhanced non-`GET` request is answered with a challenge
- **THEN** the system does not navigate or resubmit, and the challenge response is not swapped into the page
