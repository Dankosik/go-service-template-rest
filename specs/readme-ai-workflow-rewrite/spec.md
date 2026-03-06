## Context
- The repository already has a spec-first, orchestrator/subagent-first workflow contract in `AGENTS.md` and supporting guidance in `docs/spec-first-workflow.md`.
- The previous top-level `README.md` mentioned agent-centric workflow, but its primary positioning still read like a conventional Go REST template with AI workflow as a secondary concern.
- The user wants the README rewritten for AI-native development, especially for developers using LLMs, coding agents, Codex, Claude Code, Cursor-like runtimes, and similar tools.
- The user also wants:
  - solo-builder wording instead of team-centric wording;
  - explicit descriptions of available subagents and skills;
  - readable tables for those roles and capabilities;
  - inspiration from AI-native repository READMEs rather than generic backend-template docs.

## Scope / Non-goals
- In scope:
  - rewrite `README.md` so the primary story is agentic Go service development rather than the raw stack;
  - explain the orchestrator/subagent workflow in concise developer-facing language;
  - document the project-scoped agent portfolio in `.codex/agents/` and `.claude/agents/`;
  - document the repository-native skills in `skills/` and explain how they differ from subagents;
  - include concrete usage examples for Codex and Claude Code;
  - replace team-centric wording with solo-builder-friendly language;
  - preserve the stack, commands, and repo links, but move them behind the workflow story.
- Non-goals:
  - changing the workflow contract in `AGENTS.md`;
  - changing any agent or skill definition;
  - turning the README into a generic multi-agent manifesto detached from the actual Go template.

## Constraints
- The README must stay aligned with real files, commands, agent names, and skill names in the repository.
- The README should remain primarily English because the repo’s top-level docs are English and the user did not request a language switch.
- Hype is acceptable only if the text still reads as technically credible for a real Go backend template.
- Tables should be readable and grouped; a single giant undifferentiated skill catalog is not acceptable.

## Decisions
- The README will lead with an **AI-native Go REST template** position instead of a generic “production-ready template” pitch.
- The hero will use solo-builder wording closer to “solo developers”, “you”, and “your coding agent” rather than “teams”.
- The first screen will emphasize:
  - orchestrator-first workflow;
  - project-scoped agents;
  - portable skills;
  - a real Go/OpenAPI/Postgres backend stack underneath.
- The workflow section will show the actual repository loop: `intake -> research -> synthesis -> planning -> implementation -> review -> validation`.
- The README will explicitly explain the difference between:
  - **subagents**: read-only specialists for research or review;
  - **skills**: portable procedural playbooks loaded on demand.
- The agent section will use a compact table with ownership, usage timing, and expected outputs.
- The skill section will use grouped tables rather than one flat list.
- The README structure will follow this order:
  1. Hero and positioning
  2. Why this exists for solo AI-assisted backend work
  3. Workflow overview
  4. Agent portfolio
  5. Skill library
  6. Orchestrator protocol and artifact model
  7. Quickstart
  8. Repository layout
  9. Technology stack
  10. Quality gates and verification

## Open Questions / Assumptions
- Assumption: brief Codex and Claude Code examples are sufficient; the README does not need a tool-by-tool installation tutorial.
- Assumption: Markdown tables are the right presentation layer for agents and skills because the repo is small enough to keep them readable when grouped.
- Open question for a future pass: whether to add badges, visuals, or screenshots. That is out of scope here.

## Implementation Plan
1. Gather external README references and extract reusable tone/structure patterns.
   Completion criteria:
   - reference notes exist with links and condensed takeaways;
   - the notes separate solo-first tone, workflow framing, and table structure patterns.
2. Reframe the hero and “why” section.
   Completion criteria:
   - team-centric wording is removed;
   - the opening reads like an AI-native repo for solo developers and their coding agents.
3. Expand the README with explicit subagent and skill documentation.
   Completion criteria:
   - a full agent table exists;
   - grouped skill tables cover the repository-native skills;
   - the distinction between subagents and skills is explicit.
4. Keep the workflow, quickstart, stack, and quality-gate sections aligned with the actual repo.
   Completion criteria:
   - commands and paths in the README match real files and targets;
   - stack and verification sections remain concise and secondary to workflow.
5. Validate the rewritten README against the current repository.
   Completion criteria:
   - referenced files and directories exist;
   - named make targets exist;
   - the locally installed `claude` CLI exposes `--agent`;
   - the locally installed `codex` CLI exposes the expected project-oriented commands.

## Validation
- `sed -n '1,320p' README.md`
- `printf 'Agents:\n'; ls -1 .codex/agents | sed 's/\\.toml$//'`
- `printf 'Skills:\n'; ls -1 .agents/skills`
- `rg -n "^(bootstrap|template-init|check|check-full|run|ci-local|docker-ci|openapi-check|sqlc-check|test-integration|gh-protect):" Makefile`
- `for p in AGENTS.md CLAUDE.md docs/spec-first-workflow.md .codex/config.toml api/openapi/service.yaml docs/project-structure-and-module-organization.md go.mod go.sum; do test -e "$p" || echo "missing:$p"; done`
- `for d in .codex/agents .claude/agents skills specs .agents/skills .claude/skills .cursor/skills .gemini/skills .github/skills .opencode/skills; do test -e "$d" || echo "missing:$d"; done`
- `claude --help | sed -n '1,220p'`
- `codex --help | sed -n '1,220p'`

## Outcome
- Rewrote `README.md` around an AI-native, solo-builder-friendly narrative.
- Removed the team-centric hero and replaced it with solo-developer wording.
- Added a full subagent table and grouped repository-native skill tables.
- Kept the orchestrator workflow, quickstart, repo layout, stack, and quality-gate sections aligned with the actual repository.
- Validated that the referenced agent directories, skill mirrors, files, and make targets exist, and confirmed that the local `claude` CLI exposes `--agent` while the local `codex` CLI exposes the expected command surface.
