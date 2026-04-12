# Bootstrap Maintainability Fix Plan Workflow

## Task

Implement and validate the accepted maintainability fixes from the read-only review of `cmd/service/internal/bootstrap`.

## Execution Shape

- Shape: `lightweight local`.
- Current phase: `done`.
- Phase collapse waiver: approved for this planning session because the change set is bounded, local to bootstrap/config/telemetry helper seams, and the user explicitly wants the full implementation context written now while implementation happens in a separate session.
- Workflow plan adequacy challenge: waived by the lightweight-local decision; no subagents were requested for this follow-up planning turn.
- Spec clarification challenge: waived by the same lightweight-local decision. The review findings are already concrete and no planning-critical product/business question remains.

## Inputs

- Prior read-only review of `cmd/service/internal/bootstrap`.
- Prior subagent lanes: `go-idiomatic-review`, `go-language-simplifier-review`, and `go-design-review`.
- Repository baseline: `docs/repo-architecture.md`.
- Relevant local source context in `cmd/service/internal/bootstrap`, `internal/infra/telemetry`, and `internal/config`.

## Artifact Status

- `spec.md`: approved under lightweight-local waiver.
- `design/overview.md`: approved.
- `design/component-map.md`: approved.
- `design/sequence.md`: approved.
- `design/ownership-map.md`: approved.
- `plan.md`: approved.
- `tasks.md`: complete.
- `test-plan.md`: not expected; proof obligations fit in `plan.md` and `tasks.md`.
- `rollout.md`: not expected; no runtime deployment choreography or migration is required.
- Implementation phase file: `workflow-plans/implementation-phase-1.md` complete.
- Validation phase file: `workflow-plans/validation-phase-1.md` complete for task-local scope; full package/repo validation is blocked by unrelated parallel workspace changes.

## Implementation Readiness

- Status: `CONCERNS`, accepted and resolved by fresh validation.
- Accepted risk: the metric contract change intentionally introduces `startup_rejections_total{reason}` and stops using `config_validation_failures_total` for non-config startup failures. This is correct semantically but can affect local tests or any dashboard expecting non-config reasons on the old metric.
- Proof obligation: metrics tests and bootstrap tests now prove the new metric contract and absence of non-config reasons on the config metric.
- Implementation completed and task-local validation passed.

## Next Session

- Starts with: no follow-up required for this task-local scope; rerun full package/repo validation after unrelated tracing/Postgres workspace changes are reconciled.
- Must consume: not applicable.
- Must not create new workflow/process/design/planning artifacts unless a later review finds a real reopen condition.

## Stop Rule

This task-local scope is complete after the recorded validation evidence in `spec.md` and `workflow-plans/validation-phase-1.md`.
