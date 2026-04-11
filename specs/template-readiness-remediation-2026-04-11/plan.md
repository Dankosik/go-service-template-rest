# Implementation Plan

## Strategy

Implement three reviewable phases that improve the template's contributor guidance and guardrails without changing service runtime behavior.

Order the work from intent to enforcement:

1. Clarify docs and extension recipes.
2. Add or widen tests/commands that enforce the clarified rules.
3. Validate targeted packages and commands.

## Phase 1: Docs And Contributor Guidance

Own T001-T015:

- Feature path docs.
- Redis/Mongo stub clarity.
- Online migration safety guidance.
- Domain-type, telemetry, test-placement, list-limit, transaction, and DB-required-feature guidance.
- OpenAPI compatibility wording.
- README feature-work pointer.
- Strict-server endpoint checklist.
- Project-tree refresh.

## Phase 2: Config And HTTP Guardrails

Own T016-T022:

- Config snapshot proof.
- `enforceSecretSourcePolicy` cleanup.
- Config test hygiene.
- API runtime contract gate.
- Manual root-route policy.
- HTTP lifecycle test hygiene.
- `httpx` package-name convention closeout.

## Phase 3: Data, Bootstrap, Artifacts, And Validation

Own T023-T028:

- Postgres sample assertion strength.
- Test-only Postgres helper cleanup.
- Startup dependency label drift.
- `.artifacts/test/*` cleanup.
- Coverage-matrix closeout.
- Final validation.

## Checkpoints

- Checkpoint A: Phase 1 docs clarify feature path, domain type placement, telemetry placement, test placement, config additions, Redis/Mongo stubs, OpenAPI compatibility wording, online migration safety, list limits, transactions, DB-required feature wiring, and the README entry point.
- Checkpoint B: Phase 2 config snapshot proof exists and HTTP runtime contract/manual-route/lifecycle guardrails are tightened.
- Checkpoint C: Phase 3 data/bootstrap/artifact cleanup is done or explicitly closed by a documented no-op decision.
- Checkpoint D: targeted validation passes or skipped commands are explicitly justified.

## Risk Notes

- Do not accidentally claim `make openapi-check` proves breaking compatibility unless `openapi-breaking` is actually included.
- Do not turn Redis/Mongo into implemented adapters by documentation wording alone.
- Do not weaken explicit config parse errors by introducing production reflection mapping.
- Do not create new process docs when a short recipe in existing docs is enough.
- Do not edit unrelated dirty `specs/template-readiness-*` paths.
- Do not overfit Postgres sample fixes into production ping behavior.
- Do not rename `httpx` casually.

## Validation Plan

Required targeted validation after implementation:

- `go test ./internal/config -count=1`
- `go test ./cmd/service/internal/bootstrap -count=1` if startup label code changes
- `go test ./internal/infra/http -count=1`
- `go test ./internal/infra/postgres -count=1`
- `make openapi-runtime-contract-check`
- `make openapi-check`

Recommended broader validation:

- `make test`
- `make check`

Conditional validation if Postgres sample tests are changed:

- `go test ./internal/infra/postgres -run 'TestPingHistoryRepository' -count=1`
- `go test -tags=integration ./test/... -run 'TestPingHistoryRepositorySQLCReadWrite' -count=1` when Docker is available.

## Implementation Readiness

Status: PASS for a later implementation session constrained to `tasks.md`.
