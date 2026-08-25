## MODIFIED Requirements

### Requirement: Fresh database baseline
The system SHALL create only the current WGA PocketBase application collections, fields, indexes, access rules, and configuration, including the optional `portrait` file field on `artists`, when applying project migrations to a fresh database.

#### Scenario: Fresh database migration
- **WHEN** project migrations run against a new PocketBase data directory
- **THEN** the resulting application schema contains the collections defined by the database baseline, including `artists.portrait`, and no legacy synthetic-source target fields or collections.

### Requirement: Synthetic fixture import
The system SHALL import the complete embedded synthetic SQLite fixture and its required storage assets into the current PocketBase collections only when the application database contains no application records. When a supported source database provides `artists.biography_image_output_path`, the import SHALL set the corresponding artist portrait filename without transferring image bytes.

#### Scenario: Empty application database
- **WHEN** the bootstrap migration runs with no application records
- **THEN** it validates the fixture relations and assets before importing all current synthetic records, relations, and files into their corresponding PocketBase collections.

#### Scenario: Source fixture provides portrait paths
- **WHEN** an empty application database is bootstrapped from a supported source fixture containing `biography_image_output_path`
- **THEN** each non-empty safe path produces the filename reference on its corresponding artist record without the bootstrap reading or uploading image bytes.

#### Scenario: Source fixture has no portrait column
- **WHEN** an empty application database is bootstrapped from a supported source fixture without `biography_image_output_path`
- **THEN** the bootstrap completes with no artist portrait references.

#### Scenario: Populated application database
- **WHEN** the bootstrap migration runs with existing application records
- **THEN** it SHALL not create or modify synthetic bootstrap records.
