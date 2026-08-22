# Validation Commands

## Load When

Load when a proof obligation or readiness claim needs a command that actually
exercises its changed surface.

## Decide

Command fitness means the regression would fail, not that the command is broad.
Read the repository command owner and prefer the narrowest fresh focused test;
add a gate only for a wider surface.

| Surface | Matching gate |
| --- | --- |
| OpenAPI or generated bindings | `make openapi-check` |
| Docker/multi-package behavior | `REQUIRE_DOCKER=1 make test-integration` |
| Shared-memory race | focused `go test -race` or `make test-race` |
| Order/scheduler sensitivity | focused `go test -shuffle=on -count=<n>` |
| `t.Parallel()` policy | `make test-parallelism-check` |
| SQL/migration drift | `make sqlc-check`, `make migration-validate` |

Use `-count=1` when cache could hide whether current code/environment ran. A
zero-exit `-run` can match no tests; verify execution when one named test carries
the claim. Bare `go test ./...` cannot prove integration tags, contract drift,
fuzz, race, or coverage. A skipped Docker suite is not integration evidence.

## Prove

Map obligation -> narrowest command that would fail -> wider gate only when
needed -> residual limit. Report the actual execution/result through Evidence
Result V1 rather than promoting a command beyond its scope.
