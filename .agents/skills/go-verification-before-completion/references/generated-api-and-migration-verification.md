# Generated API And Migration Verification

## When To Load
Load this when changes touch OpenAPI specs, generated API code, generated mocks or enum strings, sqlc output, migration files, migration-driven schema behavior, or runtime API contract tests.

## Surface Rule
Generated and migration surfaces need drift or rehearsal checks in addition to ordinary tests:
- API contract changes need generation, drift, runtime contract, lint, and validation proof.
- SQL query or migration changes may need sqlc drift proof plus data-access tests.
- Migration changes need a migration rehearsal proof or an explicit environment gap.
- Generated artifact claims are not proven by tests alone, because stale generated files can still compile.

## Example Claims

| Claim | Sufficient proof | Insufficient proof |
|---|---|---|
| "OpenAPI generated code is current" | `make openapi-check` passes, or the relevant subtargets pass with no drift | `go test ./internal/api` alone |
| "Runtime API contract is green" | `make openapi-runtime-contract-check` or `go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1` | `make openapi-lint` alone |
| "API change is ready" | focused API/handler tests plus `make openapi-check`; add `make openapi-breaking` when compatibility against a base spec is in scope | only `make test` |
| "sqlc output is current" | `make sqlc-check` | `go test ./internal/infra/postgres/...` alone |
| "Generated mocks or stringer output is current" | `make mocks-drift-check` or `make stringer-drift-check`, matching the touched generator surface | `make test` alone |
| "Migration rehearsal passed" | `MIGRATION_DSN=... make migration-validate` or `make docker-migration-validate`/`make migration-validate` with Docker fallback actually running | `make migration-validate` output that says Docker and `MIGRATION_DSN` were unavailable and skipped |
| "Migration-backed behavior works" | migration rehearsal plus affected repository or integration tests | migration rehearsal alone |

## Exact Command Patterns

```bash
make openapi-generate
make openapi-drift-check
make openapi-runtime-contract-check
go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1
make openapi-lint
make openapi-validate
BASE_OPENAPI=/path/to/base.yaml make openapi-breaking
make openapi-check
make mocks-drift-check
make stringer-drift-check
make sqlc-check
MIGRATION_DSN='postgres://user:pass@localhost:5432/db?sslmode=disable' make migration-validate
make docker-migration-validate
make migration-validate
```

When `make migration-validate` skips because no `MIGRATION_DSN` or Docker daemon is available, report a proof gap instead of saying the migration was validated. When `make openapi-check` changes generated files, report drift and do not claim the contract is clean until the generated artifacts are reconciled and the check reruns cleanly.

## Exa Source Links
- [Go command documentation](https://pkg.go.dev/cmd/go): `go generate` is explicit and not automatically run by `go build` or `go test`; `go test` package patterns bound test scope.
- [testing package documentation](https://pkg.go.dev/testing): focused runtime contract tests can be selected with `-run`.

## Repo-Local Sources
- `docs/build-test-and-development-commands.md`: OpenAPI, sqlc, migration, and generated artifact workflows.
- `Makefile`: `openapi-check`, `sqlc-check`, and `migration-validate` recipes.
