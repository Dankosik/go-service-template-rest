## Context
- Source-of-truth updates were made under `.agents/skills` for the repository-specific skill set.
- Mirror copies of the same 31 skills exist under `skills`, `.claude/skills`, `.cursor/skills`, `.gemini/skills`, `.github/skills`, and `.opencode/skills`.
- Content hashes show the mirrors are out of sync with `.agents/skills`.
- `docs/skills` is empty and is not a mirror of the same tree.

## Scope / Non-goals
- In scope:
  - synchronize repository-local skill mirrors from `.agents/skills` into the other mirror directories with the same structure;
  - include supporting reference files that live inside those skill directories;
  - verify the resulting trees match the source.
- Non-goals:
  - changing global skills under `/home/dankos/.codex/skills`;
  - inventing content changes beyond mirroring the updated source;
  - populating `docs/skills`, which is not currently a parallel mirror tree.

## Constraints
- `.agents/skills` is the only source of truth for this sync.
- Keep directory structure identical in all repository-local mirrors.
- Do not modify unrelated files.

## Decisions
- Sync targets are limited to `skills`, `.claude/skills`, `.cursor/skills`, `.gemini/skills`, `.github/skills`, and `.opencode/skills`.
- `docs/skills` remains untouched because it is empty and not a structural peer of the mirror set.
- Validation will use recursive diff/hash checks after copying.

## Open Questions / Assumptions
- Assumption: “sync with others” refers to the repository-local mirror directories that already contain the same 31 skill folders.
- Assumption: file removals from the source should also be reflected in targets if any are encountered during sync.

## Implementation Plan
1. Mirror `.agents/skills` into each repository-local target directory.
   Completion criteria:
   - each target contains the same files and directory structure as `.agents/skills`;
   - reference subdirectories are copied along with `SKILL.md`.
2. Validate every target against `.agents/skills`.
   Completion criteria:
   - recursive diff reports no differences;
   - aggregate content hashes for each target match the source.

## Validation
- `diff -rq .agents/skills <target>` for each mirror target
- aggregate `sha256sum` over all files in each mirror tree

## Outcome
- Synchronized `.agents/skills` into `skills`, `.claude/skills`, `.cursor/skills`, `.gemini/skills`, `.github/skills`, and `.opencode/skills`.
- Mirrored both `SKILL.md` files and nested reference files under the affected skills.
- Left `docs/skills` and `/home/dankos/.codex/skills` unchanged by design.
- Fresh validation passed:
  - `diff -rq .agents/skills skills`
  - `diff -rq .agents/skills .claude/skills`
  - `diff -rq .agents/skills .cursor/skills`
  - `diff -rq .agents/skills .gemini/skills`
  - `diff -rq .agents/skills .github/skills`
  - `diff -rq .agents/skills .opencode/skills`
  - normalized aggregate tree hash matched for all seven trees: `293d70ab8d4acb11ad965126af31ecee38ba274b3246196f74210cc04152ce71`
