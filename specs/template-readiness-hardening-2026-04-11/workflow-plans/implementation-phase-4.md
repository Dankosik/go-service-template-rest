# Implementation Phase 4 Workflow Plan

## Phase Control

- Phase: implementation-phase-4.
- Phase status: complete.
- Entry condition: Phase 3 complete or explicitly skipped by the user.
- Scope: persistence sample treatment, docs, Make help, and test placement.
- Tasks: T019 through T026, then validation tasks T027 through T030 as applicable.
- Stop rule: do not claim done until final validation evidence is fresh and recorded in the existing workflow/task surfaces.

## Expected Work

- Resolved `ping_history` as an explicit template SQLC sample after proving the current sqlc setup fails with no query files.
- Made retained migrations deterministic by removing unnecessary `IF NOT EXISTS` / `IF EXISTS`.
- Updated placement, persistence, endpoint, security, outbound integration, and test docs.
- Added feature validation targets to `make help`.
- Fixed Docker tooling PATH, migration cleanup, and `.cache/` formatting exclusions needed to run the required proof commands after Docker SQLC validation.

## Proof

- `go test ./internal/infra/postgres -count=1`: passed.
- `make help`: passed.
- `make docker-sqlc-check`: passed.
- `make docker-migration-validate`: passed.
- `make test-integration`: passed.
- `make check`: passed.
- `make openapi-check`: passed with a temporary git index for the current OpenAPI source/generated pair; real git staging was unchanged.

## Completion

- Completion marker: T019 through T030 are checked in `tasks.md`.
- Session boundary reached: yes.
- Ready for next session: no implementation work remains; review/commit may start if desired.

## Reopen Conditions

- Reopen planning if SQLC cannot be validated without keeping a default sample query/schema and that changes the data-model decision.
