# Planning Phase Plan: internal/infra Review Fixes

Phase: planning
Status: complete

## Scope

Convert the completed `internal/infra` review findings into implementation-ready context without editing production code.

## Phase-Collapse Waiver

Waiver: lightweight local phase-collapse.

Rationale:
- The user explicitly asked for pre-implementation context in files, not implementation.
- Findings are bounded to handwritten infra packages and tests.
- The preceding review already used read-only specialist lanes for Go idiom, simplification, design, chi, and DB/cache.
- This planning pass creates the full handoff bundle and stops before code.

Spec clarification challenge: waived under this same rationale; no new subagent fan-out was requested in this follow-up turn.
Workflow adequacy challenge for the new planning bundle: waived under this same rationale.

## Artifact Output

- `spec.md`: approved.
- `design/overview.md`: approved.
- `design/component-map.md`: approved.
- `design/sequence.md`: approved.
- `design/ownership-map.md`: approved.
- `plan.md`: approved.
- `tasks.md`: approved.
- `workflow-plans/implementation-phase-1.md`: created.
- `workflow-plans/validation-phase-1.md`: created.

## Implementation Readiness

Status: PASS

Accepted risks:
- `SetupTracing` fallback defaults remain out of scope.
- Optional Docker-backed integration proof may be unavailable in the next session.

Proof obligations:
- Focused infra tests and vet must pass.
- HTTP trace tests must verify route attributes, not only span names.
- Postgres limiter tests must cover limiter accounting, not just `shouldKeepReleasedConn`.

## Stop Rule

Do not implement fixes in this planning session. The next session starts with `workflow-plans/implementation-phase-1.md` and `tasks.md`.

## Completion

Completion marker: satisfied.
Session boundary reached: yes.
Ready for next session: yes.
