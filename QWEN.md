# QWEN.md

Qwen Code sessions follow the same repository contract as every other harness. Qwen Code loads `AGENTS.md` automatically, so this file does not re-import it; it only records Qwen-specific wiring. [docs/agent-harness.md](docs/agent-harness.md) maps workflow concepts — durable execution control, implementation workers, read-only subagent lanes, model selection, reasoning effort — to Qwen Code's native controls. Project subagents live in `.qwen/agents/`; project skills are the canonical `.agents/skills/` set, which Qwen Code discovers natively (no per-skill symlinks are needed).
