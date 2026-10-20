## MODIFIED Requirements

### Requirement: Bounded public-read admission

The system SHALL apply separate admission profiles to interactive catalogue search routes and canonical catalogue detail routes. Admission SHALL combine a bounded per-client request rate with a concurrent-work limit held separately by each protected route profile, so that exhausting one profile does not reject work on another, and SHALL not create an unbounded queue or identity store.

#### Scenario: Client exceeds its route-profile rate

- **WHEN** a trusted client identity exceeds the configured request allowance for a protected route profile
- **THEN** the system rejects the request before catalogue database work with HTTP 429 and a `Retry-After` header

#### Scenario: Catalogue capacity is exhausted

- **WHEN** the configured concurrent-work capacity of a protected route profile is exhausted
- **THEN** the system rejects new work for that profile immediately with HTTP 503 and a `Retry-After` header

#### Scenario: Another profile's capacity is exhausted

- **WHEN** one protected route profile has exhausted its concurrent-work capacity and a request for a different protected profile arrives within its client allowance
- **THEN** the request is admitted if its own profile has capacity available

#### Scenario: Capacity is available

- **WHEN** a protected request is within its client allowance and concurrent capacity is available
- **THEN** the request proceeds and releases its capacity exactly once when processing finishes

#### Scenario: Agent resource misses the edge cache

- **WHEN** an authenticated request for `/agents/artists/{id}.md` or `/agents/artworks/{id}.md` reaches WGA
- **THEN** the detail admission profile protects the origin request without performing a PocketBase record query

#### Scenario: Admission identity cannot be trusted

- **WHEN** the configured trusted client-identity contract cannot resolve a valid identity
- **THEN** the protected request fails closed without trusting public forwarding headers
