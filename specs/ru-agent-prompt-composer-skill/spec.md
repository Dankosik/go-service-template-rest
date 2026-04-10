## Context

The repository already has a native, version-controlled skill system:
- `.agents/skills` is the canonical authoring surface for runnable `SKILL.md` definitions.
- `scripts/dev/sync-skills.sh` mirrors canonical skills into `.claude/skills`, `.cursor/skills`, `.gemini/skills`, `.github/skills`, and `.opencode/skills`.
- `README.md` is the human-facing overview for local repository skills.

The requested capability is a repository-local skill that turns rough Russian text into a strong English prompt for another coding agent working in this same repository. This is not generic translation. The skill must reconstruct intent, classify the task, select only relevant repository context, and compose a coding-agent-ready prompt that assumes repo access and tool competence.

## Scope / Non-goals

In scope:
- discover the repository’s native skill conventions and reuse them
- design and implement a new repository-local skill in the canonical repo format
- make the skill repository-aware instead of generic
- include concise usage/supporting artifacts only where they materially improve maintainability
- validate the skill against realistic messy Russian inputs with ambiguity, repetition, and dictation artifacts

Non-goals:
- building a global or user-home skill outside this repository
- creating a generic Russian-to-English translator divorced from repository context
- adding heavyweight automation, external services, or new runtime dependencies unless clearly justified
- rewriting unrelated repository skills or docs outside the surfaces needed to add this capability

## Constraints

- Execution shape: `full orchestrated`
- Research mode: `fan-out`
- Pre-spec challenge: `required`
  Rationale: the skill needs to balance repo-aware inference with hallucination avoidance and must fit existing repository conventions without creating a parallel skill system.
- Subagents stay read-only and advisory.
- Final runnable skill content must live canonically under `.agents/skills` and then be mirrored with the repository’s sync flow.
- The output of the skill must be English-only, high-signal, and specifically useful for downstream coding agents already operating in this repository.

## Decisions

1. The new skill will be named `ru-agent-prompt-composer`.
   - Canonical path: `.agents/skills/ru-agent-prompt-composer/`
   - Runtime mirrors: `.claude/skills/`, `.cursor/skills/`, `.gemini/skills/`, `.github/skills/`, `.opencode/skills/`

2. The skill will stay repo-local and lightweight, but not bare.
   - Include:
     - `SKILL.md`
     - `references/repo-profile.md`
     - `references/context-selection.md`
     - `references/example-transformations.md`
     - `evals/evals.json`
     - `evals/files/*.md`
   - Exclude:
     - scripts
     - external dependencies
     - workspace or benchmark artifacts that would be mirrored into every runtime directory

3. The skill output contract will explicitly separate grounded signals from inference.
   Usual section order:
   - `Objective`
   - `Confirmed Signals And Exact Identifiers`
   - `Relevant Repository Context`
   - `Inspect First`
   - `Requested Change / Problem Statement`
   - `Constraints / Preferences / Non-goals`
   - `Acceptance Criteria / Expected Outcome`
   - `Validation / Verification`
   - `Assumptions / Open Questions`
   Empty sections should be omitted rather than padded.

4. The skill will always load compact static references and only inspect live repo surfaces under bounded triggers.
   - Always load:
     - `references/repo-profile.md`
     - `references/context-selection.md`
   - Inspect live repo files only when at least one is true:
     - the raw input names a concrete file, package, module, command, endpoint, error string, or test
     - the task mode is clear enough that one or two mapped repo surfaces would materially sharpen the prompt
     - a vague phrase such as “that handler” or “that readiness thing” can be resolved with high confidence from the mapped repo surface
   - Keep live lookup bounded to the named surface or the smallest mapped shortlist. If confidence stays low, record an assumption instead of broadening the search.

5. The skill will be repo-aware, not repository-heavy.
   - Stable references will capture only durable workflow/layout facts.
   - The skill will not inject template-only caveats, sample-domain trivia, or broad project summaries unless the current request needs them.
   - Exact technical identifiers from the raw input must be preserved verbatim where relevant.

6. Discoverability will be updated in the human-facing overview that matters today.
   - Update `README.md` because it is the visible skill library overview.

7. Examples and evals will have distinct roles to minimize drift.
   - `references/example-transformations.md` will be a thin human-facing guide that links to fixture files and shows curated final prompts.
   - `evals/files/*.md` will be the raw Russian fixtures for regression-style validation.
   - `evals/evals.json` will encode expected behavior with prompt descriptions and expectations focused on intent reconstruction, repo-context selection, and hallucination avoidance.

## Open Questions / Assumptions

- Assumption: a new skill is needed rather than extending an existing canonical skill, because the current skill set does not cover repo-aware prompt reconstruction from Russian input.
- Assumption: the repo’s current example-module/template state should only be mentioned by the skill when the request explicitly touches bootstrap/template initialization or module-path concerns.
- Open question: none that block implementation.

## Implementation Plan

Execution will follow [`plan.md`](plan.md).
Control summary:
1. Create the canonical skill bundle with references, examples, and eval fixtures.
2. Update human-facing discoverability surfaces.
3. Sync runtime mirrors and run targeted validation.

## Validation

Executed:
- `bash ./scripts/dev/sync-skills.sh`
- `bash ./scripts/dev/sync-skills.sh --check`
- `python3 -m json.tool .agents/skills/ru-agent-prompt-composer/evals/evals.json >/dev/null`
- `git diff --check`
- `find .claude/skills/ru-agent-prompt-composer .cursor/skills/ru-agent-prompt-composer .gemini/skills/ru-agent-prompt-composer .github/skills/ru-agent-prompt-composer .opencode/skills/ru-agent-prompt-composer -maxdepth 3 -type f | sort`

## Outcome

Completed:
- implemented the canonical skill bundle under `.agents/skills/ru-agent-prompt-composer/`
- added compact repo references for stable repo profile and task-specific context selection
- added curated example transformations plus raw eval fixtures and `evals/evals.json`
- updated `README.md` for discoverability
- synced the skill into all runtime mirror directories with the repository-maintained sync flow

Residual risk:
- validation covered structure, sync integrity, and curated examples, but did not include a live agent benchmark loop comparing with-skill vs baseline outputs
