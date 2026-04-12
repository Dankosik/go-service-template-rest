# Bootstrap Review Fixes Workflow Plan

Task: Prepare implementation-ready context for the four accepted review findings from the `cmd/service/internal/bootstrap` review.

Current phase: `implementation-phase-1`
Phase status: `complete`
Execution shape: `lightweight local`
Phase-collapse waiver: yes. Rationale: this is a bounded follow-up to review findings; the earlier planning pass produced the preserved pre-implementation context bundle, and this implementation phase consumed that bundle without reopening design.
Session boundary reached: `yes`
Ready for next session: `yes`
Next session starts with: validation or closeout if required by the orchestrator.

Scope:
- Implement and preserve the correct behavior for:
  - Mongo degraded startup status metric drift.
  - Network policy configuration source-of-truth clarification.
  - `networkPolicyErrorLabels` production/test ownership.
  - Redundant context-error branch in `dependencyInitFailure`.
- Update existing workflow/progress artifacts for implementation status.

Non-goals:
- No implementation.
- No migration to new runtime config keys in this planning pass.
- No broad bootstrap rewrite.
- No Redis/Mongo adapter implementation beyond the planned bootstrap admission cleanup.

Artifact status:
- `spec.md`: approved for this lightweight-local planning handoff.
- `design/`: approved.
- `plan.md`: approved.
- `tasks.md`: T001-T008 complete.
- `test-plan.md`: not expected; proof obligations are small enough to live in `plan.md` and `tasks.md`.
- `rollout.md`: not expected; changes are local bootstrap/docs/test behavior with no deploy sequencing.
- `workflow-plans/planning.md`: complete.
- `workflow-plans/implementation-phase-1.md`: complete.

Clarification gates:
- Spec clarification challenge: waived under the lightweight-local phase-collapse waiver. The accepted findings are already review-backed, and the chosen decisions are repository-local maintenance decisions with no unresolved product or business policy question.
- Workflow plan adequacy challenge: waived under the lightweight-local phase-collapse waiver. No new subagent fan-out is planned for this pre-implementation pass.

Primary decisions:
- Treat `NETWORK_*` as a deliberate operator network-policy channel outside normal `APP__...` runtime config, but document it explicitly and keep it fail-closed.
- Keep the network policy classification helper in production only if it is consumed by production logging; otherwise move it out of production code during implementation.
- Prefer one same-package helper for degraded dependency startup logging plus status metric updates.

Blockers:
- None known.

Stop rule:
- Implementation boundary reached. Do not begin a separate validation or review phase in this session unless explicitly requested.
