**Objective**
Rewrite the repository-local `agent-prompt-composer` skill so it handles messy user input in any language, not just Russian, and produces a high-signal English handoff prompt for downstream coding agents.

**User Intent And Context**
- The task is about local skill/tooling inside this repo, not a global/home-directory skill.
- The current skill sounds too language-specific and should be reframed as a general intent-reconstruction tool for rough, incomplete, repetitive, dictation-style, or multilingual task input.
- The output should help another agent understand the real task, repo context, likely files, and validation path without re-reading the raw noise.

**Confirmed Signals And Exact Identifiers**
- `agent-prompt-composer`
- `skill`
- `messy user input`
- `English handoff prompt`
- `intent reconstruction`
- `repo context`
- `likely files`
- `validation steps`
- `codex`
- `claude`
- `skills-sync`
- `evals`
- `examples`
- `mirrors`
- `local`, not global

**Relevant Repository Context**
- Canonical local skills live under [`.agents/skills/`](/Users/daniil/Projects/Opensource/go-service-template-rest/.agents/skills/).
- Runtime mirrors are maintained by [`scripts/dev/sync-skills.sh`](/Users/daniil/Projects/Opensource/go-service-template-rest/scripts/dev/sync-skills.sh).
- The repo contract says skills are source-managed locally and mirrors can be refreshed with `make skills-sync` / checked with `make skills-check`.
- `AGENTS.md` says `.agents/skills/` is the canonical local skill source and mirrors should stay in sync.
- The `agent-prompt-composer` skill already has references, examples, and eval fixtures under:
  - [`.agents/skills/agent-prompt-composer/SKILL.md`](/Users/daniil/Projects/Opensource/go-service-template-rest/.agents/skills/agent-prompt-composer/SKILL.md)
  - [`.agents/skills/agent-prompt-composer/references/repo-profile.md`](/Users/daniil/Projects/Opensource/go-service-template-rest/.agents/skills/agent-prompt-composer/references/repo-profile.md)
  - [`.agents/skills/agent-prompt-composer/references/context-selection.md`](/Users/daniil/Projects/Opensource/go-service-template-rest/.agents/skills/agent-prompt-composer/references/context-selection.md)
  - [`.agents/skills/agent-prompt-composer/references/example-transformations.md`](/Users/daniil/Projects/Opensource/go-service-template-rest/.agents/skills/agent-prompt-composer/references/example-transformations.md)
  - [`.agents/skills/agent-prompt-composer/evals/files/`](/Users/daniil/Projects/Opensource/go-service-template-rest/.agents/skills/agent-prompt-composer/evals/files/)
  - [`.agents/skills/agent-prompt-composer/evals/evals.json`](/Users/daniil/Projects/Opensource/go-service-template-rest/.agents/skills/agent-prompt-composer/evals/evals.json)

**Inspect First**
- [`.agents/skills/agent-prompt-composer/SKILL.md`](/Users/daniil/Projects/Opensource/go-service-template-rest/.agents/skills/agent-prompt-composer/SKILL.md)
- [`.agents/skills/agent-prompt-composer/references/example-transformations.md`](/Users/daniil/Projects/Opensource/go-service-template-rest/.agents/skills/agent-prompt-composer/references/example-transformations.md)
- [`.agents/skills/agent-prompt-composer/evals/files/skill-tooling.md`](/Users/daniil/Projects/Opensource/go-service-template-rest/.agents/skills/agent-prompt-composer/evals/files/skill-tooling.md)
- [`.agents/skills/agent-prompt-composer/evals/files/harness-instruction-noise.md`](/Users/daniil/Projects/Opensource/go-service-template-rest/.agents/skills/agent-prompt-composer/evals/files/harness-instruction-noise.md)
- [`.agents/skills/agent-prompt-composer/evals/files/unconfirmed-path-grounding.md`](/Users/daniil/Projects/Opensource/go-service-template-rest/.agents/skills/agent-prompt-composer/evals/files/unconfirmed-path-grounding.md)
- [`scripts/dev/sync-skills.sh`](/Users/daniil/Projects/Opensource/go-service-template-rest/scripts/dev/sync-skills.sh)
- [`AGENTS.md`](/Users/daniil/Projects/Opensource/go-service-template-rest/AGENTS.md)

**Requested Change / Problem Statement**
- Revise the local `agent-prompt-composer` skill so it is explicitly about composing a normal English handoff from messy, incomplete, repetitive, multilingual, or dictation-style user input.
- Preserve intent reconstruction, repo grounding, exact identifiers, likely files, and validation guidance as first-class outputs.
- Remove any implication that the skill is mainly for Russian-language input.
- Keep the skill aligned with the repo’s local skill ecosystem, including mirrors, examples, and eval fixtures.

**Constraints / Preferences / Non-goals**
- Keep the work local to this repository.
- Do not turn this into a generic translation or copy-editing skill.
- Do not invent repository facts or hide uncertainty.
- Preserve exact technical identifiers, paths, commands, and error text when they appear in source notes.
- Reuse existing skill/mirror/eval conventions instead of creating a parallel format.
- Keep the downstream prompt actionable for codex or claude agents working in this repo.

**Acceptance Criteria / Expected Outcome**
- The skill text clearly describes a general messy-input-to-English-handoff transformation, independent of source language.
- The skill emphasizes intent reconstruction, context extraction, deduplication, repo-aware grounding, and validation guidance.
- The existing examples and eval fixtures are updated or expanded so they reflect multilingual, noisy, and instruction-noise cases.
- Any mirrored skill copies remain consistent with the canonical local skill after sync.
- The resulting handoff prompt style is clearly better than literal translation or surface-level prompt polishing.

**Validation / Verification**
- Run `make skills-sync`.
- Run `make skills-check`.
- Review `evals/evals.json` and the relevant fixture files for consistency.
- Use the existing examples/evals to confirm the new skill handles multilingual and noisy input without losing exact identifiers or repo grounding.

**Assumptions / Open Questions**
- [assumption] The intended scope includes the canonical local skill plus any synced mirrors and eval/example assets that keep it discoverable and consistent.