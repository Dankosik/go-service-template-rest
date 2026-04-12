# Design Overview

## Approach

Treat Codex agent files as the canonical authored agent surface, with Claude agent files as generated mirrors. Centralize shared expectations in docs and checks, while keeping domain-specific scope and skill routing in each agent file.

## Artifact Index

- `docs/subagent-contract.md`: shared invariant and fan-in envelope for all read-only subagents.
- `docs/subagent-brief-template.md`: reusable orchestrator lane brief template.
- `.codex/config.toml`: explicit fan-out and model/reasoning policy.
- `.codex/agents/*.toml`: canonical per-agent scope/routing.
- `.claude/agents/*.md`: generated mirrors from Codex agent files.
- `.agents/skills/go-*-review/SKILL.md`: canonical review-skill sources.
- `scripts/dev/sync-agents.sh`: mirror sync/check implementation.

## Readiness

Ready for implementation. The design is docs/config/tooling-only and does not change runtime service behavior.
