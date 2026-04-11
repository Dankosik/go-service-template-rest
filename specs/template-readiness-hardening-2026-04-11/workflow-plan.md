# Template Readiness Hardening Workflow Plan

## Master Control

- Task: implement the repository-template readiness hardening gaps found in `specs/template-readiness-review-2026-04-11`.
- Execution shape: lightweight local artifact synthesis with an upfront phase-collapse waiver, followed by phased implementation.
- Phase-collapse waiver: the previous planning session already ran full orchestrated review lanes across architecture, quality, HTTP/API, data, reliability, QA, and security. That session did not do new research or code; the user explicitly asked for a complete implementation handoff in files before a later implementation session. Therefore specification, technical design, and implementation planning were collapsed into one pre-code artifact-writing pass.
- Current phase: implementation-phase-4.
- Phase status: complete.
- Session boundary reached: yes.
- Ready for next session: no implementation work remains; review/commit may start if desired.
- Next session starts with: done / review handoff if desired.

## Scope

- In scope for the completed session: Phase 4 persistence sample treatment, deterministic migrations, docs/test-placement guidance, Make help discoverability, validation-tooling fixes needed for phase proof, and final validation tasks T019 through T030.
- Out of scope for the completed session: commits and unrelated refactors.
- Allowed writes in the completed session: Phase 4 migrations, Postgres sample comments, docs, Makefile/help/tooling, and existing control/progress artifacts.

## Artifact Status

- `workflow-plan.md`: approved for implementation handoff.
- `workflow-plans/planning.md`: approved.
- `workflow-plans/implementation-phase-1.md`: complete.
- `workflow-plans/implementation-phase-2.md`: complete.
- `workflow-plans/implementation-phase-3.md`: complete.
- `workflow-plans/implementation-phase-4.md`: complete.
- `spec.md`: approved.
- `design/overview.md`: approved.
- `design/component-map.md`: approved.
- `design/sequence.md`: approved.
- `design/ownership-map.md`: approved.
- `design/data-model.md`: approved.
- `design/contracts/http-security-and-generated-routes.md`: approved.
- `plan.md`: approved.
- `tasks.md`: approved.
- `test-plan.md`: approved.
- `rollout.md`: not expected; this is a reusable template hardening change, not a live rollout.

## Implementation Readiness

- Status: PASS.
- Rationale: the review findings have been reconciled into concrete decisions, ordered implementation slices, and proof expectations. No code should start before reading `spec.md`, `design/overview.md`, `plan.md`, `tasks.md`, and `test-plan.md`.
- Accepted residual risk: `make sqlc-check` was blocked during review by local `go tool sqlc` compilation failure in `pg_query_go` on macOS. Implementation should rerun the SQLC proof, preferably through Docker tooling if native sqlc remains blocked.

## Validation Evidence From Prior Review

- `make check`: passed.
- `make openapi-check`: passed.
- `make sqlc-check`: attempted; blocked before drift checking by local `go tool sqlc` compilation failure in `pg_query_go` against the macOS SDK (`strchrnul` duplicate declaration).
- Integration tests and migration rehearsal: not run during review.

## Phase 1 Validation Evidence

- `go test ./internal/config -count=1`: passed.
- `make check`: passed.

## Phase 2 Validation Evidence

- `go test ./cmd/service/internal/bootstrap ./internal/app/health ./internal/infra/http -count=1`: passed.
- `go test ./internal/config -count=1`: passed.
- `make check`: passed.

## Phase 3 Validation Evidence

- `go test ./internal/infra/http -count=1`: passed.
- `make check`: passed.
- `make openapi-runtime-contract-check`: passed.
- `make openapi-lint`: passed.
- `make openapi-validate`: passed.
- `make openapi-check`: passed with a temporary git index containing the regenerated OpenAPI source/generated pair. The first plain run failed at `openapi-drift-check` because `internal/api/openapi.gen.go` is intentionally changed in the uncommitted working tree; the real git index was left unchanged.

## Phase 4 Validation Evidence

- SQLC behavior check: a temporary empty `queries/*.sql` / migration setup failed sqlc generation with `no queries contained`, so `ping_history` was retained as an explicit template SQLC sample rather than removed.
- `go test ./internal/infra/postgres -count=1`: passed.
- `make help`: passed and shows OpenAPI, SQLC, integration-test, and Docker validation targets.
- `make docker-sqlc-check`: passed.
- `make docker-migration-validate`: passed after fixing the Docker tooling cleanup trap.
- `make test-integration`: passed.
- `make check`: passed after excluding `.cache/` from local and Docker goimports scans.
- `make openapi-check`: passed with a temporary git index containing the current OpenAPI source/generated pair; the real git index was left unchanged.

## Resume Order

1. Read this file.
2. Read `workflow-plans/planning.md`.
3. Read `spec.md`.
4. Read `design/overview.md`, then the linked design artifacts.
5. Read `plan.md`, `tasks.md`, and `test-plan.md`.
6. Task is implemented and validated; next action is review/commit if desired.
