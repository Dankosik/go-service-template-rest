# Agent Contract Refresh Technical Design Phase Plan

Status: completed

## Phase Scope

This phase turns the approved `spec.md` into a planning-ready design handoff for the instruction-only agent contract refresh.

Allowed writes in this phase:

- `design/overview.md`
- `design/component-map.md`
- `design/sequence.md`
- `design/ownership-map.md`
- `workflow-plans/technical-design.md`
- `workflow-plan.md`

Prohibited in this phase:

- `plan.md`
- `tasks.md`
- implementation edits under `.codex/agents/`, `.claude/agents/`, README, skills, or Go runtime files

## Local Research Mode

Research mode: local.

Inputs checked:

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/repo-architecture.md`
- `specs/agent-contract-refresh/spec.md`
- `specs/agent-contract-refresh/workflow-plan.md`
- `.codex/config.toml`
- `.codex/agents/challenger-agent.toml`
- `.claude/agents/challenger-agent.md`
- `.codex/agents/observability-agent.toml`
- README agent inventory excerpts
- runtime agent file inventory under `.codex/agents` and `.claude/agents`

No subagent fan-out was used in this technical-design session.

## Design Artifacts

Core design artifacts:

- `design/overview.md`: approved
- `design/component-map.md`: approved
- `design/sequence.md`: approved
- `design/ownership-map.md`: approved

Conditional artifacts:

- `design/data-model.md`: not expected
- `design/dependency-graph.md`: not expected
- `design/contracts/`: not expected
- `test-plan.md`: not expected
- `rollout.md`: not expected

Rationale: the task changes instruction files and documentation only; it does not change persisted data, Go package dependencies, runtime contracts, deployment choreography, or validation obligations that need a separate test-plan artifact.

## Completion Marker

Completed when:

- required design artifacts exist and are approved,
- conditional artifacts are explicitly not expected,
- master `workflow-plan.md` points the next session to `planning`,
- no planning-critical technical design blocker remains.

Completion status: satisfied.

## Stop Rule

Session boundary reached: yes.

Do not begin `plan.md`, `tasks.md`, implementation, or validation in this session.

## Next Action

Next session starts with: `planning`.

The planning session should use `planning-session` or `planning-and-task-breakdown` to create `plan.md`, `tasks.md`, and any post-code phase workflow files needed before implementation starts.

## Blockers

No active blockers.

Reopen technical design or specification if planning decides to include canonical-source generation, CI drift checks, new review skills, model/reasoning policy, or workflow-document rewrites in the same task cycle.
