# Agent Contract Refresh Planning Phase Plan

Created: 2026-04-12

## Phase Scope

This phase turns the approved `spec.md` and approved `design/` bundle into implementation-ready planning artifacts.

Allowed writes in this phase:

- `plan.md`
- `tasks.md`
- `workflow-plans/planning.md`
- `workflow-plans/implementation-phase-1.md`
- `workflow-plans/implementation-phase-2.md`
- `workflow-plans/implementation-phase-3.md`
- `workflow-plans/implementation-phase-4.md`
- `workflow-plans/implementation-phase-5.md`
- `workflow-plans/implementation-phase-6.md`
- `workflow-plans/implementation-phase-7.md`
- `workflow-plans/validation-phase-1.md`
- `workflow-plan.md`

Prohibited in this phase:

- agent runtime edits under `.codex/agents/` or `.claude/agents/`
- README edits
- skill edits
- Go runtime, generated code, migration, test, or config edits
- specification or design rewrites

## Inputs Considered

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/repo-architecture.md`
- `.agents/skills/planning-session/SKILL.md`
- `.agents/skills/planning-and-task-breakdown/SKILL.md`
- planning references for implementation readiness and phase-control files
- `specs/agent-contract-refresh/workflow-plan.md`
- `specs/agent-contract-refresh/spec.md`
- `specs/agent-contract-refresh/design/overview.md`
- `specs/agent-contract-refresh/design/component-map.md`
- `specs/agent-contract-refresh/design/sequence.md`
- `specs/agent-contract-refresh/design/ownership-map.md`
- runtime agent inventory under `.codex/agents` and `.claude/agents`
- `.codex/config.toml`
- README agent inventory excerpt

## Planning Outputs

- `plan.md`: approved for implementation handoff.
- `tasks.md`: approved for implementation handoff.
- `test-plan.md`: not expected.
- `rollout.md`: not expected.
- `workflow-plans/implementation-phase-1.md`: created, pending.
- `workflow-plans/implementation-phase-2.md`: created, pending.
- `workflow-plans/implementation-phase-3.md`: created, pending.
- `workflow-plans/implementation-phase-4.md`: created, pending.
- `workflow-plans/implementation-phase-5.md`: created, pending.
- `workflow-plans/implementation-phase-6.md`: created, pending.
- `workflow-plans/implementation-phase-7.md`: created, pending.
- `workflow-plans/validation-phase-1.md`: created, pending.
- Dedicated review phase-control file: not expected.

## Implementation Readiness

Status: PASS.

Gate result: implementation may start with `implementation-phase-1` in a later session.

Proof path: `tasks.md` lists task-level proof, `plan.md` lists phase checkpoints and final validation commands, and `workflow-plans/validation-phase-1.md` owns the final closeout route.

Accepted risks: none that block implementation.

Reopen triggers:

- `observability-agent` is intentionally Codex-only.
- Codex and Claude runtime formats cannot preserve equivalent semantics with hand-maintained mirrors.
- The user wants canonical-source generation, CI drift checks, new review skills, model/reasoning policy, or workflow-document rewrites in this same task cycle.
- Implementation finds that `AGENTS.md`, `docs/spec-first-workflow.md`, or skill bodies must change to preserve the approved agent contract.

## Workflow Plan Adequacy Challenge

Status: waived by lightweight-local exception for this planning handoff.

Rationale: the task remains instruction-only, the approved spec and design bundle define the work, no subagent fan-out was requested for this planning session, and the first implementation phase is narrow, reversible, and independently verifiable.

Reopen rule: if the task upgrades to full orchestrated execution, separate review fan-out, generated mirror tooling, CI drift checks, new skills, or model policy, reopen planning and run `challenger-agent` with exactly one skill: `workflow-plan-adequacy-challenge`.

## Completion Marker

Completed when:

- `plan.md` exists and records phased execution, checkpoints, validation, readiness, and reopen conditions.
- `tasks.md` exists and lists dependency-ordered executable tasks with proof obligations.
- Required implementation and validation phase-control files exist before implementation starts.
- `workflow-plan.md` points the next session to `implementation-phase-1`.
- No implementation or README/agent runtime edits happened in this planning session.

Completion status: satisfied.

## Stop Rule

Session boundary reached: yes.

Do not begin implementation, review, or validation in this session.

## Next Action

Next session starts with: `implementation-phase-1`.

Implementation-phase-1 should consume `plan.md`, `tasks.md`, and `workflow-plans/implementation-phase-1.md`, then implement T001-T004 only.

## Blockers

No active blockers.
