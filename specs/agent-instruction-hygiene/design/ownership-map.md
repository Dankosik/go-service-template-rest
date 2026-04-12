# Ownership Map

| Owner | Responsibility |
| --- | --- |
| `AGENTS.md` | Repository-wide authority and hard workflow invariants. |
| `docs/spec-first-workflow.md` | Detailed task artifact mechanics and resume rules. |
| `docs/subagent-contract.md` | Shared read-only subagent contract, fan-in envelope, and brief-quality expectations. |
| `.codex/agents/*.toml` | Canonical per-agent mission, use/do-not-use, inspect-first surfaces, and skill routing. |
| `.claude/agents/*.md` | Generated runtime mirror of Codex agent files for Claude Code. |
| `.agents/skills` | Canonical skill implementations and review/spec playbooks. |
| `scripts/dev/sync-agents.sh` | Agent mirror sync/check logic. |
| `scripts/dev/sync-skills.sh` | Skill mirror sync/check logic. |
| Orchestrator | Final decisions, lane selection, synthesis, implementation, and validation. |
| Subagents | Read-only advisory research/review inside assigned scope. |
