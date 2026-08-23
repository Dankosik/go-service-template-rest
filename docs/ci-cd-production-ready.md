# CI/CD Production Readiness

## Local leaves

Use only the command that observes the changed claim:

```bash
make check
```

`make check` is the one local full-repository owner: format, `lint-all`,
`test-all`, module tidy, and generated-contract drift. Package iteration uses
`go test -vet=off ./internal/<package>` or `make unit-check`. Heavy leaves
(`template-init-check`, `govulncheck`, `gosec`, `audit-full-manual`) require
`ALLOW_HEAVY=1` or CI.

Real PostgreSQL, migration rehearsal, runtime images, and image scanning remain
separate Docker-backed leaves. A host-only result does not prove them.

## Pull-request and push CI

[ci.yml](../.github/workflows/ci.yml) classifies the exact diff once and starts
only applicable leaves. `quality` composes `check-go`, OpenAPI, Protobuf, SQLC,
initializer, and instruction proof; `security` runs dependency and Go scanners;
`secrets` always scans the pull-request range; `delivery` selects actionlint,
ShellCheck, and BuildKit; `integration` owns real PostgreSQL, one production
image, migration/runtime rehearsal, and Trivy. The always-reported `required`
job fails when any applicable leaf fails or is cancelled and accepts deliberate
path skips.

GitHub Rulesets or organization policy own merge admission. Require `required`
and `codeql-required`; the repository does not rewrite its own protection
settings.

## Generated contracts

OpenAPI, SQLC, and Protobuf checks run the canonical generator and use a scoped
Git diff. Breaking OpenAPI and Protobuf comparisons run only for pull requests
against the event's exact base SHA. Generated output is never edited by hand.

## Secrets and dependencies

Pull requests run redacted gitleaks against the exact base-to-HEAD range. Local
review additionally scans the current tree. Push and release admission use full
history. Every mode consumes the reviewed baseline; missing base authority
fails instead of widening or silently skipping the intended range.

Dependency Review rejects new high-severity dependencies on pull requests.
`govulncheck`, `gosec`, and CodeQL remain independent because they observe
different source/dependency paths.

## Publication

[cd.yml](../.github/workflows/cd.yml) has one publication job. Main publication
consumes a successful same-repository push CI run, waits for full exact-SHA
CodeQL, and checks out that SHA. A release tag waits for its own full exact-SHA
CI and CodeQL runs; tag CI executes integration regardless of changed paths.
Publication does not rerun either suite.

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
