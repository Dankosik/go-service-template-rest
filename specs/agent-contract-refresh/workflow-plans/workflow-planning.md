# Workflow Planning Phase

Created: 2026-04-12

## Phase Scope

This phase creates the task-local handoff for the agent contract refresh. It does not edit agent files, write `spec.md`, create design artifacts, or start implementation.

## Inputs Considered

- User-provided review of the subagent portfolio and workflow contract.
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.codex/config.toml`
- `.codex/agents/challenger-agent.toml`
- `.claude/agents/challenger-agent.md`
- `.codex/agents/api-agent.toml`
- `.codex/agents/observability-agent.toml`
- README agent/workflow references via targeted search.
- Agent inventory comparison between `.codex/agents` and `.claude/agents`.

## Local Findings For Handoff

- `challenger-agent` currently describes only `pre-spec-challenge` in both Codex and Claude runtime files.
- Repository workflow docs require challenger use for `workflow-plan-adequacy-challenge` and `spec-clarification-challenge` as well.
- README already describes the challenger role as broader than the runtime files.
- `.codex/agents/observability-agent.toml` exists; `.claude/agents/observability-agent.md` was absent in the inventory check.
- Codex `nickname_candidates` were present for `api-agent`, `data-agent`, `observability-agent`, and `quality-agent`, but not for the other Codex agents.
- Current review-skill directories did not include dedicated observability, delivery, or distributed review skills.

## Execution Shape And Research Mode

Execution shape: `lightweight local`.

Research mode: `local`.

No subagent lanes are planned for this phase.

Escalation trigger: if the next session expands into new skill creation, CI tooling, generated mirror tooling, or unresolved policy conflicts, update the master workflow plan and consider `full orchestrated` before fan-out.

## Phase Completion Marker

Complete when:

- `workflow-plan.md` exists and records execution shape, artifact expectations, next phase, and the key handoff facts.
- `workflow-plans/workflow-planning.md` exists and records only phase-local orchestration and handoff context.
- The next session start point is explicit.
- No agent files, specs, design files, plan files, task ledgers, or implementation files are edited in this phase.

Status: `completed`.

## Workflow Plan Adequacy Challenge

Status: `waived for this lightweight-local closeout`.

Rationale: the current turn is a handoff artifact creation pass, and no subagent fan-out was requested or needed to record the next safe phase. If later work is upgraded to full orchestrated execution, run a read-only `challenger-agent` lane with exactly one skill, `workflow-plan-adequacy-challenge`, before treating the repaired workflow-control pair as ready.

## Next Action

Next session should start with `specification` and create `specs/agent-contract-refresh/spec.md`.

Suggested first decision in the spec: approve the `challenger-agent` fix as the first implementation slice and keep broader portfolio cleanup as ordered follow-up slices.

## Stop Rule

Stop at this workflow-planning boundary. Do not begin `spec.md`, technical design, implementation planning, or agent file edits in this session.
