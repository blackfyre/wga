## Context

FOSSA's current scan reports ten `policy_flag` licensing issues: nine for `modernc.org/libc@v1.74.1` and one for `modernc.org/sqlite@v1.54.0`. FOSSA's licence matches include `modernc.org/libc/uuid/uuid_linux_amd64.go` and `modernc.org/libc/sys/types/types_linux_amd64.go`; `go list -deps ./cmd/wga` confirms both packages are compiled into WGA's Linux/amd64 build.

The repository has no `.fossa.yml`. FOSSA therefore auto-detects `go.mod`, the root `yarn.lock`/`package.json`, and `.opencode/package-lock.json`. The Go and root frontend targets belong to WGA's application inventory; `.opencode` belongs to development tooling.

## Goals / Non-Goals

**Goals:**

- Record the evidence necessary for a future legal review of the compiled `modernc` findings.
- Make WGA's FOSSA dependency inventory deterministic and limited to application-owned dependency manifests.
- Keep WGA's generated licence notices and CycloneDX SBOM as separate, reviewed compliance artefacts.

**Non-Goals:**

- Establish a legal opinion about the compiled copyleft findings.
- Resolve or ignore the current FOSSA issues.
- Add a credentialed FOSSA CI policy gate.
- Remove or replace PocketBase's embedded SQLite driver.
- Approve GPL/LGPL licences globally or modify FOSSA's dependency licence records.

## Decisions

### Record matched compiled source without inferring a licensing outcome

Document the ten FOSSA issue IDs, affected module versions, exact licence-match paths, dependency origins, and Go package selection for every match that is compiled by WGA. The record SHALL distinguish compiled matches from source that is not selected for the reviewed target.

The previously proposed exception was rejected because FOSSA's exact matches include compiled Linux/amd64 files. The evidence record is not a legal conclusion and SHALL not be used to resolve a FOSSA finding.

### Retain WGA manifests and exclude tooling-only paths with `.fossa.yml`

Add a version-3 `.fossa.yml` that excludes `.opencode` from FOSSA target discovery. Retain the root Go and frontend manifests until a separate assessment proves that an alternative lockfile strategy produces an equivalent application inventory.

Excluding all Yarn detection was rejected because FOSSA currently identifies the root frontend dependencies through `yarn.lock` and `package.json`; doing so would reduce the scanned application inventory. Removing the root lockfile is not part of this change.

### Pin canonical repository metadata

`LICENSE.md` and `package.json` establish Apache-2.0 as the repository licence, so the README SHALL make the same declaration. The FOSSA CLI's configured project locator and both README badges SHALL use the same `custom+40901/git@github.com:blackfyre/wga.git` project, avoiding a separate legacy badge project.

### Preserve the existing shipped-artifact process

The existing licence generator and its committed manifest remain the source for WGA's shipped notices and CycloneDX file. This change verifies that the two flagged modules remain represented in those artefacts; FOSSA findings neither replace nor modify this process.

## Risks / Trade-offs

- [The evidence record could be mistaken for legal approval] → State explicitly that findings remain unresolved and prohibit an exception or CI policy gate without a future legal decision.
- [Path filtering could omit an application dependency] → Use `fossa list-targets` to identify candidate targets, then use `fossa analyze --output` to verify the filtered result preserves the root Go and frontend targets.
- [The configured FOSSA project could become stale] → Keep the locator in `.fossa.yml` and the README badges identical, and recheck both when FOSSA project ownership changes.
- [A future module version changes the legal analysis] → Require a fresh review whenever the relevant `modernc` module version changes.
- [A deployment architecture differs from the reviewed target] → Record the Railway UAT service as a separate build target requiring architecture confirmation before relying on the evidence record.

## Migration Plan

1. Record FOSSA's exact match data and the Go build selection for the current modules.
2. Add and validate the minimal analysis-scope configuration.
3. Document the evidence record, review boundary, and dependency-upgrade rule.
4. Regenerate and verify WGA's existing licence notices and CycloneDX SBOM.
5. Create a new change only after a legal conclusion determines whether an exception, policy gate, dependency upgrade, or architectural replacement is appropriate.

## Open Questions

- What legal conclusion applies to the compiled `modernc` declarations and WGA's distributed binary/container model?
- Which CPU architecture builds the Railway UAT service?
