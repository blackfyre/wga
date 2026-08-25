# Database Baseline Bootstrap

## Purpose

Define the current PocketBase schema baseline and synthetic fixture bootstrap for fresh WGA installations.

## Requirements

### Requirement: Fresh database baseline

The system SHALL create only the current WGA PocketBase application collections, fields, indexes, access rules, configuration, and visual thumbnail field declarations when applying project migrations to a fresh database. The `artworks.image` FileField SHALL declare `120x0`, `200x0`, `400x0`, `500x0`, `600x0`, `700x0`, `800x0`, `900x0`, `1000x0`, `1100x0`, `1400x0`, `1600x0`, and `2000x0`; the `artists.portrait` FileField SHALL declare `500x0` and `600x0`.

#### Scenario: Fresh database migration

- **WHEN** project migrations run against a new PocketBase data directory
- **THEN** the resulting application schema contains the collections defined by the database baseline, the declared artwork and portrait thumbnail variants, and no legacy synthetic-source target fields or collections.

### Requirement: Synthetic fixture import

The system SHALL import the complete embedded synthetic SQLite fixture and its required storage assets into the current PocketBase collections only when the application database contains no application records. When a supported source database provides `artists.biography_image_output_path`, the import SHALL set the corresponding artist portrait filename without transferring image bytes.

#### Scenario: Empty application database

- **WHEN** the bootstrap migration runs with no application records
- **THEN** it validates the fixture relations and assets before importing all current synthetic records, relations, and files into their corresponding PocketBase collections

#### Scenario: Source fixture provides portrait paths

- **WHEN** an empty application database is bootstrapped from a supported source fixture containing `biography_image_output_path`
- **THEN** each non-empty safe path produces the filename reference on its corresponding artist record without the bootstrap reading or uploading image bytes.

#### Scenario: Source fixture has no portrait column

- **WHEN** an empty application database is bootstrapped from a supported source fixture without `biography_image_output_path`
- **THEN** the bootstrap completes with no artist portrait references.

#### Scenario: Populated application database

- **WHEN** the bootstrap migration runs with existing application records
- **THEN** it SHALL not create or modify synthetic bootstrap records

### Requirement: Art-period bootstrap data

The system SHALL retain `Art_periods` in the PocketBase baseline and import each synthetic `art_periods` row with its identifier, name, date range, description, and generated slug.

#### Scenario: Art periods are imported

- **WHEN** the bootstrap migration completes on a fresh database
- **THEN** `Art_periods` contains the complete fixture period taxonomy

### Requirement: Current feature and model-call parity

The system SHALL preserve each current feature's PocketBase collection and field contract, and all database model calls SHALL be verified against the replacement baseline before legacy migrations are removed.

#### Scenario: Existing collection query

- **WHEN** an existing feature looks up, queries, saves, or filters a PocketBase collection after the baseline migration
- **THEN** its collection, field, relation, and index assumptions match the baseline schema or the call is updated and covered by a focused test

### Requirement: Collection naming boundary

The system SHALL name administrator-managed PocketBase collections with leading-capital snake case and WGA tracking collections with lower snake case. The `users` and PocketBase-owned system collections SHALL retain their existing names.

#### Scenario: Tracking collection baseline

- **WHEN** a fresh database applies the schema baseline
- **THEN** delivery and contributor tracking collections are named `postcard_deliveries`, `postcard_delivery_attempts`, `contributor_snapshots`, and `contributor_refresh_executions`

### Requirement: Catalogue-scale query paths

The system SHALL support approximately 5,000 artists and 56,000 artworks without materialising all matching records for pagination or performing an avoidable per-artwork author lookup during sitemap generation.

#### Scenario: Browse index baseline

- **WHEN** a fresh database applies the schema baseline
- **THEN** `Artists` defines an index on `(published, name, id)`, `Artworks` defines an index on `(published, title, id)`, and postcard claims use an index on `(status, available_at, id)`

#### Scenario: Paginated catalogue query

- **WHEN** an artist or artwork list calculates its pagination total
- **THEN** it uses a count-only query with the same filter semantics rather than loading all matching records

#### Scenario: Sitemap author lookup

- **WHEN** the sitemap processes published artworks
- **THEN** it loads required authors in batches rather than expanding one author relation per artwork
