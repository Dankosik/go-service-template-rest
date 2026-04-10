## Phase 1: Canonical Skill Bundle

Tasks:
- Create `.agents/skills/ru-agent-prompt-composer/SKILL.md`.
- Create compact reference files:
  - `references/repo-profile.md`
  - `references/context-selection.md`
  - `references/example-transformations.md`
- Create validation assets:
  - `evals/evals.json`
  - `evals/files/*.md` with realistic messy Russian fixtures

Planned verification:
- Read the skill and reference files end-to-end for consistency with the decided output contract.
- Validate `evals/evals.json` as strict JSON.

Exit criteria:
- The skill clearly distinguishes grounded signals from assumptions.
- The live-lookup trigger policy is explicit and bounded.
- Examples and evals cover at least three realistic messy-input cases without turning into generic translation.

Review / reconciliation checkpoint:
- Reconcile the skill text against repo-native authoring conventions before editing discoverability or mirrors.

## Phase 2: Discoverability And Mirror Sync

Tasks:
- Update `README.md` skill library with the new skill.
- Update `docs/skills/skills-catalog.md`.
- Run `bash ./scripts/dev/sync-skills.sh` to create/update runtime mirrors.

Planned verification:
- Confirm the new mirrored skill exists in all runtime skill directories.
- Run `bash ./scripts/dev/sync-skills.sh --check`.

Exit criteria:
- Canonical and mirrored skill directories are aligned.
- Both human-facing discoverability surfaces mention the new skill.

Review / reconciliation checkpoint:
- Check for accidental bloat in mirrored content and remove anything that should not be replicated.

## Phase 3: Targeted Validation

Tasks:
- Review the curated example prompts as downstream-agent handoff artifacts.
- Verify the raw fixtures and eval expectations still match the final skill behavior.
- Run repository-safe validation commands for formatting/sync integrity.

Planned verification:
- `python -m json.tool .agents/skills/ru-agent-prompt-composer/evals/evals.json >/dev/null`
- `bash ./scripts/dev/sync-skills.sh --check`
- `git diff --check`

Exit criteria:
- Validation assets are structurally valid.
- Mirror sync is clean.
- The final examples are specific, repo-aware, and better than literal translation.
