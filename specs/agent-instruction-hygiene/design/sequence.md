# Sequence

## Authoring Flow

1. Edit canonical Codex agent files in `.codex/agents/*.toml`.
2. Run `make agents-sync` to regenerate `.claude/agents/*.md`.
3. Run `make agents-check` in CI or local validation to detect semantic mirror drift.
4. Edit canonical skills under `.agents/skills`.
5. Run `make skills-sync` and `make skills-check` to refresh runtime skill mirrors.

## Subagent Runtime Flow

1. The orchestrator uses `docs/subagent-brief-template.md` to prepare a read-only lane prompt.
2. The prompt names one agent and at most one skill.
3. The agent reads its domain-specific instructions and the shared contract.
4. If a skill is selected, the skill owns detailed procedure/output shape.
5. Otherwise, the shared fan-in envelope and the agent's domain-specific output delta guide the response.

## Failure Points

- If `.claude/agents` diverges from `.codex/agents`, `make agents-check` fails.
- If skill mirrors miss the new review skills, `make skills-check` fails.
- If model/reasoning fields drift out of the documented policy, targeted validation checks fail.
