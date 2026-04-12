# Specification Phase

Created: 2026-04-12

## Phase Scope

This phase creates the task-local `spec.md` decision record for the agent contract refresh. It does not edit agent runtime files, create `design/`, write `plan.md` or `tasks.md`, or start implementation.

## Inputs Considered

- User-provided review of the subagent portfolio and workflow contract.
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/repo-architecture.md`
- `.agents/skills/specification-session/SKILL.md`
- `.agents/skills/spec-document-designer/SKILL.md`
- `.agents/skills/spec-clarification-challenge/SKILL.md`
- `specs/agent-contract-refresh/workflow-plan.md`
- `specs/agent-contract-refresh/workflow-plans/workflow-planning.md`
- `.codex/config.toml`
- `.codex/agents/challenger-agent.toml`
- `.claude/agents/challenger-agent.md`
- `.codex/agents/observability-agent.toml`
- README agent inventory and workflow references
- Agent inventory comparison between `.codex/agents` and `.claude/agents`

## Readiness Check

Status: `satisfied`.

Rationale: the workflow-planning handoff already framed the task, the user's review supplied candidate priorities, and local repository intake confirmed the main runtime-contract drift and mirror inventory drift. Remaining uncertainties are narrow enough to record as assumptions or reopen conditions in `spec.md`.

## Clarification Challenge

Status: `waived by lightweight-local exception`.

Lane: not run in this session.

Rationale: the current pass is bounded to instruction and documentation contracts, no user-requested subagent fan-out is active, and the highest-risk issue is already directly evidenced by repository files: `challenger-agent` runtime files only describe `pre-spec-challenge`, while workflow docs and skills require two additional challenge gates. The waiver applies only while scope stays instruction-only and does not include new skills, CI checks, generated mirror tooling, model policy, or conflicting runtime support decisions.

Reopen trigger: run a read-only `challenger-agent` lane with exactly one skill, `spec-clarification-challenge`, if later work pulls canonical-source generation, CI drift-checks, new review skills, model/reasoning overrides, or unresolved Codex-vs-Claude runtime policy into the approved scope.

## Specification Result

`spec.md` status: `approved for technical-design handoff`.

Approval rationale: the spec records the blocking challenger-contract drift, the Codex/Claude observability inventory drift, the minimum return-format and inspect-first standards, explicit non-goals, validation consequences, and reopen conditions for broader tooling or policy work.

## Phase Completion Marker

Complete when:

- `spec.md` exists and records scope, constraints, decisions, assumptions, validation expectations, and handoff notes.
- `workflow-plans/specification.md` records phase-local status, clarification challenge status, stop rule, and next action.
- `workflow-plan.md` records `spec.md` status and routes the next session.
- No `design/`, `plan.md`, `tasks.md`, agent runtime files, or implementation files are edited in this phase.

Status: `completed`.

## Next Action

Next session should start with `technical-design`.

The technical-design session should either:

- create a minimal task-local design bundle for the agent instruction refresh, or
- record an explicit design-skip rationale if the approved scope remains instruction-only and planning can safely derive tasks directly from `spec.md`.

Planning and implementation must not start until that handoff is resolved.

## Stop Rule

Stop at this specification boundary. Do not begin `technical design`, `plan.md`, `tasks.md`, or agent file edits in this session.

## Parallelizable Later Work

Later planning may split implementation by the approved high-level slices:

- `challenger-agent` three-mode contract fix.
- `observability-agent` Claude mirror and README inventory repair.
- Shared return-format contract.
- Role-specific `Inspect first` blocks.
- Safe deduplication of repeated global policy.
- Drift-policy checkpoint or separate follow-up spec.
- Optional ergonomics/backlog checkpoint.
