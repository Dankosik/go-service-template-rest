# Generated API And Migration Verification

## Behavior Change Thesis
When loaded for generated-code, API-contract, sqlc, or migration symptoms, this file makes the model add drift or rehearsal proof instead of treating ordinary tests or a successful compile as evidence that generated artifacts and schema transitions are current.

## When To Load
Load this when changes touch OpenAPI specs, generated API code, generated mocks, enum stringer output, sqlc output, SQL queries, migration files, migration-backed schema behavior, or runtime API contract tests.

## Decision Rubric
- Generated artifacts require drift proof because stale generated files can still compile and tests can still pass.
- OpenAPI contract readiness usually needs `make openapi-check`, not just handler tests.
- Add `BASE_OPENAPI=/path/to/base.yaml make openapi-breaking` only when compatibility against a base spec is part of the claim.
- SQL query or migration changes need `make sqlc-check` when generated sqlc output may change.
- Migration changes need an actual migration rehearsal, or an explicit proof gap when neither `MIGRATION_DSN` nor Docker is available.
- Migration-backed behavior usually needs both rehearsal proof and affected data-access or integration tests.
- If a drift check modifies files or reports drift, the clean claim is not verified until artifacts are reconciled and the check reruns cleanly.

## Imitate
| Claim | Choose | Copy this behavior |
|---|---|---|
| "OpenAPI generated code is current" | `make openapi-check` | Use the repo composite for generation, drift, runtime contract, lint, and validation. |
| "Runtime API contract is green" | `make openapi-runtime-contract-check` or `go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1` | Use the narrow runtime contract target when only runtime contract wiring is claimed. |
| "API change is ready" | focused API/handler tests plus `make openapi-check`; add `make openapi-breaking` only when compatibility is in scope | Combine behavior proof with generated-contract proof. |
| "sqlc output is current" | `make sqlc-check` | Check generated query drift and stale generated query stems. |
| "Generated mocks are current" | `make mocks-drift-check` | Match the generator surface actually touched. |
| "Stringer output is current" | `make stringer-drift-check` | Keep enum string artifacts tied to the generator, not to unrelated tests. |
| "Migration rehearsal passed" | `MIGRATION_DSN=... make migration-validate`, `make docker-migration-validate`, or `make migration-validate` only when it actually falls through to Docker | Verify that the command rehearsed up/down/up instead of skipping. |

## Reject
| Plausible bad conclusion | Why it fails |
|---|---|
| "OpenAPI is current" after `go test ./internal/api` | Unit tests do not prove generated artifact drift, spec lint, spec validation, or runtime contract wiring. |
| "sqlc is current" after `go test ./internal/infra/postgres/...` | Data-access tests can pass with stale or extra generated files. |
| "Migration validated" when `make migration-validate` says `MIGRATION_DSN` is empty and Docker is unavailable | The target exits after a skip message; the rehearsal did not run. |
| "Migration-backed behavior works" after only migration rehearsal | Rehearsal proves schema transition mechanics, not repository behavior over the new schema. |

## Agent Traps
- `go generate` is explicit in this repo's make targets. Do not assume `go test` refreshed generated files.
- `make openapi-check` can fail by producing drift. Treat generated-file modifications as evidence of drift, not success.
- `make migration-validate` can exit successfully after printing a skip message. Inspect the output, not just the exit status.
- Docker-backed migration proof is real only when the Docker fallback actually ran.
- If command names feel stale, inspect `Makefile` and `docs/build-test-and-development-commands.md`.

## Validation Shape
Name the generated or migration surface, the drift or rehearsal command, whether drift/skip occurred, and the behavior tests, if any, that prove runtime behavior over that surface.
