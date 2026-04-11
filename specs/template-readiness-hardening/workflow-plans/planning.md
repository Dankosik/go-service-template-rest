# Template Readiness Hardening Planning Phase

## Phase Control

- Phase: planning.
- Status: complete.
- Scope: turn all planned template-readiness research findings into implementation-ready decisions, design context, plan, and tasks.
- Stop rule: stop after writing planning artifacts; do not edit code/docs targeted by the fixes in this session.

## Phase-Collapse Waiver

The session intentionally collapses specification, technical design, and implementation planning into one lightweight local pre-code pass.

Rationale:
- The change is a bounded hardening follow-up from completed subagent-backed review.
- No product decision, API behavior change for live endpoints, data migration, or runtime rollout is introduced.
- The user asked for the full pre-implementation context now and deferred implementation to a later session.
- The resulting artifacts still preserve the `spec.md -> design/ -> plan.md -> tasks.md` chain for the implementation session.

## Decisions Produced

- `spec.md` records the stable decisions and non-goals.
- `design/` records component ownership, sequence, and source-of-truth boundaries.
- `plan.md` records the execution strategy and readiness.
- `tasks.md` records the executable task ledger.
- `workflow-plans/implementation-phase-1.md` records the next-session implementation boundary.

## Implementation Readiness Gate

- Status: PASS.
- Reason: each finding has a selected correction, exact change surfaces, proof expectations, and no unresolved design blockers.
- Required proof in implementation session:
  - `make openapi-runtime-contract-check`
  - `go test ./internal/infra/http -count=1`
  - `go test ./internal/infra/postgres -count=1`
  - `go test ./internal/config ./cmd/service/internal/bootstrap -count=1`
  - `make openapi-check` when the local OpenAPI toolchain is available; otherwise record why it was not run.

## Completion Marker

Complete when this phase's artifacts exist and the implementation phase can start from `tasks.md` without re-planning.
