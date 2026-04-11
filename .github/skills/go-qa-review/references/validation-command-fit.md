# Validation Command Fit

## Behavior Change Thesis
When loaded for validation evidence that may not prove the changed risk, this file makes the model choose the smallest command that would fail for the regression instead of accepting broad `go test ./...` or demanding full CI by reflex.

## When To Load
Load this when proposed or reported validation commands may miss the changed package, build tag, generated contract, integration harness, race-only behavior, fuzz target, or CI-parity requirement.

## Decision Rubric
- A command is fit when it would fail for the regression described in the finding.
- Prefer targeted package or named-test commands for narrow behavior.
- Add `-count=1` when cached success could hide whether the changed package was exercised.
- Do not accept `go test ./...` as integration, OpenAPI drift, race, or fuzz evidence when those require separate tags/tools.
- Do not require full CI when a smaller command proves the only changed risk.
- Ask for CI-parity commands only when the change crosses generated code, Docker-backed infrastructure, tooling, or multiple packages.

## Imitate

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

Copy this shape: explain why the reported command misses the risk and name the repo command that covers it.

```text
[medium] [go-qa-review] internal/infra/postgres/ping_history_repository.go:88
Issue:
The validation note lists `go test ./internal/infra/postgres`, but the changed repository path is exercised only by integration-tag tests against Postgres.
Impact:
The package command can pass without running the DB-backed rollback scenario that would catch the regression.
Suggested fix:
Run the integration target or the narrowed integration-tag package command for this repository test.
Reference:
Validation command: `make test-integration` or `go test -tags=integration ./test/... -run 'PingHistoryRepository' -count=1`.
```

Copy this shape: distinguish package compile/unit proof from tagged integration proof.

## Reject

```text
[medium] [go-qa-review] api/openapi/service.yaml:1
Issue:
Run all tests.
Impact:
The PR is not fully validated.
Suggested fix:
Run `make check-full`.
Reference:
N/A
```

Reject this because it turns validation into ceremony and does not say what regression the command would catch.

## Agent Traps
- `go test ./...` can be broad and still miss integration tags, generated artifacts, OpenAPI drift, fuzz targets, and race-only failures.
- A targeted command without `-count=1` may be fine for local iteration but weak as freshness evidence after a change.
- Coverage output is not validation unless coverage policy or a named untested behavior is the issue.
- Fuzz smoke is relevant only when fuzz targets or fuzz-suitable parser/input hardening are touched.

## Validation Shape
Map command to risk: package `go test ... -run ... -count=1` for narrow behavior; `go test -race` or `make test-race` for shared-memory proof; `make test-integration` for integration-tag/Docker-backed behavior; `make openapi-check` for OpenAPI generation and drift; fuzz commands only for fuzz targets; CI-local commands only for cross-cutting parity.
