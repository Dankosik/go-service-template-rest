# Template Readiness Hardening Plan

## Execution Context

This plan consumes `spec.md`, `research/coverage-audit.md`, and the approved `design/` bundle. Implementation is a single lightweight hardening phase with targeted verification. No code is changed in the planning session.

## Phase Plan

### Phase 1: Template Hardening Fixes

- Objective: fix the must-fix review findings and the supporting research recommendations marked planned in `research/coverage-audit.md`.
- Depends on: approved `spec.md` and `design/`.
- Task ledger: `tasks.md` T001-T018.
- Change surface:
  - `internal/infra/http/openapi_contract_test.go`
  - `internal/api/README.md`
  - `internal/infra/postgres/ping_history_repository.go`
  - `internal/infra/postgres/ping_history_repository_test.go`
  - `README.md`
  - `CONTRIBUTING.md`
  - `docs/repo-architecture.md`
  - `docs/project-structure-and-module-organization.md`
  - `docs/build-test-and-development-commands.md`
  - `test/README.md`
  - `internal/config/types.go` or nearby config-owned file
  - `internal/config/validate.go`
  - `internal/config/config_test.go`
  - `cmd/service/internal/bootstrap/startup_probe_helpers.go`
  - `cmd/service/internal/bootstrap/startup_dependencies.go`
  - bootstrap tests if needed
- Acceptance criteria:
  - `make openapi-runtime-contract-check` selects the security-decision guard.
  - The API README describes app-owned behavior, scoped auth placement, Problem responses, and public-route non-regression tests for protected endpoints.
  - The ping history sample rejects invalid and over-limit list sizes before SQL.
  - Redis store-mode readiness policy is expressed through config-owned API and consumed by bootstrap.
  - README, CONTRIBUTING, command docs, project-structure docs, architecture docs, and test README expose the placement and validation conventions identified in the coverage audit without duplicating full guides.
- Planned verification:
  - `make openapi-runtime-contract-check`
  - `go test ./internal/infra/http -count=1`
  - `go test ./internal/infra/postgres -count=1`
  - `go test ./internal/config ./cmd/service/internal/bootstrap -count=1`
  - `make openapi-check` when local OpenAPI tooling is available
  - `make check` when time/environment allows
  - Documentation review against `research/coverage-audit.md`
- Exit criteria:
  - All targeted tests pass or blocked commands are recorded with reason.
  - `tasks.md` is updated to reflect completed work and proof.
  - Deferred research items remain out of implementation scope unless explicitly reopened.

## Cross-Phase Validation Plan

Minimum implementation-session proof is the targeted command set above plus documentation review against `research/coverage-audit.md`. Full `make check` is recommended but not required to claim the planned hardening scope is addressed if the environment lacks optional OpenAPI/Node tooling; any skipped command must be explicitly reported.

## Implementation Readiness

- Status: PASS.
- Reason: selected fixes are explicit and bounded; no product/security/data model decision remains open.
- Proof obligations: listed in Phase 1 planned verification.

## Blockers / Assumptions

- Assumption: no protected endpoint exists today, so endpoint auth implementation remains guidance-only.
- Assumption: `ping_history` max limit can be sample-owned and does not need API contract work.
- Assumption: config-owned Redis helper methods are acceptable within repository-internal API.
- Assumption: docs updates should stay compact and link to canonical guides rather than duplicating whole sections.

## Handoffs / Reopen Conditions

Reopen technical design if:
- implementation needs to add real auth middleware,
- Redis mode changes beyond `cache` and `store`,
- `ping_history` is discovered to be used as production behavior rather than template sample,
- OpenAPI component additions trigger generated-code or contract changes beyond the current scope.
