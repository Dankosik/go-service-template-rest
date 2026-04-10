## Phase 1: Active Contract Cleanup

Tasks:
- Remove legacy-transition wording from `AGENTS.md`.
- Remove the legacy-compatibility section from `docs/spec-first-workflow.md`.
- Add or tighten any nearby wording needed so the current workflow reads as self-contained without legacy disclaimers.

Planned verification:
- Re-read both files after the edit.
- Run targeted searches for the removed workflow markers and outdated numbered artifact references in the active contract files.

Exit criteria:
- The active workflow contract no longer talks about legacy instructions or conflict resolution with old docs.
- The current artifact model reads cleanly on its own.

Review / reconciliation checkpoint:
- Confirm the cleanup did not remove any still-needed current workflow guidance.

## Phase 2: Documentation-Layer Pruning

Tasks:
- Delete the obsolete `docs/agent-centric-rewrite/` files.
- Delete outdated per-skill design notes and adaptation/problem-statement docs under `docs/skills/`.
- Keep only the current documentation-only guides and catalog in `docs/skills/`.

Planned verification:
- List the remaining files under `docs/skills/`.
- Confirm no surviving docs point at removed files.

Exit criteria:
- The documentation-only layer no longer contains the old workflow model.
- No deleted-file references remain in the touched surfaces.

Review / reconciliation checkpoint:
- Confirm the remaining `docs/skills/` files are enough for discoverability and authoring guidance.

## Phase 3: Alignment And Validation

Tasks:
- Update `docs/skills/skills-catalog.md`, `docs/skills/portable-agent-skills.md`, and `docs/skills/skill-writing-guide.md`.
- Update `.agents/skills/api-contract-designer-spec/SKILL.md` and its eval file.
- Clean the directly affected spec records under `specs/`.
- Run the final validation scans and record the results in `spec.md`.

Planned verification:
- `rg` scans for removed old-workflow markers in active docs and runnable skills.
- `git diff --check`

Exit criteria:
- Remaining active docs consistently describe only the current workflow and artifact model.
- Validation confirms the removed markers are gone from the intended surfaces.
