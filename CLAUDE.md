# CLAUDE.md

@AGENTS.md

Claude Code sessions follow the same repository contract as every other harness. [docs/agent-harness.md](docs/agent-harness.md) maps workflow concepts — durable execution control, implementation workers, read-only subagent lanes, model selection, reasoning effort — to Claude Code's native controls. Project subagents live in `.claude/agents/`; project skills are the canonical `.agents/skills/` set, exposed to Claude Code through per-skill symlinks in `.claude/skills/` (resync with `make claude-skills-sync` after adding or removing a skill).
