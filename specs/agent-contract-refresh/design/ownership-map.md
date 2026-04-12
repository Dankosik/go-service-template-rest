# Ownership Map

Status: approved

## Source-Of-Truth Boundaries

| Decision Or Contract | Owner | Consumers |
| --- | --- | --- |
| Repository orchestration rules, read-only subagent boundary, one-skill-per-pass invariant, phase gates | `AGENTS.md` | Runtime agent instructions, workflow docs, task-local specs and plans |
| Workflow artifact mechanics and phase handoff examples | `docs/spec-first-workflow.md` | Task-local workflow plans and design/planning sessions |
| Skill protocols and allowed skill-specific behavior | `.agents/skills/*/SKILL.md` | Runtime agent `Mode routing` / `Skill policy` sections |
| Codex runtime agent metadata and instructions | `.codex/agents/*.toml` plus `.codex/config.toml` | Codex project-scoped subagent runtime |
| Claude runtime agent metadata and instructions | `.claude/agents/*.md` | Claude Code project-scoped subagent runtime and README links |
| Human-facing agent inventory and examples | `README.md` | Repository users and future planning sessions |

## Mirror Policy For This Task

Codex and Claude agent files should preserve equivalent role semantics when both runtimes expose the same agent:

- same mission and ownership boundary,
- same use/do-not-use routing intent,
- same one-skill-per-pass policy,
- same `Inspect first` source-of-truth intent,
- same fan-in return contract,
- same escalation and handoff boundaries.

Format-specific metadata may differ:

- Codex files use TOML fields such as `name`, `description`, `sandbox_mode`, optional `nickname_candidates`, and a multi-line `developer_instructions` string.
- Claude files use Markdown frontmatter such as `name`, `description`, `tools`, followed by Markdown body instructions.

## Orchestrator Boundary

The agent files remain advisory role instructions. They must not:

- make subagents final decision owners,
- instruct subagents to edit code or repository artifacts,
- let a single lane use multiple skills in one pass,
- turn `challenger-agent` into a generic reviewer or mini-orchestrator,
- replace task-local `spec.md`, `design/`, `plan.md`, or `tasks.md`.

## Backlog Ownership

These items are intentionally not owned by the first implementation cycle:

- new review skills for observability, delivery, or distributed roles,
- canonical agent-instruction generation or CI drift checks,
- model or reasoning-effort overrides,
- broad workflow-document rewrites.

Planning may create explicit checkpoint tasks to record or defer them, but implementation should not start those changes without reopening the appropriate earlier phase.
