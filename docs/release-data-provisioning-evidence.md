# Release data provisioning evidence (version 1)

This is a version 1, fill-in template for the manual provisioning record. It
proves that one approved `wga-src` output, database, storage tree, originals,
dimensions, eligible renditions, and artwork colour profiles belong to the
same release. It does not provision, inspect, or authorise any real
deployment. Do not put credentials, secrets, tokens, or credential-bearing
URLs in this record.

## 1. Release summary and sign-off

| Field                                          | Value |
| ---------------------------------------------- | ----- |
| Evidence format version                        | `1`   |
| Release ID                                     |       |
| Producer revision (commit/tag)                 |       |
| Producer output identifier/path                |       |
| Operator                                       |       |
| Operator timestamp (UTC)                       |       |
| Reviewer                                       |       |
| Review timestamp (UTC)                         |       |
| Final decision (`authorised`/`not-authorised`) |       |
| Attached report index                          |       |

The reviewer records the final decision only after every required section and
attached report is present. A blank, partial, or contradictory record is
`not-authorised`.

## 2. Manifest and source validation

Record the manifest filename and its SHA-256, then record each declared path
and hash exactly as validated. Paths and hashes must describe the same
approved producer output.

| Artifact | Declared path | SHA-256 | Verified path | Size (bytes) | Verification timestamp (UTC) | Result |
| -------- | ------------- | ------- | ------------: | -----------: | ---------------------------- | ------ |
| manifest |               |         |               |              |                              |        |
| database |               |         |               |              |                              |        |
| storage  |               |         |               |              |                              |        |
| schema   |               |         |               |              |                              |        |

For storage, also record the source root, ordinary-file count, total bytes,
and the manifest storage hash method (lexicographically sorted relative names
and bytes, with a NUL before and after each file). Record the schema path and
hash. The declared paths must be `out/production/wga-src.sqlite`,
`out/production/storage`, and the approved repository-root `pb_schema.json`
after canonical resolution.

## 3. Installed destinations

| Item                                             | Value | Verification method/result | Timestamp (UTC) |
| ------------------------------------------------ | ----- | -------------------------- | --------------- |
| Installed database path (`WGA_SEED_SQLITE_PATH`) |       |                            |                 |
| Installed database size (bytes)                  |       |                            |                 |
| Installed database SHA-256                       |       |                            |                 |
| Source database SHA-256 (comparison)             |       |                            |                 |
| Volume identifier / `/prod-data` confirmation    |       |                            |                 |
| Remote endpoint identifier (non-secret)          |       |                            |                 |
| Bucket                                           |       |                            |                 |
| Exact key prefix (including empty)               |       |                            |                 |
| Remote object count                              |       |                            |                 |
| Remote object bytes                              |       |                            |                 |
| Remote content-verification method and result    |       |                            |                 |

The remote result must verify content, not only names or sizes: either
recompute the storage hash from retrieved objects or compare every object with
a documented per-object SHA-256. Do not treat an ETag as SHA-256 without an
explicit provider guarantee.

## 4. Asset-level evidence (attached report)

Attach a CSV or Markdown report; do not paste thousands of rows here. The
report is required for every non-empty published original and every referenced
asset. One row per asset uses this column contract:

```text
release_id,asset_class,collection,record_id,source_key,original_output_path,
original_local_path,original_local_size,original_local_sha256,original_remote_key,
original_remote_size,original_remote_sha256,local_availability,remote_availability,
source_width,source_height,biography_image_original_output_path,portrait_artwork_id,
portrait_match_method,portrait_match_score,portrait_match_version,result,
evidence_timestamp_utc
```

For artworks, `original_output_path` is `output_image_path`; for active
portraits it is `biography_image_output_path`. Preserve and record
`biography_image_original_output_path` when it is non-empty. Record structured
portrait provenance in the five `portrait_*` columns. Unmatched portraits may
have null or empty match fields; for artwork rows all portrait-only columns
remain empty. Every non-empty published original must have positive dimensions
and content-verifiable local and remote key/hash values.

`local_availability` and `remote_availability` must each be one of
`verified-present`, `missing`, `inaccessible`, or `hash-mismatch` (use
`verified-present` only when the content was read and its hash matched the
recorded value). Startup requires `verified-present` for both fields for every
required original. Missing dimensions fail the release; runtime fallback to
the original does not waive this evidence requirement.

## 5. Rendition-level evidence (attached report)

Attach one row per asset/profile, including profiles that are not required.
The authoritative producer key formulas are:

```text
original key:  <collection>/<record-id>/<basename>
rendition key: <collection>/<record-id>/thumbs_<basename>/<profile>_<basename>
```

Require every `staged_key` to equal the rendition formula, and every
`remote_key` to equal the recorded remote prefix followed by that formula.
`pb_schema.json` defines profile names; the producer staging code defines this
key layout. Do not infer either from a local path.

```text
release_id,asset_class,collection,record_id,original_key,source_width,source_height,
target_profile,target_width,eligibility,staged_key,staged_size,staged_sha256,
remote_key,remote_size,remote_sha256,availability,evidence_timestamp_utc
```

For artworks, evaluate exactly these target widths:
`120, 200, 400, 500, 600, 700, 800, 900, 1000, 1100, 1400, 1600, 2000`.
Eligibility is strictly `target_width < source_width`. For eligible profiles,
record staged and remote keys, sizes, hashes, and availability. For
`target_width >= source_width`, record `original-only/not-required`; never
require or generate an upscale.

For portraits, evaluate the profiles configured for the portrait file field
in the approved `pb_schema.json` and record the exact profile names and target
widths found there. Do not substitute artwork widths or invent portrait
profiles. Apply the same strict eligibility and original-only rule.

## 6. Artwork colour-profile evidence (attached report)

Attach one row for each published artwork original. This requirement does not
apply to portraits.

```text
release_id,collection,record_id,output_image_path,colour_palette,colour_signature,
colour_profile_version,colour_image_hash,published_original_sha256,
profile_refers_to_published_original,result,evidence_timestamp_utc
```

`colour_image_hash` must equal the SHA-256 of the published original bytes,
and `profile_refers_to_published_original` must be true. The profile fields
are the producer's palette, signature, and version values; do not create a
portrait colour-profile requirement.

## 7. Failures and remediation (attached report)

Record every failure, including one resolved before sign-off:

```text
release_id,asset_class,collection_or_record,source_or_original_key,target_profile,
expected,actual,failure_code,failure_message,remediation_owner,remediation_status,
evidence_timestamp_utc
```

Use actionable codes covering at least: `missing-unapproved-seed`,
`hash-mismatch`, `missing-original`, `invalid-dimensions`,
`missing-eligible-rendition`, `unexpected-inaccessible-remote-object`,
`invalid-mismatched-colour-profile`, and `upscale-present-or-generated`.
Unresolved failures make the final decision `not-authorised`.

## 8. Startup authorisation

The operator and reviewer confirm that the decision is fail-closed. Startup
is authorised only when all sections and their attached reports exist, the
manifest/database/schema/storage hashes agree, source and installed database
evidence agrees, remote content verification succeeds, every required
original and eligible rendition is available, no upscale is present or
required, artwork profiles match their originals, and no unresolved failure
remains. Any missing section/report, differing hash, or differing content
verification result requires `not-authorised`.

| Role     | Name | Decision | Signature/reference | Timestamp (UTC) |
| -------- | ---- | -------- | ------------------- | --------------- |
| Operator |      |          |                     |                 |
| Reviewer |      |          |                     |                 |
