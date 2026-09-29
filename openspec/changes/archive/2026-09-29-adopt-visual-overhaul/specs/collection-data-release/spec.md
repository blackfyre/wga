## ADDED Requirements

### Requirement: Release provisioning consumes the real collection data and staged storage together
The release operator SHALL follow a documented manual runbook to validate the producer manifest, pre-populate PocketBase storage from its corresponding staged tree, install the approved `wga-src` production SQLite database at the existing `WGA_SEED_SQLITE_PATH`, and record the verification evidence before authorising WGA startup. WGA SHALL retain its existing external-seed contract and SHALL NOT gain a runtime manifest setting or storage-copy responsibility.

#### Scenario: Fresh release data directory is initialised
- **WHEN** a release environment initialises an empty WGA data directory
- **THEN** the completed provisioning record shows that the paired inputs were verified and installed before WGA started, and the resulting records reference public assets already available from PocketBase storage.

### Requirement: Release media includes every source-eligible rendition
The release process SHALL verify that the thirteen approved artwork profiles are defined and that each published image has its original dimensions plus every staged downscale whose target width is smaller than the source width. It SHALL NOT require or generate an upscaled rendition.

#### Scenario: Required rendition is missing
- **WHEN** release verification finds a published image whose source width exceeds an approved target but whose corresponding staged downscale is unavailable
- **THEN** the verification fails with an actionable asset and rendition report.

#### Scenario: Target width is not smaller than the source
- **WHEN** release verification evaluates an approved target width that is greater than or equal to the published image's source width
- **THEN** it accepts the original as the only eligible asset for that target and does not require an upscaled derivative.

### Requirement: Synthetic data remains isolated to development and tests
The release provisioning runbook SHALL require the approved external database path in production while allowing WGA's existing empty-path synthetic seed behaviour to remain available for focused development and test fixtures.

#### Scenario: Production release is provisioned
- **WHEN** the release operator prepares the release data
- **THEN** the operator does not authorise WGA startup with a missing or unapproved external seed and records the configured `WGA_SEED_SQLITE_PATH` in the provisioning evidence.
