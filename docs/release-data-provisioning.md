# Release data provisioning

This runbook is for the current Railway deployment: WGA uses the `/prod-data`
volume for its data directory and S3-compatible PocketBase storage for uploaded
files. It is a manual release step. It does not add a WGA command, runtime
manifest setting, or storage-copy responsibility.

Do not start WGA until every check below succeeds. In particular, startup is
not authorised until the producer and installed SQLite SHA-256 values match
and storage pre-population verification succeeds.

## Inputs and destination

Obtain the approved, paired output from `wga-src`:

```text
out/production/wga-src.sqlite
out/production/storage/
the `schema.path` named by `out/production/source-bundle-manifest.json`
out/production/source-bundle-manifest.json
```

The manifest is producer-owned format 1. Its fields are `format_version`,
`database`, `storage`, and `schema`; each artifact has `path` and `sha256`.
The database and schema hashes are SHA-256 of the file bytes. The storage hash
is SHA-256 over the lexicographically sorted relative file names and bytes,
with a NUL after each relative name, before and after each file's bytes.
Symbolic links and non-regular files are invalid storage inputs.

The installed database path is the existing `WGA_SEED_SQLITE_PATH` value. In
Railway, install it on the mounted `/prod-data` volume and configure that
variable to the resulting absolute path. Do not invent or configure
`WGA_SEED_MANIFEST_PATH`.

## 1. Validate the source bundle (before any destination change)

Run from a checkout containing the producer output. The final producer task
invokes the manifest writer with `out/production/wga-src.sqlite`,
`out/production/storage`, and the repository-root `pb_schema.json`, and writes
`out/production/source-bundle-manifest.json`. The manifest's three paths remain
authoritative, but must equal those approved paths after canonical resolution.
Do not relocate `pb_schema.json` into `out/production`.

Use this fail-closed validator. It does not use `eval` or process substitution:
JSON parsing, path checks, file checks, file hashes, and the storage hash all
occur in one process. It prints the three canonical paths only after all checks
pass. The storage-root ordinary-directory requirement is a stricter release
precondition than every possible `filepath.WalkDir` root edge case; nested
symlink/non-regular rejection and the hash framing match the producer.

```bash
python3 - "$PWD" <<'PY'
import hashlib, json, pathlib, stat, sys

checkout = pathlib.Path(sys.argv[1]).resolve()
manifest_path = checkout / "out/production/source-bundle-manifest.json"
expected = {
    "database": pathlib.PurePosixPath("out/production/wga-src.sqlite"),
    "storage": pathlib.PurePosixPath("out/production/storage"),
    "schema": pathlib.PurePosixPath("pb_schema.json"),
}
try:
    manifest = json.loads(manifest_path.read_text())
    if not isinstance(manifest, dict):
        raise ValueError("manifest must be a JSON object")
    if manifest.get("format_version") != 1:
        raise ValueError("format_version must be 1")
    resolved = {}
    for name, approved in expected.items():
        artifact = manifest[name]
        value = artifact["path"]
        digest = artifact["sha256"]
        if not isinstance(value, str) or not isinstance(digest, str):
            raise ValueError(f"{name} path and sha256 must be strings")
        if not digest.isascii() or len(digest) != 64 or any(c not in "0123456789abcdef" for c in digest):
            raise ValueError(f"{name} sha256 is not lowercase hexadecimal")
        if not value or any(ord(c) < 32 or ord(c) == 127 for c in value):
            raise ValueError(f"{name} path is empty or contains a control character")
        supplied = pathlib.PurePosixPath(value)
        if supplied.is_absolute() or ".." in supplied.parts or supplied != approved:
            raise ValueError(f"{name} path is not the approved relative path")
        candidate = checkout / pathlib.Path(*approved.parts)
        if candidate.is_symlink():
            raise ValueError(f"{name} path is a symbolic link")
        actual = candidate.resolve(strict=True)
        if not actual.is_relative_to(checkout):
            raise ValueError(f"{name} path resolves outside the producer checkout")
        resolved[name] = actual
    if not stat.S_ISREG(resolved["database"].stat().st_mode) or not stat.S_ISREG(resolved["schema"].stat().st_mode):
        raise ValueError("database and schema must be regular files")
    root = resolved["storage"]
    if root.is_symlink() or not stat.S_ISDIR(root.lstat().st_mode):
        raise ValueError("storage root must be an ordinary directory")
    files = []
    for path in root.rglob("*"):
        mode = path.lstat().st_mode
        if stat.S_ISLNK(mode):
            raise ValueError(f"storage contains a symbolic link: {path}")
        if stat.S_ISDIR(mode):
            continue
        if not stat.S_ISREG(mode):
            raise ValueError(f"storage contains an unsupported entry: {path}")
        files.append(path)
    files.sort(key=lambda path: path.relative_to(root).as_posix())
    storage_hash = hashlib.sha256()
    for path in files:
        storage_hash.update(path.relative_to(root).as_posix().encode() + b"\0")
        storage_hash.update(path.read_bytes())
        storage_hash.update(b"\0")
    for name, path in (("database", resolved["database"]), ("schema", resolved["schema"])):
        actual_hash = hashlib.sha256(path.read_bytes()).hexdigest()
        if actual_hash != manifest[name]["sha256"]:
            raise ValueError(f"{name} hash mismatch")
    if storage_hash.hexdigest() != manifest["storage"]["sha256"]:
        raise ValueError("storage hash mismatch")
except (KeyError, TypeError, ValueError, OSError, json.JSONDecodeError) as error:
    raise SystemExit(f"source bundle rejected: {error}")
for name in ("database", "schema", "storage"):
    print(f"{name}={resolved[name]}")
PY
```

Record the printed paths, manifest filename, all three manifest hashes, and the
verification time.

## 2. Pre-populate PocketBase storage

Before copying, resolve the deployed PocketBase storage key root from the
running deployment's PocketBase S3 configuration and the collection/file
convention. Record the endpoint identifier, bucket, and exact key prefix/root
(including whether the prefix is empty); do not infer it from a local path.
The exact source-to-key mappings are: `artworks.output_image_path` to
`artworks/<artwork-id>/<basename>`, `artists.biography_image_output_path` to
`artists/<artist-id>/<basename>`, and `music_tracks.local_path` to
`music_song/<track-id>/<basename>` under that resolved PocketBase storage root.
`artists.biography_image_original_output_path` is not imported by WGA and is
not an importer reference. Use the organisation-approved S3 client
and its read-only/dry-run facilities first. Never use a blanket-delete
operation.

Copy the **complete** `storage.path` tree named by the validated manifest,
preserving relative paths and filenames exactly.

The destination mapping is one-to-one: each file below the staged storage root
must exist at the same relative PocketBase storage key. Verify content, not
just names or sizes: either retrieve every remote object and recompute the
manifest storage hash over the retrieved tree, or retrieve/verify a documented
per-object SHA-256 for every object and compare it with the staged file bytes.
Do not treat an ETag as SHA-256 unless that provider explicitly documents that
guarantee. A missing, extra, renamed, truncated, altered, or inaccessible
object is a failed pre-start check.

## 3. Verify database-referenced assets

Before installing the database, open a copy read-only and extract every
non-empty reference used by the importer. The current source fields are
`artworks.output_image_path`, `artists.biography_image_output_path`,
and `music_tracks.local_path` in the approved producer export. For each value,
apply the importer's `path.Clean`, then reject an empty, absolute, `..`, or
`../` path.
Compare each cleaned source reference with the staged relative path and verify
that the resulting destination key exists after the resolved deployment
prefix. The complete manifest-verified storage tree still covers staged
originals and derived files; broader original/dimension/profile evidence is
task 2.5, not this check.
Record the exact producer-export extraction, row counts, and missing-path
report. Any missing or ambiguous mapping aborts provisioning; do not substitute
a basename.

## 4. Install the SQLite volume file

Stop the WGA process. On the Railway volume, copy the approved database to the
configured absolute `WGA_SEED_SQLITE_PATH` under `/prod-data`. Do not overwrite
an active database and do not start WGA during the copy. After installation:

```bash
sha256sum "$WGA_SEED_SQLITE_PATH"
```

The installed hash must equal the manifest's `.database.sha256` and the source
file's hash. Check that the path exists, is readable by the WGA process, and
refers to the intended `/prod-data` volume. Record the path, volume identifier,
file size, installed hash, and source hash. Do not record credentials.

## 5. Pre-start decision and evidence

Complete the [version 1 release provisioning evidence template](release-data-provisioning-evidence.md)
alongside these checks. It defines the attached reports and the fail-closed
startup decision; it is a record to complete for a release, not evidence that
this runbook has provisioned any deployment.

The release operator may authorise WGA startup only when all of these are
recorded as successful:

- manifest format and fields validated;
- source database, schema, and storage hashes match the manifest;
- complete storage tree is present at the exact destination mapping;
- installed `WGA_SEED_SQLITE_PATH` is on `/prod-data` and its hash matches;
- no source, storage, path, permission, or connectivity check failed.

Record release identifier, manifest name, verification timestamp, source and
installed hashes, storage destination identifiers, file counts/bytes, the
operator, and links to the protected verification record. Never include S3
access keys, secrets, tokens, or connection strings containing credentials.

Abort and resolve the release instead of starting WGA if any input is missing
or unapproved, any hash differs, the storage tree cannot be completely
verified, or the configured external seed path is absent. WGA's empty-path
synthetic seed remains available for focused development and tests; it is not
a production release substitute.
