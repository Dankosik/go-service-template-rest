# Template Readiness Improvements Plan

## Execution Context

This plan consumes the completed review, `spec.md`, and `design/`. It prepared the Phase 1 implementation and validation path now recorded as complete.

Execution shape: one bounded implementation phase plus a validation phase.

Implementation readiness: `CONCERNS`.

Accepted concerns:

- The implementation must remain domain-neutral and must not add a fake business feature.
- The implementation must not rename `ping_history` schema/query/generated surfaces without explicit maintainer approval.
- The implementation must not design or implement auth policy.

## Phase Plan

### Phase 1: Guidance And Guardrails

Objective: make the correct first-production-feature path obvious and enforce the most important boundaries with narrow guardrails.

Depends on: `spec.md` and `design/` in this task folder.

Task ledger: `tasks.md` T001-T012.

Status: complete.

Change surface:

- `README.md`
- `docs/project-structure-and-module-organization.md`
- `docs/configuration-source-policy.md`
- `internal/api/README.md`
- `test/README.md`
- `scripts/ci/required-guardrails-check.sh`
- `internal/infra/http/router.go`
- `internal/infra/http/router_test.go`
- `cmd/service/internal/bootstrap/startup_common.go`
- bootstrap tests under `cmd/service/internal/bootstrap`

Acceptance criteria:

- The docs describe where a first production business feature goes across app, HTTP, Postgres, bootstrap, and tests.
- `ping_history` is clearly described as a replaceable SQLC fixture, not production business state.
- Redis/Mongo guard stubs are clearly not cache/store runtime contracts.
- Protected-operation guidance names the need for real security design without adding placeholder auth.
- Route registration guardrails protect the generated OpenAPI path from manual `/api/...` routes.
- HTTP `Allow` responses use a client-friendly canonical header value with matching tests.
- App/domain import guardrails prevent infra/generated SQLC/driver imports.
- Dependency-probe startup rejection logs include the error value consistently with sibling rejection helpers.

Planned verification:

- `make guardrails-check`
- `go test ./cmd/service/internal/bootstrap ./internal/infra/http`
- `go test ./...`
- `make openapi-check` only if OpenAPI source or generated API artifacts change.
- `make sqlc-check` only if migration, SQL query, or SQLC generated surfaces change.

Review/checkpoint:

- Phase 1 stopped for validation; validation is complete. If auth, schema rename, or real Redis/Mongo adapter work appears necessary, reopen specification instead of continuing.

Exit criteria:

- All tasks in `tasks.md` are complete or explicitly deferred with rationale.
- Validation phase confirms the selected proof set.

## Cross-Phase Validation Plan

Use the smallest proof set matched to touched surfaces:

- Docs-only changes: targeted review plus `go test ./...` if code/test changes are also present.
- Guardrail script change: `make guardrails-check`.
- HTTP test guard change: `go test ./internal/infra/http`.
- Bootstrap log/test change: `go test ./cmd/service/internal/bootstrap`.
- Generated-code source change: run the matching OpenAPI or SQLC check, but this plan does not expect generated-code source changes.

## Implementation Readiness

Status: `CONCERNS`.

Implementation proceeded under these conditions:

- keep the implementation inside the listed change surface;
- do not add a fake production domain;
- do not rename `ping_history` schema/query/generated artifacts;
- do not implement auth;
- stop and reopen specification if any of those boundaries become necessary.

## Blockers / Assumptions

No blockers.

Assumption: documentation-plus-guardrails is the correct first fix because the repository is intended as a template, not a fixed business product.

Deferred low-priority review points:

- `startup_probe_helpers.go` file split/rename.
- Telemetry init failure reason vocabulary centralization.
- SQLC fixture rename or migration churn.

## Handoffs / Reopen Conditions

Phase 1 implementation and validation are complete. No next implementation session is required unless a maintainer chooses a deferred follow-up below.

Reopen specification if:

- the maintainer chooses to rename the SQLC fixture;
- the maintainer wants a runnable exemplar business domain;
- real auth behavior is requested;
- Redis/Mongo runtime adapters are requested.
