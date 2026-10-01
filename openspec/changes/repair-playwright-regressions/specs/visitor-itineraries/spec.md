# Spec Delta

## ADDED Requirements

### Requirement: Itinerary admission budgets are configurable
The system SHALL bound new itinerary drafts and successful itinerary publications per trusted client identity within a rolling one-hour window, using budgets loaded from deployment configuration. When no budget is configured, the draft budget and the publication budget SHALL each default to 3. A configured budget SHALL be a positive integer; the application SHALL refuse to start with a non-positive or non-integer value and SHALL report which setting is invalid. Exhausting a budget SHALL keep the existing rate-limited response and toast.

#### Scenario: Production defaults apply
- **WHEN** no admission budget is configured and a client creates a fourth new draft within an hour
- **THEN** the request is refused with the "created too many itineraries" response

#### Scenario: Raised budget for verification
- **WHEN** the draft budget is configured to 1000 and one client creates 20 drafts within an hour
- **THEN** every draft creation is admitted

#### Scenario: Invalid budget is rejected
- **WHEN** a budget setting is configured as `0` or a non-integer
- **THEN** the application does not start and names the invalid setting
