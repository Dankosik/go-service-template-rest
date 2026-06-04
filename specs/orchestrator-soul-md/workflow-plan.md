# Workflow Plan

Task: SOUL.md orchestrator personality layer for production-ready Go service template work
Execution shape: full orchestrated instruction-surface workflow
Execution rationale: the change affects repository-wide agent identity, AGENTS.md integration, and future service-generation behavior. It is not runtime code, API, data, or deployment work, but it changes the standing instruction surface that future agents will use for production-readiness tradeoffs.

Current phase: planning
Phase status: complete
Session boundary reached: yes
Ready for next session: yes
Next session starts with: implementation from `tasks.md` at T001
Next session context bundle:
- `workflow-plan.md`: current phase state, artifact status, gates, and next-session routing.
- `workflow-plans/planning.md`: planning readiness, task-ledger review/readiness gate, and implementation handoff state.
- `tasks.md`: approved executable implementation ledger and Goal-ready handoff.
- `workflow-plans/specification.md`: specification readiness, clarification-gate handling, and phase-local stop state.
- `spec.md`: approved canonical decisions for SOUL.md purpose, non-goals, content shape, AGENTS integration, validation, and design-depth routing.
- `workflow-plans/research.md`: prior research mode, fan-in summary, and source limits.
- `research/soul-md-agent-personality-practices.md`: evidence for SOUL.md ownership, structure, and AGENTS.md integration.
- `AGENTS.md`: repository-wide operating contract that must remain the authority for workflow, gates, and project rules.
- `docs/spec-first-workflow.md`: detailed artifact and phase-boundary mechanics.
- `Makefile` and `scripts/ci/required-guardrails-check.sh`: validation surfaces the task ledger must account for.
- `docs/build-test-and-development-commands.md`: docs surface named by the ledger for the expected `scripts/ci/` guardrail change.

Default implementation resume order:
1. `tasks.md`
2. `spec.md`
3. `workflow-plans/planning.md`
4. `research/soul-md-agent-personality-practices.md`
5. `AGENTS.md`
6. `docs/spec-first-workflow.md`
7. `Makefile`
8. `scripts/ci/required-guardrails-check.sh`
9. `docs/build-test-and-development-commands.md`

## Artifact Status

| Artifact | Status | Rationale |
| --- | --- | --- |
| `workflow-plan.md` | complete | Records planning completion, task-ledger readiness, and next-session routing. |
| `workflow-plans/research.md` | complete | Records local-only research mode, tracks, fan-in, limits, and stop rule. |
| `research/soul-md-agent-personality-practices.md` | complete | Preserves source-backed evidence for later specification. |
| `workflow-plans/specification.md` | complete | Records specification readiness, local clarification handling, phase result, and next-phase routing. |
| `workflow-plans/planning.md` | complete | Records planning readiness, task-ledger review/readiness, local-only review rationale, and stop rule. |
| `spec.md` | approved | Canonical decision record for accepted SOUL.md purpose, non-goals, content shape, AGENTS integration, validation obligations, and design-depth decision. |
| `design/` | not expected | Separate design depth is not triggered; the change is a bounded instruction/docs surface with compact ownership and validation decisions in `spec.md`. |
| `tasks.md` | approved | Executable Goal-ready ledger created and reviewed with readiness PASS. |
| `test-plan.md` | not expected | Validation obligations fit in `tasks.md`; no separate layered test strategy is needed. |
| `rollout.md` | not expected | No runtime deployment, migration, rollback, failback, or staged rollout is expected for this change. |

## Gates

- Research subagent gate: local-only.
- Local research rationale: the available `spawn_agent` tool permits subagents only when the user explicitly asks for subagents, delegation, or parallel agent work. The user asked for research and examples, not subagents. The research surfaces were also inspectable directly: Hermes official docs and source docs, an OpenClaw persona repository, local setup-repo AGENTS.md examples, and this repository's workflow contract.
- Formal spec clarification challenge: complete via scoped-down local orchestrator challenge. A spawned challenger lane was not used because the available `spawn_agent` tool requires explicit user permission for subagents, and the user requested a specification-session rather than delegation or parallel agent work.
- Clarification scoped-down rationale: the only approval-critical seam is SOUL/AGENTS authority and validation. Runtime/API/data/security/reliability/rollout lenses cannot change approval for the accepted non-runtime instruction-surface scope.
- Clarification gate result: PASS.
- Separate technical design: not triggered.
- Technical design review: not required.
- Planning and task-ledger review/readiness: complete.
- Task-ledger review fan-out: local-only. Rationale: the available `spawn_agent` tool requires explicit user permission for subagents, and the readiness risk is concentrated in one tightly coupled instruction/docs/guardrail seam that local review can cover without weaker read-only enforcement.
- Task-ledger review result: PASS.
- Implementation readiness: PASS.
- Workflow-plan adequacy challenge: local self-check PASS. No spawned challenger was used for the same tool-contract reason, and no independent workflow-control question remains after `workflow-plans/planning.md` records phase status, approved ledger status, artifact expectations, proof obligations, and next-session routing.

## Specification Result

Specification is complete and approved. The accepted decisions are:

- Root `SOUL.md` is the only SOUL artifact in scope.
- `SOUL.md` owns stable orchestrator identity, engineering judgment posture, ambiguity behavior, communication style, avoid-list behavior, and boundaries.
- `AGENTS.md` remains the authority for workflow, gates, commands, paths, subagent protocol, task-local artifacts, and validation.
- AGENTS integration must include both explicit lower-precedence wording and an `@SOUL.md` include/reference.
- Guardrails must require `SOUL.md` and check the AGENTS/SOUL precedence boundary.
- Separate technical design is not needed; planning may start from the approved `spec.md`.

## Blockers

None.

## Planning Result

Planning is complete. `specs/orchestrator-soul-md/tasks.md` is approved with implementation readiness PASS.

Executable ledger coverage:

- T001 creates root `SOUL.md`.
- T002 adds the AGENTS lower-precedence bridge and `@SOUL.md` include/reference.
- T003 updates guardrails to require `SOUL.md` and preserve the AGENTS/SOUL boundary.
- T004 updates the docs surface required by `scripts/ci/` changes.
- T005 runs required validation: `rtk make guardrails-check`, `rtk make agents-check`, `rtk git diff --check`, and docs-drift handling when valid refs exist.
- T006 records ledger-owned closeout evidence in `tasks.md` and `spec.md`.

## Accepted Assumptions

- The target is a durable orchestrator personality for this template, not a one-off session persona.
- `SOUL.md` must improve technical judgment and communication defaults without weakening existing workflow gates.
- The file should be concise enough to remain useful when injected into model context.
- Root-only SOUL integration is enough for the accepted scope; host-specific mirrors require a reopened workflow plan and spec if later required.

## Reopen Targets

- Reopen specification if implementation discovers a required host-specific SOUL distribution behavior, an AGENTS include limitation that invalidates the root-only decision, or a conflict that would weaken AGENTS.md authority.
- Reopen workflow planning if the accepted scope expands from one repository-level SOUL.md plus AGENTS.md integration into a multi-host distribution/mirroring system.
