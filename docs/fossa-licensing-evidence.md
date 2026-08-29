# FOSSA Licensing Evidence

This document records the technical evidence for the FOSSA licensing findings reported on 29 August 2026. It is not legal advice and does not approve, ignore, or otherwise resolve any FOSSA issue.

## Reviewed FOSSA revision

- Project locator: `custom+40901/git@github.com:blackfyre/wga.git`
- Revision: `dependency-update-2026-08-29`
- Dependency origins: both reviewed modules are discovered from `go.mod`.

| Issue IDs | Module | FOSSA licence finding |
| --- | --- | --- |
| 20417934–20417941, 20417943 | `modernc.org/libc@v1.75.6` | GPL-2.0-or-later, LGPL-2.0-only, GPL-3.0-with-GCC-exception, GPL-3.0-or-later, LGPL-2.1-or-later, GPL-2.0-only, LGPL-3.0-only, LGPL-2.1-only, GPL-3.0-only |
| 20417942 | `modernc.org/sqlite@v1.57.0` | GPL-3.0-or-later |

FOSSA's authoritative match record is available to authorised reviewers through each issue URL or the revision dependency API with `includeMatches=true`. Do not copy the API token into a command history or document.

```text
GET /api/v2/revisions/
  custom%2B40901%2Fgit%40github.com%3Ablackfyre%2Fwga.git%24dependency-update-2026-08-29/
  dependencies?includeMatches=true&count=100
```

## Linux/amd64 build selection

WGA's release configuration builds a `linux/amd64` binary with `CGO_ENABLED=0`. The following command identifies the Go files selected for that target:

```sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go list -f '{{.ImportPath}}: {{join .GoFiles " "}}' \
  modernc.org/libc/uuid/uuid modernc.org/libc/sys/types
```

It selects these FOSSA-matched files:

| File | FOSSA licence groups containing the file | Build status |
| --- | --- | --- |
| `modernc.org/libc@v1.75.6/uuid/uuid/uuid_linux_amd64.go` | GPL-3.0-with-GCC-exception | Compiled |
| `modernc.org/libc@v1.75.6/sys/types/types_linux_amd64.go` | LGPL-2.0-only | Compiled |
| `modernc.org/sqlite@v1.57.0/lib/sqlite_g_0000000000060000.go` | GPL-3.0-or-later | Not selected for Linux/amd64 |

FOSSA also matches many files in `modernc.org/libc` packages and platform variants that are not selected by WGA's Linux/amd64 dependency graph. Their presence does not negate the compiled matches above.

The reviewed module source identifies `modernc.org/libc` and `modernc.org/sqlite` as BSD-3-Clause at their top-level `LICENSE` files. `modernc.org/libc` also includes third-party notices. Those records are evidence about upstream licensing, not a conclusion about the obligations of the compiled generated files.

## Deployment targets

| Target | Evidence | Status |
| --- | --- | --- |
| Release archive | `.goreleaser.yaml` sets `linux`, `amd64`, and `CGO_ENABLED=0`. | Reviewed |
| Railway UAT service | The service deploys `blackfyre/wga` from `main` with Railpack. Railway configuration does not expose its CPU architecture. | Architecture confirmation outstanding |

No FOSSA issue exception, policy approval, licence correction, or credentialed CI policy gate may rely on this document until a legal review covers the compiled files and every shipped target.

## Dependency upgrades

When either reviewed `modernc` module changes version, repeat the FOSSA-match retrieval and build-selection review. Do not treat this evidence as applying to the new version.
