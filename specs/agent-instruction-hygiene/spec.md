# Agent Instruction Hygiene Spec

## Context

The audit found that the repository's orchestration-first/spec-first model is strong, but the agent portfolio has drift and maintenance risks: repeated boilerplate in agent files, no agents mirror check comparable to skills sync, review-skill gaps for delivery/distributed/observability, implicit model/reasoning policy, aggressive fan-out ceiling, missing reusable subagent brief template, inconsistent fan-in envelope, and an unrecorded compatibility check for `.codex/config.toml` agent registry wiring.

## Scope / Non-goals

In scope:
- repository instruction, config, docs, and tooling changes that address those findings,
- new review skills only for the three named gaps,
- mirror tooling for `.codex/agents` to `.claude/agents`,
- README and command docs updates.

Out of scope:
- service runtime or Go business-code changes,
- full prompt rewrite of every subagent,
- adding write-capable subagents,
- replacing the existing skill sync system.

## Constraints

- `AGENTS.md` remains the compact controlling contract.
- Subagents stay advisory and read-only.
- Skill bodies own detailed procedure only when selected; agent files own scope and routing.
- `.agents/skills` remains the canonical skill source for skill mirrors.
- `.codex/agents` is the canonical source for Claude agent mirrors unless a future source generator replaces it.
- `.codex/config.toml` registry wiring is retained because official Codex config reference documents `agents.<name>.config_file`.

## Decisions

- Add `docs/subagent-contract.md` as the shared per-agent invariant and fan-in envelope so individual agent files can point to one common surface instead of repeating every rule.
- Add `docs/subagent-brief-template.md` as the reusable orchestrator prompt template for read-only specialist lanes.
- Keep `max_threads = 20` per repository preference and keep `max_depth = 1`.
- Pin high-scrutiny roles to `gpt-5.4` with `high` reasoning: `challenger-agent`, `security-agent`, `reliability-agent`, and `quality-agent`.
- Pin read-heavy specialist roles to `gpt-5.4-mini` with `medium` reasoning: architecture, API, data, delivery, design-integrator, distributed, domain, observability, performance, and QA.
- Pin concurrency to `gpt-5.4` with `medium` reasoning because concurrent behavior review is small enough to justify stronger local reasoning.
- Add `go-devops-review`, `go-distributed-review`, and `go-observability-review` as concise instruction-only review skills.
- Route delivery, distributed, and observability agents to those review skills in review mode.
- Add `scripts/dev/sync-agents.sh`, `make agents-sync`, `make agents-check`, and Docker/CI integration.
- Update README and build-command docs to make the agent mirror check and new review coverage visible.

## Open Questions / Assumptions

- Future iteration may generate both `.codex/agents` and `.claude/agents` from a neutral source, but the first fix uses `.codex/agents` as canonical to minimize churn.
- Review skills start without references; add reference files after repeated review examples justify them.

## Plan Summary / Link

Implementation follows `plan.md` and `tasks.md` in this task bundle.

## Validation

- `make agents-check` passed.
- `make skills-check` passed.
- `make guardrails-check` passed.
- `git diff --check` passed.
- `bash -n scripts/dev/sync-agents.sh scripts/dev/sync-skills.sh scripts/dev/setup.sh scripts/dev/docker-tooling.sh scripts/ci/required-guardrails-check.sh` passed.
- `python3` `tomllib` parse of `.codex/config.toml` and all `.codex/agents/*.toml` passed.
- Targeted `rg` checks found no stale "no dedicated review skill" / old mirror-drift wording, and confirmed new delivery/distributed/observability review routes.
- Full `make check` was not run because the change is docs/config/scripts/skill instructions only and the narrower mirror, guardrail, syntax, and TOML checks cover the changed surfaces.

## Outcome

All audit findings are addressed in the repository surfaces: shared subagent contract and brief template are added, review-skill gaps are closed, model/reasoning and fan-out policy are explicit, `.codex` to `.claude` agent mirror checks are automated, skill mirrors are synced, and docs now describe the updated operating model.
