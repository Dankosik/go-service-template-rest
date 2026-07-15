# Quality Gates And Execution

## Behavior Change Thesis
When loaded for symptom "the strategy must name executable validation", this file makes the model map proof obligations to repository-supported commands and proof limits instead of likely mistake "claim generic `go test` or stale CI as sufficient evidence."

## When To Load
Load this when test strategy must name local or CI validation commands, focused checks, coverage/race/fuzz/integration gates, OpenAPI checks, migration checks, generated-code drift checks, or residual proof limits.

## Decision Rubric
- Use `docs/build-test-and-development-commands.md`, `Makefile`, `.github/workflows/ci.yml`, `.github/workflows/nightly.yml`, `test/README.md`, and `internal/api/README.md` as command sources.
- Start with the proof obligation, then choose the narrowest focused command and the broader repo/CI gate that cover the same surface.
- Use focused `go test ./internal/<pkg> -run <RelevantPattern> -count=1` when fresh package-local evidence matters.
- Use `make test-race` or focused `go test -race ...` only when the scenario executes the shared-state or goroutine path.
- Use `make openapi-check` when OpenAPI, generated API, or runtime contract behavior changed.
- Use `make test-integration` or `REQUIRE_DOCKER=1 make test-integration` when Docker-backed/runtime dependency behavior is the proof target; local skips are not CI-equivalent evidence.
- Use `make sqlc-check` for SQL query/source drift and `make migration-validate` or `docker-migration-validate` when migrations changed.
- Use `make test-report COVERAGE_MIN=<value>` only for coverage threshold/artifact claims, not as a synonym for "tests passed."
- Use `make test-fuzz-smoke FUZZ_TIME=<bounded>` only when fuzz targets exist or the skip is recorded honestly.

## Imitate
| Risk Surface | Focused Command Pattern | Broader Gate | Proof Limit To State |
| --- | --- | --- | --- |
| Package-local behavior | `go test ./internal/<pkg> -run <RelevantPattern> -count=1` | `make test` | Focused pass proves changed package only; repo pass gives baseline. |
| Concurrency path | `go test -race ./internal/<pkg> -run <RelevantPattern> -count=1` | `make test-race` | Race run proves only executed interleavings, not unexercised paths. |
| API contract | focused handler/runtime contract command if known | `make openapi-check` | Contract gate covers generation, drift, runtime contract, lint, and validation. |
| Integration behavior | focused integration package if known | `make test-integration` or `REQUIRE_DOCKER=1 make test-integration` | Docker-dependent local skip is not success evidence. |
| SQL/migration | focused affected package if useful | `make sqlc-check`; `make migration-validate` when migrations changed | SQL drift and migration rehearsal are separate proof obligations. |
| Fuzz smoke | `go test ./<pkg> -run '^$' -fuzz=<Target> -fuzztime=<bounded>` | `make test-fuzz-smoke FUZZ_TIME=<bounded>` | No fuzz targets means a recorded skip, not robustness proof. |

## Reject
- "Run `go test ./...`" as the only gate for API drift, migrations, SQL generation, race-sensitive behavior, integration dependencies, or coverage claims.
- "CI passed before" as proof. Readiness needs fresh evidence for the changed surface or an explicit reason it cannot be rerun.
- "Run integration tests" when Docker is unavailable and the local command skipped, without stating that CI still must provide the proof.
- "Coverage is fine" without a coverage command, threshold, and artifact expectation.

## Agent Traps
- Do not invent Makefile targets; inspect repo command docs or Makefile before naming them.
- Do not use docker and native command names interchangeably when the environment expectation changes.
- Do not claim fuzz, race, coverage, OpenAPI, or migration proof from a command that does not produce that evidence.
- Do not defer the command mapping to implementation for high-risk surfaces; the test strategy must be executable enough to plan.

## Validation Shape
Every validation recommendation should state: proof obligation -> focused command if useful -> broader repo/CI gate -> artifact or pass/fail observable -> residual proof limit.
