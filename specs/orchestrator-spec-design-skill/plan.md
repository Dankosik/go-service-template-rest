## Phase 1: Canonical Skill Bundle

Tasks:
- Create `.agents/skills/spec-document-designer/SKILL.md`.
- Create `.agents/skills/spec-document-designer/references/spec-patterns.md`.
- Create `.agents/skills/spec-document-designer/evals/evals.json`.

Planned verification:
- Re-read the skill bundle for boundary clarity against `idea-refine`, `spec-first-brainstorming`, `go-design-spec`, and `planning-and-task-breakdown`.
- Validate `evals/evals.json` as strict JSON.

Exit criteria:
- The skill clearly owns repo-native `spec.md` design and normalization.
- The external-pattern mapping is explicit and does not create a competing artifact model.
- The reference file stays supportive rather than becoming a second `AGENTS.md`.

Review / reconciliation checkpoint:
- Check whether any instruction belongs in `AGENTS.md`/`docs/spec-first-workflow.md` instead of the skill. If yes, keep the skill lean and do not duplicate the repository contract.

## Phase 2: Discoverability And Mirror Sync

Tasks:
- Update `README.md` skill library.
- Update `docs/skills/skills-catalog.md`.
- Run `bash ./scripts/dev/sync-skills.sh`.

Planned verification:
- Confirm the new skill exists in all mirrored runtime directories.
- Run `bash ./scripts/dev/sync-skills.sh --check`.

Exit criteria:
- Canonical and mirrored skill directories are aligned.
- Both discoverability surfaces mention the new skill in the appropriate category.

Review / reconciliation checkpoint:
- Check that the catalog wording matches the final `description` in `SKILL.md`.

## Phase 3: Structural Validation

Tasks:
- Validate the eval JSON.
- Run diff hygiene checks.
- Review the final spec artifacts for consistency with the implemented skill.

Planned verification:
- `python3 -m json.tool .agents/skills/spec-document-designer/evals/evals.json >/dev/null`
- `bash ./scripts/dev/sync-skills.sh --check`
- `git diff --check`

Exit criteria:
- Skill assets are structurally valid.
- Mirrors are in sync.
- The implemented skill matches the decisions captured in `spec.md`.
