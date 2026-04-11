# Validation Command Fit Examples

## When To Load
Load this when review needs to assess whether the proposed or reported validation commands actually exercise the changed risk surface at the right package, tag, contract, integration, race, fuzz, or CI-parity level.

## Review Lens
Validation commands are evidence, not ceremony. A command is fit when it would fail for the regression under discussion. Broad commands can be useful, but they may still miss integration tags, OpenAPI drift, race-only failures, fuzz targets, or Docker-backed infrastructure. Conversely, demanding full CI for every small local change dilutes review signal.

## Bad Finding Example
```text
[medium] [go-qa-review] api/openapi/service.yaml:1
Issue:
Run all tests.
Impact:
The PR is not fully validated.
Suggested fix:
Run make check-full.
Reference:
N/A
```

Why it fails: it does not explain why the current command misses the changed OpenAPI risk or which narrower command would prove it.

## Good Finding Example
```text
[high] [go-qa-review] api/openapi/service.yaml:1
Issue:
The validation evidence only lists `go test ./...`, but this change updates the OpenAPI contract. The repo's contract proof also requires generation/drift checks, generated API compile checks, runtime contract tests, lint, and validation through `make openapi-check`.
Impact:
The PR can merge a spec/runtime drift or invalid OpenAPI document even though regular Go tests pass.
Suggested fix:
Run `make openapi-check`, or at minimum the failing substep plus `go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1` when narrowing an investigation.
Reference:
Validation command: `make openapi-check`.
```

## Non-Findings To Avoid
- Do not require `make check-full` when a targeted unit package command proves the only changed behavior.
- Do not accept `go test ./...` as integration evidence for tests behind the `integration` build tag.
- Do not accept a cached or unrelated package run as proof for the changed package; use `-count=1` when freshness matters.
- Do not ask for coverage output unless the risk is coverage policy or a concrete untested behavior.
- Do not ask for fuzz smoke unless fuzz targets or fuzz-suitable parsing/input behavior are touched.

## Smallest Safe Correction
Choose the command that maps to the risk:
- package-level `go test ... -run ... -count=1` for narrow behavior;
- `make test-race` or `go test -race` for shared-memory or concurrency-sensitive proof;
- `make test-integration` for Docker-backed Postgres or integration-tag behavior;
- `make openapi-check` for OpenAPI generation, drift, runtime contract, lint, and validation;
- `make test-fuzz-smoke` or a targeted `go test -fuzz` for fuzz targets and parser/input hardening;
- `make ci-local` or `make docker-ci` when cross-cutting CI parity is the actual proof goal.

## Validation Command Examples
```bash
go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1
go test -race ./internal/infra/http -run '^TestServerRunAndShutdown$' -count=50
make test-integration
make openapi-check
make test-fuzz-smoke FUZZ_TIME=60s
make ci-local
```

## Source Links From Exa
- [cmd/go test flags docs](https://pkg.go.dev/cmd/go/internal/test)
- [Data Race Detector](https://go.dev/doc/articles/race_detector.html)
- [Go Fuzzing](https://go.dev/doc/fuzz/)

## Repo-Local Convention Links
- `docs/build-test-and-development-commands.md`
- `Makefile`
- `test/README.md`
