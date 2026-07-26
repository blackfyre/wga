# Licence Notices and SBOM

`internal/licences/manifest.json` is the reviewed record of every third-party component shipped by WGA. It contains the source evidence, licence text, NOTICE material, integrity value, distribution target, and dependency relationships for the component version under review.

`cmd/generate-licences` discovers shipped Go modules with `go list -deps -json ./cmd/wga`, JavaScript packages from `dist/browser-metafile.json`, and packages imported by the browser CSS entrypoint. The normal command validates that discovery against the manifest, then writes:

- `internal/assets/views/open-source-licences.html`, embedded by the application and served at `/open-source-licences`;
- `dist/wga.cdx.json`, the CycloneDX 1.7 release artefact.

Run the normal generator after `bun run build` and `templ generate`:

```bash
go run ./cmd/generate-licences
```

When a dependency changes, rebuild the assets and bootstrap a candidate manifest:

```bash
go run ./cmd/generate-licences --bootstrap
```

Review every changed component against the recorded source evidence. Confirm the SPDX identifier, complete licence text, and any NOTICE or attribution requirements before committing the manifest and regenerated notice page. Do not edit the generated HTML or SBOM directly.

`mise run app:build` and GoReleaser run generation automatically. Release archives include `wga.cdx.json` alongside the binary.
