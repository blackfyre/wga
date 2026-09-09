# Dependency Update and First Release

Use this skill when upgrading a project's dependency graph and preparing or validating its first deployment of that upgraded graph.

## Scope

Treat dependency updates as a compatibility change, not a lockfile-only edit. Include:

- application and build-tool dependencies;
- required language/runtime/toolchain versions;
- lockfiles, generated assets, notices, SBOMs, and compliance evidence owned by the repository;
- deployment build arguments and environment variables that can override repository defaults.

Do not upgrade unrelated infrastructure, change licence policy, or deploy without explicit user authorisation.

## Workflow

1. **Establish the contract.** Read the active change/specification, package manifests, lockfiles, build definitions, and documented verification commands. Record the starting deployed version and current dependency/toolchain constraints.
2. **Audit before changing.** Identify available updates and separate compatible major-version paths from breaking upgrades. Read the current migration notes for dependencies whose APIs, runtime requirements, or generated configuration may change.
3. **Update deterministically.** Use the ecosystem's package manager to update manifests and lockfiles together. Let the compiler and focused tests identify required compatibility changes; do not hand-edit lockfiles.
4. **Align toolchains.** When a dependency raises the minimum language/runtime version, update every owned build definition together: local version manager, Docker image, CI, and release configuration. Check deployment variables and build arguments separately, because they may override Dockerfile defaults.
5. **Regenerate owned artefacts.** Build browser assets and templates, then regenerate licensing notices, SBOMs, or other dependency-derived artefacts. For version-specific compliance reviews, obtain fresh evidence for the final graph rather than reusing prior findings.
6. **Verify in layers.** Run formatter, compilation/type checking, focused tests, full test suite, static analysis, and the production image build. Run browser checks against a freshly seeded environment when fixtures or frontend tooling changed.
7. **Release safely.** Inspect deployment build logs for the actual selected toolchain, not just the repository default. Confirm the release environment uses the intended dependency-compatible runtime before accepting a deployment. If a deployment fails, preserve the last healthy deployment and correct the smallest configuration or source cause.

## Deployment Triage

For a failed first release:

1. Identify the failed deployment, build stage, commit, service, and environment.
2. Read build logs and inspect the effective service configuration, including variable names and build overrides.
3. Compare the actual image/toolchain reported in logs with the repository's required version.
4. Correct the authoritative setting. A platform `GO_VERSION`, `NODE_VERSION`, or equivalent can override an `ARG` default in a Dockerfile.
5. Trigger or observe a new deployment, then confirm build success and service health before reporting resolution.

## Evidence Checklist

- Dependency manifests and lockfiles are in sync.
- Local, CI, container, and deployment runtime versions agree.
- Generated dependency artefacts represent the final graph.
- Full automated checks pass.
- Browser checks use the intended assets and a fresh, compatible seed.
- Deployment logs confirm the selected toolchain and successful image build.
- The deployed service is healthy; any previous failed deployment is documented with its root cause.

## Guardrails

- Do not expose credentials while inspecting deployment configuration or logs.
- Do not treat an available latest version as compatible without compiler, test, and production-image evidence.
- Do not replace production data to test a seed; use a fresh temporary data directory.
- Do not claim release success while a deployment is only queued, building, or deploying.
