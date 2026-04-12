# Agent Contract Refresh Workflow Plan

Created: 2026-04-12

## Task Frame

The user reviewed the repository's subagent portfolio and wants the review decomposed into small follow-up sessions so agent/workflow instruction changes can be made step by step.

The review identified one blocking contract drift and several follow-up improvements:

- `challenger-agent` runtime instructions in `.codex/agents/challenger-agent.toml` and `.claude/agents/challenger-agent.md` only describe `pre-spec-challenge`, while `AGENTS.md`, `docs/spec-first-workflow.md`, README, and skills require the challenger role for `workflow-plan-adequacy-challenge`, `pre-spec-challenge`, and `spec-clarification-challenge`.
- `.codex/agents/observability-agent.toml` exists, but `.claude/agents/observability-agent.md` was not present during the intake check, even though README links project-scoped agents through `.claude/agents`.
- Agent files repeat global repo policy; later passes should reduce duplication only after the per-agent contract is made explicit.
- Agent return formats are not strict enough for predictable orchestrator fan-in.
- Agent files mostly lack short `Inspect first` / source-of-truth blocks.
- `.codex/agents` and `.claude/agents` are manually mirrored enough to create drift risk.
- Review-skill coverage is asymmetric for observability, delivery, and distributed roles.
- Some Codex agents lack `nickname_candidates`; model/reasoning tuning is a later optional optimization.

## Scope

In scope:

- Refresh project-scoped subagent instructions under `.codex/agents/` and `.claude/agents/`.
- Keep subagents read-only and advisory.
- Preserve one-skill-per-pass routing.
- Align the challenger role with the repository workflow gates.
- Standardize return shapes and evidence anchors.
- Add concise `Inspect first` source-of-truth blocks.
- Decide whether `.codex/agents` and `.claude/agents` remain manual mirrors or need a separate drift-check/canonical-source task.
- Track missing review skills and optional Codex ergonomics as later backlog unless explicitly pulled into the implementation plan.

Out of scope for this implementation cycle:

- Changing Go service runtime behavior.
- Creating new review skills such as `go-observability-review`, `go-delivery-review`, or `go-distributed-review`.
- Introducing model/reasoning overrides before support and desired policy are checked.
- Introducing canonical agent-instruction generation or CI drift checks.
- Rewriting the repository-wide workflow in `AGENTS.md` or `docs/spec-first-workflow.md` unless a concrete drift is found and the task is reopened upstream.

## Execution Shape

Execution shape: `lightweight local`.

Rationale: the work is non-trivial and multi-session, but the target remains bounded to repository instruction/configuration artifacts. No read-only subagent fan-out was requested for this planning handoff, and the first implementation phase is narrow, reversible, and independently verifiable.

Escalate to `full orchestrated` if the scope expands into new skill design, generated mirror tooling, CI policy changes, model/reasoning policy, separate review fan-out, or conflicting decisions across agent/runtime surfaces.

## Current Phase

Current phase: `implementation-phase-7`.

Status: `completed`.

Session boundary reached: `yes`.

Ready for next session: `yes`.

Next session starts with: `validation-phase-1`.

Next-session goal: run T900-T903 from `tasks.md`, performing final proof and closeout.

Implementation phase 7 completed with focused proof. Validation phase 1 may start next because the safe deduplication and drift-policy checkpoint is complete and no new blockers were found.

## Research Mode

Research mode for the completed planning phase: `local`.

Reason: the approved spec and design bundle define the instruction-only slices; planning needed bounded inventory and README checks rather than read-only fan-out.

Fan-out is not planned for the implementation phases. If implementation upgrades to `full orchestrated`, record lanes in the active phase workflow plan before calling subagents.

## Workflow Plan Adequacy Challenge

Status: `waived by lightweight-local exception for this planning handoff`.

Rationale: the task remains instruction-only, the approved spec and design bundle define the work, no subagent fan-out was requested for this planning session, and the first implementation phase is narrow, reversible, and independently verifiable.

Reopen rule: if the task upgrades to full orchestrated execution, separate review fan-out, generated mirror tooling, CI drift checks, new skills, or model policy, reopen planning and run `challenger-agent` with exactly one skill: `workflow-plan-adequacy-challenge`.

## Specification Clarification Gate

Status: `waived by lightweight-local exception`.

Rationale: the approved scope is instruction/documentation contract cleanup, the highest-risk drift is directly evidenced by repository files, and no user-requested subagent fan-out is active in this session. The waiver applies only while scope stays instruction-only. If later work pulls in canonical-source generation, CI drift-checks, new review skills, model/reasoning overrides, or unresolved Codex-vs-Claude runtime policy, reopen specification and run a read-only `challenger-agent` lane with exactly one skill: `spec-clarification-challenge`.

## Implementation Readiness

Status: `PASS`.

Gate result: implementation completed through `implementation-phase-7`; `validation-phase-1` may begin next.

Proof path: `tasks.md` lists task-level proof, `plan.md` lists phase checkpoints and final validation commands, and `workflow-plans/validation-phase-1.md` owns the final closeout route.

Accepted risks: none that block implementation.

## Artifact Status

- `workflow-plan.md`: `approved for validation handoff`
- `workflow-plans/workflow-planning.md`: `approved for handoff`
- `workflow-plans/specification.md`: `approved for handoff`
- `workflow-plans/technical-design.md`: `approved for planning handoff`
- `workflow-plans/planning.md`: `approved for implementation handoff`
- `workflow-plans/implementation-phase-1.md`: `completed`
- `workflow-plans/implementation-phase-2.md`: `completed`
- `workflow-plans/implementation-phase-3.md`: `completed`
- `workflow-plans/implementation-phase-4.md`: `completed`
- `workflow-plans/implementation-phase-5.md`: `completed`
- `workflow-plans/implementation-phase-6.md`: `completed`
- `workflow-plans/implementation-phase-7.md`: `completed`
- `workflow-plans/validation-phase-1.md`: `pending`
- dedicated review phase-control file: `not expected`
- `spec.md`: `approved for implementation handoff`
- `design/overview.md`: `approved`
- `design/component-map.md`: `approved`
- `design/sequence.md`: `approved`
- `design/ownership-map.md`: `approved`
- `research/*.md`: `not expected`
- `design/data-model.md`: `not expected`
- `design/dependency-graph.md`: `not expected`
- `design/contracts/`: `not expected`
- `plan.md`: `approved for implementation handoff`
- `tasks.md`: `approved for validation handoff`
- `test-plan.md`: `not expected`; validation fits in `plan.md`
- `rollout.md`: `not expected`

## Session Decomposition

1. Workflow planning: completed.
2. Specification: completed; `spec.md` approves the scope, non-goals, assumptions, validation expectations, and high-level slices.
3. Technical design: completed; minimal design bundle approved for planning handoff.
4. Planning: completed; `plan.md`, `tasks.md`, implementation phase-control files, validation phase-control file, and implementation-readiness `PASS` are recorded.
5. Implementation phase 1: challenger three-mode contract fix, T001-T004.
6. Implementation phase 2: observability Claude mirror and README inventory repair, T010-T013.
7. Implementation phase 3: return contracts for review-focused agents, T020-T023.
8. Implementation phase 4: return contracts for advisory and mixed-mode agents, T030-T033.
9. Implementation phase 5: `Inspect first` blocks for runtime and domain roles, T040-T043.
10. Implementation phase 6: `Inspect first` blocks for workflow and meta roles, T050-T053.
11. Implementation phase 7: safe deduplication and drift-policy checkpoint, T060-T064.
12. Validation phase 1: final proof and closeout, T900-T903.

## Handoff Notes

The challenger contract drift, observability mirror drift, review-focused return-contract slice, advisory/mixed-mode return-contract slice, runtime/domain inspect-first slice, workflow/meta inspect-first slice, and safe deduplication/drift-policy checkpoint are complete. Do not start by broad-rewriting all agents.

Validation phase 1 should run only the final proof and closeout listed in `tasks.md` T900-T903 and existing task-local control/progress artifacts.

Do not broaden into CI tooling, new skills, canonical-source generation, workflow-document rewrites, or model policy without reopening specification or technical design.
