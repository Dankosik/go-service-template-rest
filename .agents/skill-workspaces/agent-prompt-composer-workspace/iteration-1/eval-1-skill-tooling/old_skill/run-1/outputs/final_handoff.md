Objective
Revise the repo-local `agent-prompt-composer` skill so it reconstructs messy user input into a strong English handoff prompt for downstream coding agents, regardless of input language, instead of sounding like a Russian-only translator.

User Intent And Context
- The user wants a local skill in this repository, not a global/home-directory skill.
- The skill should handle messy, incomplete, repetitive, dictation-style, and mixed-language input such as English/Russian/whatever.
- The output should be a normal English prompt for `codex` or `claude` that already works in this repo.
- This is not a plain translation task; the core goal is intent reconstruction, repository grounding, likely file targeting, and validation guidance.

Confirmed Signals And Exact Identifiers
- `agent-prompt-composer`
- `skill`
- `messy user input`
- `after dictation`
- `missing context`
- `mixed English/Russian/whatever`
- `normal English prompt`
- `codex`
- `claude`
- `skills-sync`
- `examples`
- `evals`
- local repo skill, not global

Relevant Repository Context
- Canonical local skills live under `.agents/skills/`.
- Skill mirrors are maintained by `scripts/dev/sync-skills.sh`.
- Repository-wide workflow rules live in `AGENTS.md`.
- The repo uses a spec-first / orchestrator-first workflow; `docs/spec-first-workflow.md` is the detailed companion when workflow context matters.
- The prompt-composer skill already has local references and eval fixtures under `.agents/skills/agent-prompt-composer/` and `.agents/skills/agent-prompt-composer-workspace/`.

Inspect First
- `.agents/skills/agent-prompt-composer/SKILL.md`
- `.agents/skills/agent-prompt-composer/references/repo-profile.md`
- `.agents/skills/agent-prompt-composer/references/context-selection.md`
- `.agents/skills/agent-prompt-composer/references/example-transformations.md`
- `scripts/dev/sync-skills.sh`
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- likely `.agents/skills/agent-prompt-composer/evals/`

Requested Change / Problem Statement
- Update the existing repo-local prompt-composer skill so it is clearly about transforming any messy task input into a high-signal English handoff, not about one language.
- Preserve exact technical signals from the source input, including paths, commands, filenames, skill names, eval references, and repo-specific terms.
- Make the skill explicitly reconstruct intent, separate source-task notes from wrapper or injection noise, infer the smallest useful repo boundary, and include likely files plus validation steps for the downstream agent.
- Keep the output oriented toward another coding agent that will immediately inspect the right repo surfaces.

Constraints / Preferences / Non-goals
- Keep the skill local to this repository.
- Do not make the skill a literal sentence-by-sentence translation layer.
- Do not erase uncertainty; put unresolved ambiguity into `Assumptions / Open Questions`.
- Include repo-aware guidance, but do not dump broad repository summaries.
- Preserve existing skill conventions and update mirrors/examples/evals consistently if the skill content changes.

Acceptance Criteria / Expected Outcome
- The skill description and body clearly say it handles messy, incomplete, repetitive, multilingual input in general.
- The output style is a clean English handoff prompt with intent reconstruction, repo grounding, likely starting files, and validation direction.
- The skill no longer reads as if Russian is the special case.
- Skill references, examples, and eval fixtures stay aligned with the revised behavior.
- Repo-local mirrors remain refreshable through the existing sync flow.

Validation / Verification
- Run `make skills-sync`.
- Run `make skills-check`.
- Review the updated skill output against representative messy fixtures, including mixed-language input and dictation-like noise.
- Check that the final handoff still points to the right repo surfaces and does not collapse into generic prompt polish.

Assumptions / Open Questions
- [assumption] The right scope is the repo-local `agent-prompt-composer` skill and its mirrors, not a new skill or a global install.