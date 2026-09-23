# CI/CD Production Readiness

## Local leaves

Ordinary local Go completion uses a matching build and relevant unit tests
under [AGENTS.md](../AGENTS.md#validation-budget), not an automatic expanded
verification route. [Go Validation](validation/go.md) owns package-scoped
commands first; use the root-module suite when `go.mod`/`go.sum` changes or the
package graph cannot bound the change:

```bash
make build
make test-package PKG=./path/to/package
```

```bash
ALLOW_FULL=1 make test-all
```

Load the [Evidence Contract](spec-first-workflow/shared/evidence-contract.md)
for explicit additions, reuse, or unavailable infrastructure. Missing optional
integration or image observations do not block that criterion; known defects
still do. Do not provision or repair a test environment merely to finish.

`make verify` remains available for explicitly selected expanded verification.
It selects affected Go packages and changed contract owners, and can select
Docker, integration, migration, and image checks. `make plan` diagnoses this
route without authorizing or executing it. Valid exact receipts and scoped
results remain reusable under the Evidence Contract.

`ALLOW_FULL=1 make check` is an explicit full-repository gate: formatting,
`lint-all`, `test-all`, root-module tidy, and generated drift. `ALLOW_FULL=1 make
test-all` is only the ordinary unit-test suite. Full and heavy execution guards
remain unchanged; actual CI supplies its own authority. Do not impersonate CI.

Local completion does not assert platform admission, runtime behavior, or a
release. Existing CI/release gates below remain intact and do not need duplicate
local runs. An explicit request for green CI or a release retains that outcome
until its real result is available.

## Pull-request and push CI

[ci.yml](../.github/workflows/ci.yml) classifies the exact diff once and starts
only applicable leaves. Runtime Go, root/tool dependencies, lint config,
initializers, validation routing, workflows, dependency automation, performance harness, database,
messaging, process, race, migrations, runtime image, and image security are
separate surfaces. A root `Makefile` change selects the validation-routing
surface, not every gate. Pull-request Go tests use the reverse-importer
closure; `test-all` and standalone `gosec` remain the main/nightly oracles.
Instruction-only quality does not install Go. Gitleaks uses the tools-module
version through a checksum-pinned binary and range scans pull requests, merge
groups, and main pushes; tags and manual runs retain full-history proof.
Integration runs each selected suite (database, messaging, process, race,
image) as its own parallel leg; a suite leg starts its own containers, so the
slowest suite, not their sum, bounds the job. The image leg builds one image
only when a selected runtime gate needs it and reuses that image for
lifecycle, migration, and the canonical Make-owned vulnerability gate. The always-reported `required` job fails when any
applicable leaf fails or is cancelled and accepts deliberate path skips.
Pull requests and merge groups are path-aware; main pushes, tags, and manual
runs deliberately select the full surface set.

Performance-harness changes run a pinned k6 `inspect` and evidence-owner
self-test. That gate makes broken scenario wiring fail without starting a
service or turning noisy benchmark thresholds into merge admission.

GitHub Rulesets or organization policy own merge admission. Require `required`
and `codeql-required`; the repository does not rewrite its own protection
settings.

## Generated contracts

OpenAPI, SQLC, and Protobuf checks run the canonical generator and use a scoped
Git diff. Breaking OpenAPI and Protobuf comparisons run only for pull requests
against the event's exact base SHA. Generated output is never edited by hand.

## Secrets and dependencies

The canonical [CI workflow](../.github/workflows/ci.yml) owns scanner routing.
It retains current-worktree coverage, release/manual full-history, and
missing-base failure. [Security Validation](validation/security.md) owns the
corresponding local command selection.

Dependency Review rejects new high-severity dependencies on pull requests.
`govulncheck` runs on runtime Go pull requests. `gosec` runs inside `lint-pr`
on pull requests and as a standalone main/nightly oracle. Go CodeQL runs on
pull requests, merge groups, main, tags, schedule, and manual dispatch when
handwritten Go or root dependencies change. There is no merge queue, and
`codeql-required` is a required admission context, so pull-request Go analysis
remains a merge gate. Actions CodeQL remains pull-request scoped when workflow
source changes. These tools observe different source and dependency paths.

## Publication

[cd.yml](../.github/workflows/cd.yml) has one publication job. Main publication
consumes a successful same-repository push CI run, waits for full exact-SHA
CodeQL, and checks out that SHA. A release tag waits for its own full exact-SHA
CI and CodeQL runs; tag CI executes integration regardless of changed paths.
Publication does not rerun either suite.

Publication remains opt-in through `ENABLE_GHCR_PUBLISH=true`. The shared
[publish-image action](../.github/actions/publish-image/action.yml):

1. builds one run-scoped production candidate for the exact admitted commit;
2. preserves the previously published migration corpus;
3. rehearses migrations through the shared runtime lifecycle checker;
4. scans the image through the canonical Make target and generates a CycloneDX SBOM;
5. pushes the candidate and resolves its digest;
6. signs and attests that digest;
7. verifies signature, provenance, and SBOM attestation;
8. uploads the run-scoped SBOM artifact and records the verified digest;
9. advances the verified migration-history marker;
10. promotes the SHA/main or version/latest tags and reads their digest back.

Both main and release events use the same non-cancelling
`migration-publication-${{ github.repository }}` concurrency group. Public tags
never move before verification or migration-history preservation. Promotion is
the final required step; a partial multi-tag registry failure records the exact
digest, the already-promoted tags, and the failed tag before returning failure.

## Migrations

Derived repositories enforce append-only migration files against Git history.
Publication independently compares every previously published migration byte
against the candidate image. `internal/infra/postgresmigrate` owns canonical
source admission and the PostgreSQL session lock; Goose owns source validation.

`make migration-validate` uses a disposable PostgreSQL instance, runs focused
real-database migration proof, executes the image migrator, starts the exact
runtime image under restricted container settings, waits for readiness, checks
the expected version, and verifies clean SIGTERM shutdown.

The rehearsal proves shape and runtime wiring, not data recoverability or
mixed-version compatibility. Expansion/backfill/contract policy and backup
evidence remain release decisions.

## Recovery

- Failed CI changes no external state.
- Failed integration owns its disposable Docker resources through target traps.
- A failed publication never promotes public tags before verification.
- Publication candidates are run-scoped tags; SHA, version, `main`, and `latest`
  are promoted only after verification. Rollback resolves a
  previously verified digest rather than rebuilding it.
