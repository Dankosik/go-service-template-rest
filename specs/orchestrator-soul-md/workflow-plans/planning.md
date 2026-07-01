# Planning Phase Plan

Phase: planning
Phase status: complete
Session boundary reached: yes
Ready for next session: yes
Next action: implementation
Next session starts with: `T001` in `specs/orchestrator-soul-md/tasks.md`

## Input Sources Used

- `workflow-plan.md`: active phase state, accepted gates, artifact expectations, and next-session routing.
- `workflow-plans/specification.md`: approved specification readiness, clarification-gate handling, and stop state.
- `spec.md`: canonical SOUL.md decisions, non-goals, compact design answers, validation obligations, and no-separate-design decision.
- `workflow-plans/research.md`: prior research routing and source limits.
- `research/soul-md-agent-personality-practices.md`: evidence for SOUL.md purpose, file boundary, content shape, and AGENTS integration.
- `AGENTS.md`: repository-wide workflow authority and planning/readiness gate rules.
- `docs/spec-first-workflow.md`: detailed task-ledger, implementation-readiness, and next-session handoff mechanics.
- `Makefile`: validation target names and docs-drift target shape.
- `scripts/ci/required-guardrails-check.sh`: current required-file and regex guardrail patterns to extend.
- `scripts/ci/docs-drift-check.sh`: proof that `scripts/ci/` changes require a paired docs update.
- `docs/build-test-and-development-commands.md`: primary docs surface for `guardrails-check`, `agents-check`, docs-drift, and CI mapping.

## Planning Readiness

Result: PASS.

Planning can start from the approved `spec.md` because:

- Specification is approved and records root-only SOUL scope, AGENTS/SOUL precedence, required content shape, guardrail obligations, validation obligations, and reopen triggers.
- Separate technical design is explicitly not triggered; the compact design decisions in `spec.md` are sufficient for executable tasking.
- Technical design review is not required.
- No listed artifact contradicted the approved SOUL purpose, root-only scope, lower-precedence boundary, or no-separate-design decision.
- `tasks.md` was missing and is now the required planning output.

## Task Ledger

Created: `specs/orchestrator-soul-md/tasks.md`

Ledger summary:

| Task | Purpose |
| --- | --- |
| T001 | Create root `SOUL.md` with approved compact personality sections and precedence boundary. |
| T002 | Add the `AGENTS.md` lower-precedence bridge and `@SOUL.md` include/reference. |
| T003 | Extend required guardrails for `SOUL.md` presence and AGENTS/SOUL precedence drift. |
| T004 | Update the docs surface required by `scripts/ci/` guardrail changes. |
| T005 | Run required validation, including guardrails, agent mirror check, diff whitespace check, and docs-drift handling. |
| T006 | Record ledger-owned closeout evidence in `tasks.md` and `spec.md`. |

Companion artifacts:

- `test-plan.md`: not expected. Validation obligations are direct repository commands and targeted reads that fit in `tasks.md`.
- `rollout.md`: not expected. No runtime deployment, migration, rollback, failback, compatibility window, or staged rollout is in scope.
- `workflow-plans/review-phase-N.md`: not expected. No separate post-code review phase is planned.
- `workflow-plans/validation-phase-N.md`: not expected. Validation is ledger-owned in T005/T006.

## Task-Ledger Review

Ledger-review fan-out rationale:

- Spawned task-ledger review lanes were not used because the available `spawn_agent` tool is permitted only when the user explicitly asks for subagents, delegation, or parallel agent work.
- The implementation readiness risk is concentrated in one tightly coupled instruction/docs/guardrail seam: `SOUL.md` must remain lower-precedence personality guidance while `AGENTS.md` and task-local artifacts remain authoritative.
- API, data, security, reliability, performance, observability, concurrency, deployment, and rollout lenses cannot change implementation readiness for the approved non-runtime instruction-surface scope.
- A local orchestrator review can cover the needed lenses without weaker read-only enforcement: coverage/traceability, dependency ordering, proof/QA, and workflow-control adequacy.

Review result: PASS.

Review checks:

| Lens | Result | Evidence |
| --- | --- | --- |
| Coverage / traceability | PASS | D1-D8 from `spec.md` map to T001-T004, validation obligations map to T005, and closeout maps to T006. D9/D10 are represented by no-design and planning-to-implementation routing. |
| Dependency ordering | PASS | `SOUL.md` is created before the AGENTS bridge; guardrails follow both files; docs follow the `scripts/ci/` change; validation and closeout are last. |
| Proof / QA | PASS | Required commands are explicit: `rtk make guardrails-check`, `rtk make agents-check`, `rtk git diff --check`, and docs-drift when valid refs exist. Targeted reads cover content boundaries that a command alone cannot prove. |
| Workflow-control adequacy | PASS | `tasks.md` has a Goal Contract, Implementation Handoff, evidence fields, blocked-stop rule, reopen targets, and no implementation-time open questions. |
| Companion artifacts | PASS | `test-plan.md`, `rollout.md`, and separate review/validation phase files are not triggered by the approved scope. |

Implementation readiness: PASS.

Accepted concerns: none.

Proof obligations carried into implementation:

- Keep `SOUL.md` identity/personality-only and explicitly lower-precedence.
- Preserve AGENTS authority and `@RTK.md` while adding `@SOUL.md`.
- Encode SOUL presence and boundary checks in `scripts/ci/required-guardrails-check.sh`.
- Pair the `scripts/ci/` change with docs and handle docs-drift proof.
- Use RTK for shell validation commands.

Reopen target: none for implementation start. Reopen `specification` if implementation requires changing SOUL purpose, root-only scope, precedence boundary, or no-separate-design decision. Reopen `workflow planning` if the scope expands to host-specific SOUL distribution or mirroring.

## Stop Rule

Planning is complete. Do not create `SOUL.md`, edit `AGENTS.md`, modify guardrails/docs, or run implementation in this session.

## Recommended Next Action

Start implementation from `specs/orchestrator-soul-md/tasks.md` at T001 in a new session. The next session may orchestrate the approved ledger through its named proof without stopping between task IDs unless blocked by a reopen condition.
