# Planning Phase Plan

## Phase Scope

- Phase: planning.
- Status: complete.
- Mode: lightweight local with phase-collapse waiver.
- Deliverable: `spec.md`, core `design/` bundle, `plan.md`, `tasks.md`, and next-session phase-control files.
- Out of scope: implementation, generated-code edits, auth design, sample migration/query rename, Redis/Mongo adapter design.

## Waiver

This planning session collapses specification, technical design, and implementation planning into one local pre-code pass because:

- the previous review phase already ran architecture, maintainability, API, data, QA, and challenger lanes;
- the user explicitly requested full preimplementation context now and implementation in a separate later session;
- no code changes are made in this session;
- the selected implementation is bounded and non-breaking.

Spec clarification and workflow adequacy challenges are waived for this planning pass under the same rationale. The waiver does not authorize implementation in this session.

## Outputs

- `spec.md`: approved for this bounded implementation scope.
- `design/overview.md`: approved.
- `design/component-map.md`: approved.
- `design/sequence.md`: approved.
- `design/ownership-map.md`: approved.
- `plan.md`: approved.
- `tasks.md`: approved.
- `workflow-plans/implementation-phase-1.md`: ready for next session.
- `workflow-plans/validation-phase-1.md`: ready for validation after implementation.

## Implementation Readiness

Status: `CONCERNS`.

Accepted risks:

- no runnable fake business domain is added;
- `ping_history` remains the SQLC fixture unless a maintainer explicitly chooses a rename;
- protected endpoint guidance is documented without implementing auth policy.

Proof obligations:

- `make guardrails-check`
- `go test ./cmd/service/internal/bootstrap ./internal/infra/http`
- `go test ./...`
- `make openapi-check` only if OpenAPI sources/generated artifacts change
- `make sqlc-check` only if migration/query/generated SQLC surfaces change

## Stop Rule

Stop after writing and validating the planning artifacts. Next session starts at `workflow-plans/implementation-phase-1.md`.

## Completion Marker

Complete: all planned artifacts were created, implementation readiness is recorded as `CONCERNS`, and implementation is deferred to the next session.

