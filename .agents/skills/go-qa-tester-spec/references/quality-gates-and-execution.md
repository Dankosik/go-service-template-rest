# Quality Gates And Execution

## When To Load
Load this when the test strategy must name repository-executable validation commands, CI mapping, focused local checks, coverage/race/fuzz/integration gates, OpenAPI checks, migration checks, or residual proof limits.

## Source Grounding
- Local source of truth: `docs/build-test-and-development-commands.md`, `Makefile`, `.github/workflows/ci.yml`, `.github/workflows/nightly.yml`, `test/README.md`, and `internal/api/README.md`.
- Use Go `cmd/go` docs for test flags, coverage mode, caching, focused runs, race, and fuzz semantics.
- Do not claim readiness from commands that were not run or from commands that do not cover the changed surface.

## Selected/Rejected Level Examples
| Change risk | Selected execution gate | Rejected gate | Why |
| --- | --- | --- | --- |
| Pure local Go behavior | Focused `go test` for the package plus `make test`/`make vet` as broader evidence | Only a stale previous CI run | Fresh evidence must cover the changed package and repo baseline. |
| API contract or generated binding change | `make openapi-check` plus relevant focused contract test | `go test ./...` only | Repo contract target also covers generation, drift, runtime contract, lint, and validation. |
| Concurrency, shared state, worker, or cancellation change | `make test-race` or focused `go test -race` plus targeted scenario | Non-race unit pass only | Race detector evidence is required, but it must execute the risky path. |
| Fuzzable parser/decoder/validator | Seeded unit cases plus `make test-fuzz-smoke` when fuzz targets exist | Unbounded fuzz in normal CI plan | Smoke is CI-compatible; longer fuzz belongs in nightly or explicit hardening. |
| Integration behavior or Docker-backed runtime dependency | `make test-integration` locally, `REQUIRE_DOCKER=1 make test-integration` in CI | Unit-only | The proof depends on composed runtime or external dependency behavior. |
| Coverage reporting or release evidence | `make test-report COVERAGE_MIN=<value>` | Interpreting `go test` pass as coverage proof | Coverage requires coverage artifacts and threshold evaluation. |
| SQL/migration changes | `make sqlc-check` and migration validation target when migrations changed | App unit tests only | Drift and migration compatibility are separate proof obligations. |

## Scenario Matrix Examples
| Risk surface | Focused command pattern | Broader repo gate | CI mapping or artifact | Pass/fail observable |
| --- | --- | --- | --- | --- |
| Package-local behavior | `go test ./internal/<pkg> -run <RelevantPattern> -count=1` | `make test` | CI `test` job | targeted suite passes fresh; repo tests pass |
| Concurrency path | `go test -race ./internal/<pkg> -run <RelevantPattern> -count=1` | `make test-race` | CI `test-race` job | race run passes and scenario exercises risky path |
| API contract | focused handler/runtime contract command if known | `make openapi-check` | CI `openapi-contract` job | generated drift clean, runtime contract check pass, OpenAPI validate/lint pass |
| Integration | focused integration package if known | `make test-integration` | CI `test-integration` with `REQUIRE_DOCKER=1` | Docker-backed suite passes or local skip is explicitly not CI evidence |
| Coverage | none unless a focused coverage question exists | `make test-report COVERAGE_MIN=<value>` | CI `test-coverage` plus `coverage.out`, JUnit, JSON artifacts | threshold pass and artifacts produced |
| Fuzz smoke | `go test ./<pkg> -fuzz=<Target> -fuzztime=<bounded>` | `make test-fuzz-smoke FUZZ_TIME=<bounded>` | nightly fuzz smoke when configured | target runs bounded; no fuzz target skip is recorded honestly |

## Pass/Fail Observables
- Each command maps to the changed risk surface and is repository-supported.
- Local skips, especially Docker-dependent integration skips, are called out and not presented as CI-equivalent proof.
- Focused `go test` commands use `-count=1` when fresh non-cached evidence matters.
- Coverage claims require coverage command/artifact evidence, not a plain test pass.
- Race claims require `-race` execution on a scenario that touches the shared-state path.
- Fuzz claims distinguish seed corpus execution under regular `go test` from active fuzzing under `-fuzz`.
- API claims use `make openapi-check` when OpenAPI/runtime contract behavior changed.

## Exa Source Links
- [go command test packages](https://pkg.go.dev/cmd/go#hdr-Test_packages)
- [go command testing flags](https://pkg.go.dev/cmd/go#hdr-Testing_flags)
- [testing package](https://pkg.go.dev/testing)
- [Go Fuzzing](https://go.dev/doc/fuzz/)
- [Data Race Detector](https://go.dev/doc/articles/race_detector.html)
- [Go security best practices](https://go.dev/doc/security/best-practices)
