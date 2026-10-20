## MODIFIED Requirements

### Requirement: Protection decisions are observable

The system SHALL emit structured telemetry for rejected admission, observe-mode admission decisions, and abandoned protected work, including the route profile, decision or stage, status, and current capacity without including a raw client address or requested record slug. Persisted events SHALL be bounded: for each event type and route profile, the system SHALL emit at most one event per sampling interval and SHALL report the number of events suppressed since the previous emission. Admission metrics SHALL continue to count every decision.

#### Scenario: Request is rate limited

- **WHEN** admission rejects a protected request for exceeding its client allowance
- **THEN** a structured event records the rate-limit decision and route profile without raw client data, unless an event of the same type and profile was already emitted in the current interval

#### Scenario: Request is load shed

- **WHEN** admission rejects a protected request because concurrent capacity is exhausted
- **THEN** a structured event records the capacity decision and bounded utilisation values, unless an event of the same type and profile was already emitted in the current interval

#### Scenario: Request cancellation is observed

- **WHEN** a protected workflow stops at a cancellation checkpoint
- **THEN** a structured event records the cancellation stage and route profile without reporting it as a server fault, unless an event of the same type and profile was already emitted in the current interval

#### Scenario: Events are suppressed during a flood

- **WHEN** further events of the same type and profile occur within the sampling interval
- **THEN** they are not persisted individually, the next emitted event of that type and profile reports how many were suppressed, and admission metrics count every decision

## ADDED Requirements

### Requirement: Public request volume does not amplify persistent log writes

The system SHALL NOT persist a request activity log for a request that completes without an error and with a status below 500, or that admission rejects as an expected outcome, and SHALL retain persisted logs for at most 7 days. Requests that fail with an error or end in any other 5xx response SHALL continue to be persisted.

#### Scenario: Public request succeeds

- **WHEN** a public request completes without error
- **THEN** no request activity log record is persisted

#### Scenario: Admission rejects a public request

- **WHEN** admission rejects a protected request with a 403, 421, 429, or 503 response
- **THEN** no request activity log record is persisted for it

#### Scenario: Request fails with an error

- **WHEN** a request handler returns an error
- **THEN** the request activity log record is persisted

#### Scenario: Handler renders a server fault

- **WHEN** a request handler renders a 5xx response without returning an error
- **THEN** the request activity log record is persisted

#### Scenario: Logs age beyond retention

- **WHEN** a persisted log record is older than 7 days
- **THEN** it is eligible for PocketBase's scheduled log cleanup
