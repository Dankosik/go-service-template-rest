# Portable Agent Skills in This Repository

This repository keeps runnable skills in provider runtime directories so they can be executed directly by agent tools.

## Goal

- Keep `docs/skills/` as documentation-only.
- Store executable `SKILL.md` where agents actually load them.
- Keep the same `SKILL.md` format across providers.

## Repository layout

Runnable skill locations:

```text
.agents/skills/
.claude/skills/
.gemini/skills/
.github/skills/
.cursor/skills/
```

Documentation-only location:

```text
docs/skills/
```

`docs/skills/` is for guides, specifications, and writing rules only.
Do not store runnable `SKILL.md` files there.

## Why these paths

The paths above align with provider docs and give practical cross-tool compatibility:

- OpenAI Codex: repository/user/admin/system skill locations and `.agents/skills` repo scanning.
- Anthropic Claude Code: `.claude/skills` for project skills and `~/.claude/skills` for personal skills.
- Cursor: `.agents/skills`, `.cursor/skills`, and compatibility loading from Claude/Codex paths.
- GitHub Copilot / VS Code: project skills in `.github/skills` (plus `.claude/skills` and `.agents/skills` support in VS Code docs).
- Gemini CLI: `.gemini/skills` and `.agents/skills` alias.

## Authoring rules for portability

- Keep required frontmatter minimal and standard: `name`, `description`.
- Keep instructions agent-agnostic in `SKILL.md` body.
- Do not put provider-specific behavior in the core workflow unless clearly optional.
- Keep vendor-specific metadata additive only (for example `agents/openai.yaml`) so other tools can ignore it safely.
- Keep equivalent skills consistent across runtime directories when cross-tool behavior must match.

## References

- OpenAI Codex skills docs: https://developers.openai.com/codex/skills/
- Anthropic Claude Code skills docs: https://code.claude.com/docs/en/skills
- Cursor skills docs: https://cursor.com/docs/context/skills
- VS Code Copilot agent skills docs: https://code.visualstudio.com/docs/copilot/customization/agent-skills
- GitHub Copilot agent skills concept: https://docs.github.com/en/copilot/concepts/agents/about-agent-skills
- Gemini CLI skills docs: https://geminicli.com/docs/cli/skills/
- Agent Skills open standard overview: https://agentskills.io/home
