## MODIFIED Requirements

### Requirement: Fresh database baseline
The system SHALL create only the current WGA PocketBase application collections, fields, indexes, access rules, configuration, and visual thumbnail field declarations when applying project migrations to a fresh database. The `artworks.image` FileField SHALL declare `120x0`, `200x0`, `400x0`, `500x0`, `600x0`, `700x0`, `800x0`, `900x0`, `1000x0`, `1100x0`, `1400x0`, `1600x0`, and `2000x0`; the `artists.portrait` FileField SHALL declare `500x0` and `600x0`.

#### Scenario: Fresh database migration

- **WHEN** project migrations run against a new PocketBase data directory
- **THEN** the resulting application schema contains the collections defined by the database baseline, the declared artwork and portrait thumbnail variants, and no legacy synthetic-source target fields or collections.
