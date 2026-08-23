# CI/CD Production Readiness

## Local leaves

Use only the command that observes the changed claim:

```bash
make check
```

`make check` is the one local full-repository owner: format, `lint-all`,
`test-all`, module tidy, and generated-contract drift. Package iteration uses
`go test -vet=off ./internal/<package>` or `make unit-check`. Heavy leaves
(`template-init-check`, `govulncheck`, `gosec`, `verify`) require
`ALLOW_HEAVY=1` or CI.

Real PostgreSQL, migration rehearsal, runtime images, and image scanning remain
separate Docker-backed leaves. A host-only result does not prove them.

## Pull-request and push CI

[ci.yml](../.github/workflows/ci.yml) exposes three stable contexts:

- `quality`: `make check` plus path-gated initializer and instruction proof;
- `security`: dependency review, Go security, and gitleaks;
- `delivery`: actionlint, ShellCheck, and BuildKit Dockerfile checks.

[integration.yml](../.github/workflows/integration.yml) is selected by native
GitHub path filters for PostgreSQL, migrations, integration tests, the runtime
image, and their direct owners. It runs real PostgreSQL tests, builds one
production image, rehearses migrations and runtime lifecycle against that exact
image, then scans it.

GitHub Rulesets or organization policy own merge admission. Require the three
CI contexts, CodeQL, and the integration workflow where repository policy makes
it applicable. The repository does not rewrite its own protection settings.

## Generated contracts

OpenAPI, SQLC, and Protobuf checks run the canonical generator and use a scoped
Git diff. Breaking OpenAPI and Protobuf comparisons run only for pull requests
against the event's exact base SHA. Generated output is never edited by hand.

## Secrets and dependencies

Pull requests run redacted gitleaks against the current tree and the exact
base-to-HEAD range. Push and release admission use full history. Missing base
authority fails instead of widening or silently skipping the intended range.

Dependency Review rejects new high-severity dependencies on pull requests.
`govulncheck`, `gosec`, and CodeQL remain independent because they observe
different source/dependency paths.

## Publication

[cd.yml](../.github/workflows/cd.yml) has one publication job. Main publication
consumes a successful same-repository push CI run and checks out its exact SHA.
A release tag first reads back successful push CI for the exact tag SHA. It
does not rerun the suite.

Publication remains opt-in through `ENABLE_GHCR_PUBLISH=true`. The shared
[publish-image action](../.github/actions/publish-image/action.yml):

1. builds one production image;
2. preserves the previously published migration corpus;
3. rehearses migrations and runtime lifecycle;
4. scans the image and generates a CycloneDX SBOM;
5. pushes the candidate and resolves its digest;
6. signs and attests that digest;
7. verifies signature, provenance, and SBOM attestation;
8. advances the verified migration-history marker;
9. promotes mutable tags and reads their digest back.

Both main and release events use the same non-cancelling
`migration-publication-${{ github.repository }}` concurrency group. Public tags
never move before verification or migration-history preservation.

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
- Publication candidates are immutable run/SHA tags; rollback resolves a
  previously verified digest rather than rebuilding it.
