## Context

The repository still contains a documentation-only layer that preserves the previous workflow vocabulary and the old numbered artifact pack. Those files are not the canonical runtime contract anymore, but they still read like active instructions and create competing guidance.

The requested change is to remove that legacy workflow language from the repository so the active instruction surfaces read as one coherent system with no transition-era caveats.

## Scope / Non-goals

In scope:
- remove obsolete workflow language from active contract docs
- remove documentation-only design notes that still teach the old phase/gate and numbered-artifact model
- align remaining discoverability docs with the current `spec.md` / `workflow-plan.md` / `plan.md` / `test-plan.md` artifact model
- scrub remaining references from runnable skill docs and nearby spec records where they would otherwise preserve the old model

Non-goals:
- changing runtime agent behavior beyond wording and artifact references
- changing repository code, CI, or tooling unrelated to documentation cleanup
- rewriting every historical decision record in `specs/`; only directly affected references are cleaned up

## Constraints

- The canonical runtime contract must stay in `AGENTS.md`, `docs/spec-first-workflow.md`, and `.agents/skills/*/SKILL.md`.
- The resulting documentation should not rely on "legacy" disclaimers; it should simply describe the current workflow.
- Keep the artifact model consistent across all touched files: `spec.md` is canonical, `workflow-plan.md` captures orchestration when needed, `plan.md` is the coder-facing execution plan for non-trivial work, and `test-plan.md` stays optional.

## Decisions

1. Remove legacy-transition wording from the active contract surfaces.
   - Delete the `legacy` compatibility language from `AGENTS.md`.
   - Delete the `Legacy Compatibility` section from `docs/spec-first-workflow.md`.

2. Remove the obsolete documentation-only design-note layer instead of preserving it behind caveats.
   - Delete the obsolete workflow rewrite notes.
   - Delete outdated skill-spec and adaptation notes that still teach the old workflow model.

3. Keep only current documentation surfaces.
   - Preserve the active root workflow docs and repository overview.
   - Remove docs-only skill indexes and guides that no longer exist.

4. Align runnable and discoverability surfaces to the current artifact model.
   - Update `.agents/skills/api-contract-designer-spec/SKILL.md` to point only at `spec.md`.
   - Update its eval expectations accordingly.
   - Update remaining discoverability wording so it no longer points at removed documentation files.

5. Clean the nearest repository specs that would otherwise preserve removed-file references or old workflow wording.
   - Update the affected spec records in `specs/pre-spec-challenge-workflow/` and `specs/workflow-rigor-alignment/`.

## Open Questions / Assumptions

- Assumption: deleting the documentation-only legacy files is preferable to keeping short redirect stubs because the user explicitly asked to remove the old system as though it never existed.
- Assumption: `README.md` is enough as the human-facing discoverability surface after removing the docs-only skill index layer.

## Plan Summary / Link

Execution follows [`plan.md`](plan.md).

Control summary:
1. Update the active contract surfaces first.
2. Remove the obsolete documentation-only layers.
3. Align remaining runnable skills and nearby spec records.
4. Re-scan for the removed markers and record the validation evidence.

## Validation

Executed:
- `git diff --name-only --diff-filter=D -- docs`
- targeted `rg` scan for the retired workflow vocabulary across `AGENTS.md`, `README.md`, `docs/`, `.agents/`, `specs/`, and `.codex/`
- targeted `rg` scan for references to the deleted documentation paths
- `git diff --check`

## Outcome

Completed:
- removed the transition-era compatibility wording from `AGENTS.md` and `docs/spec-first-workflow.md`
- deleted the obsolete documentation-only workflow/design-note layer and the old skill-doc layer
- aligned remaining discoverability wording with the current artifact model and removed references to files that no longer exist
- updated `.agents/skills/api-contract-designer-spec/SKILL.md` and its evals so the runnable skill points only at `spec.md`
- cleaned the directly affected spec records so they no longer point at removed files or preserve the retired workflow wording

Residual risk:
- repository history in `git` still preserves the removed material, but the active instruction surfaces in the working tree no longer expose it
