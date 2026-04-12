# Agent Contract Refresh Spec

Created: 2026-04-12
Status: approved for technical-design handoff

## Context

The user reviewed the repository subagent portfolio across `.codex/agents`, `.claude/agents`, workflow docs, skills, and Codex/OpenAI agent guidance. The review concluded that the repository already has a strong orchestrator-first, read-only subagent model, but it has one blocking runtime-contract drift and several ordered cleanup opportunities.

Repository intake confirmed:

- `.codex/agents/challenger-agent.toml` and `.claude/agents/challenger-agent.md` currently define `challenger-agent` as a `pre-spec-challenge` role only.
- `AGENTS.md`, `docs/spec-first-workflow.md`, README, `.agents/skills/workflow-plan-adequacy-challenge/SKILL.md`, and `.agents/skills/spec-clarification-challenge/SKILL.md` require or describe challenger use for `workflow-plan-adequacy-challenge` and `spec-clarification-challenge` as well.
- `.codex/config.toml` registers `observability-agent` and `.codex/agents/observability-agent.toml` exists, but `.claude/agents/observability-agent.md` is missing.
- README says project-scoped agents live in both `.codex/agents/` and `.claude/agents/`, but its agent table currently omits `observability-agent`.
- Existing agent files have `Return` sections, but they are mostly topic lists rather than a consistent fan-in shape with evidence anchors and confidence.
- Existing agent files do not have a consistent `Inspect first` / source-of-truth section.

## Scope / Non-goals

In scope:

- Refresh project-scoped subagent instruction files under `.codex/agents/` and `.claude/agents/`.
- Align `challenger-agent` with the three challenge gates already required by repository workflow: `workflow-plan-adequacy-challenge`, `pre-spec-challenge`, and `spec-clarification-challenge`.
- Resolve the `observability-agent` Codex/Claude mirror and README inventory drift.
- Standardize a minimal agent instruction shape: `Mission`, `Use when`, `Do not use when`, `Inspect first`, `Mode routing`, `Return`, and `Escalate when`.
- Standardize fan-in-oriented return contracts with evidence anchors and confidence.
- Add concise, role-specific `Inspect first` source-of-truth blocks.
- Trim repeated global policy only after role-local contracts and routing are explicit.
- Record later drift-policy, review-skill, nickname, and model-tuning opportunities without forcing them into the first implementation slice.

Out of scope for this task unless reopened explicitly:

- Go service runtime behavior, API contracts, DB schema, generated service code, migrations, or application tests.
- Creating new review skills such as `go-observability-review`, `go-delivery-review`, or `go-distributed-review`.
- Introducing a canonical agent-instruction generator, CI drift-check, or generated mirror workflow.
- Adding model or reasoning-effort overrides before a separate policy/support check.
- Rewriting `AGENTS.md` or `docs/spec-first-workflow.md` unless implementation finds concrete drift caused by this task.

## Constraints

- Subagents remain read-only and advisory; final decisions stay with the orchestrator.
- Each subagent pass keeps the repository invariant of at most one skill per pass.
- `.agents/skills` remains the canonical repository skill source; this task updates agent routing to existing skills rather than redesigning skills.
- Codex and Claude runtime agent files should have equivalent role semantics when both runtimes expose the same agent.
- Shared return shapes should help orchestrator fan-in without turning every agent into a rigid schema machine.
- `Inspect first` blocks should be short and role-specific; they should guide repository lookup, not duplicate `docs/repo-architecture.md`.
- The specification clarification gate is waived for this lightweight-local specification pass only because the current scope is instruction-only, bounded by the user's review plus repository intake, and no user-requested subagent fan-out is active in this session. If scope expands into new skills, CI, generated mirrors, or conflicting runtime policy, reopen the gate with `challenger-agent` and exactly one skill: `spec-clarification-challenge`.

## Decisions

1. First implementation slice: fix `challenger-agent` in both `.codex/agents/challenger-agent.toml` and `.claude/agents/challenger-agent.md`.
   - The role must explicitly own three modes: `workflow-plan-adequacy-challenge`, `pre-spec-challenge`, and `spec-clarification-challenge`.
   - The instructions must distinguish when each mode is used, what to inspect first, and what output each mode returns.
   - The role must remain non-review, read-only, advisory, and one-skill-per-pass.

2. Mirror inventory slice: resolve `observability-agent` drift by adding the missing Claude runtime agent and updating README inventory.
   - Preferred outcome: create `.claude/agents/observability-agent.md` semantically equivalent to `.codex/agents/observability-agent.toml`, and add the README row/link for `observability-agent`.
   - Reopen specification if implementation discovers an intentional Codex-only policy for `observability-agent`.

3. Return-format slice: use a shared minimum fan-in contract for agent outputs.
   - Research/adjudication lanes should return: `Conclusion`, `Evidence`, `Open risks`, `Recommended handoff`, and `Confidence`.
   - Review lanes should return: `Findings by severity`, `Evidence`, `Why it matters`, `Validation gap`, `Handoff`, and `Confidence`.
   - Agent-specific return sections may add role-specific fields only when they improve fan-in for that role.

4. Inspect-first slice: add concise source-of-truth blocks to agent files.
   - The block should name 3-6 starting surfaces or artifacts for the role.
   - Examples to preserve for later design/planning: API agents start with `api/openapi/service.yaml`, `internal/api/`, `internal/infra/http/`, and current task `spec.md`; data agents start with `env/migrations/`, `internal/infra/postgres/`, generated SQLC surfaces, and relevant `internal/app/*`; lifecycle/reliability agents start with `cmd/service/main.go`, `cmd/service/internal/bootstrap/`, health/readiness/shutdown paths, and external dependency call sites.
   - The exact per-agent list belongs in technical design or planning, not in this spec.

5. Deduplication slice: after the role-local contracts are explicit, reduce duplicated global policy in agent files where it is safe.
   - Keep role-specific boundaries, handoffs, skill routing, and escalation rules.
   - Do not remove constraints whose absence would make an individual agent unsafe or ambiguous outside the full repo context.

6. Drift-policy slice: do not implement canonical source generation or CI drift-checking in the first task cycle.
   - Later planning may include a short decision checkpoint to either keep manual mirrors plus an inventory check, or open a separate spec for canonical-source/generation/CI drift-check work.

7. Backlog-only items:
   - Dedicated review skills for observability, delivery, and distributed roles are separate skill-design work.
   - Nickname additions are optional ergonomics after semantic contract cleanup.
   - Model/reasoning-effort overrides need a separate policy/support check before implementation.

## Open Questions / Assumptions

- [assumption] The observed `observability-agent` Codex-only state is accidental drift, not an intentional runtime split.
- [assumption] The first task cycle can remain instruction-only and does not require Go tests, migrations, generated code checks, or runtime validation.
- [reopen_spec_if_false] If a canonical agent-instruction source or CI drift-check is required in this same task, return to specification or technical design before planning.
- [reopen_spec_if_false] If Codex and Claude agent formats cannot preserve equivalent semantics without format-specific wording, technical design must record the runtime-specific constraints before planning.

## Plan Summary / Link

This spec approves an ordered implementation direction but does not replace `plan.md` or `tasks.md`.

Approved high-level slices for later planning:

1. `challenger-agent` three-mode contract fix.
2. `observability-agent` Claude mirror and README inventory repair.
3. Shared return-format contract.
4. Role-specific `Inspect first` blocks.
5. Safe deduplication of repeated global policy.
6. Drift-policy checkpoint or separate follow-up spec.
7. Optional ergonomics/backlog checkpoint.

Next session should start with `technical-design` or an explicit design-skip decision for this instruction-only task, then planning can produce `plan.md` and `tasks.md`.

## Validation

Future validation should prove:

- `.codex/agents/challenger-agent.toml` and `.claude/agents/challenger-agent.md` both mention and route all three challenge skills.
- Codex and Claude agent inventories either match or document any intentional runtime-only agent.
- README agent inventory matches the project-scoped agent files it claims to list.
- Agent files contain the agreed return-contract fields and concise `Inspect first` blocks.
- `.toml` agent files remain parseable and Markdown agent files remain readable.
- No Go runtime files, service contracts, migrations, or generated code are changed for this instruction-only task.

## Outcome

Pending implementation.
