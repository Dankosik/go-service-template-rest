## Context

The repository previously treated top-level `skills/` as the canonical source for skill mirrors. That policy has already been replaced: `.agents/skills` is now canonical and `skills-check` is no longer a blocking CI gate.

The user then decided to remove `skills/` completely rather than keep it as a compatibility mirror.

## Scope / Non-goals

In scope:
- switch skills sync/check tooling to use `.agents/skills` as canonical
- remove the top-level `skills/` tree from the repository
- keep only runtime mirrors outside `.agents/skills`
- remove blocking `skills-check` from CI and CI-like aggregate flows
- update repository docs and specs that still point at `skills/...`

Non-goals:
- rewriting unrelated historical docs that do not reference deleted skill paths
- removing manual `make skills-sync` / `make skills-check` support

## Constraints

- Execution shape: `lightweight local`
- Research mode: `local`
- Pre-spec challenge: `waived`
  Rationale: the user already chose the target policy, and the main risk is repository path coverage rather than unresolved design ambiguity.
- `.agents/skills` must become the only canonical authoring surface after this change.
- No active repository flow should rely on a top-level `skills/` directory after this change.

## Decisions

1. `.agents/skills` becomes the canonical source for repository skill content.
2. The top-level `skills/` directory is removed rather than retained as a compatibility mirror.
3. `scripts/dev/sync-skills.sh` will sync from `.agents/skills` into runtime mirrors only:
   - `.claude/skills`
   - `.gemini/skills`
   - `.github/skills`
   - `.cursor/skills`
   - `.opencode/skills`
4. Blocking `skills-check` remains removed from GitHub Actions workflows and CI-like aggregate commands, while `make skills-sync` / `make skills-check` stay available for manual maintenance.
5. Template/bootstrap flows may still run `make skills-sync`, because keeping mirrors refreshed during setup is useful and does not reintroduce CI coupling.
6. Active docs and canonical eval metadata should reference `.agents/skills/...` instead of `skills/...`.

## Open Questions / Assumptions

- Assumption: updating active docs/specs and canonical `.agents` eval metadata is sufficient; target-only mirror leftovers in non-canonical runtime directories do not block the deletion.
- Assumption: deleting `skills/` is acceptable even though old historical validation transcripts may continue to mention the removed path as part of past state.

## Implementation Plan

1. Update sync tooling to stop mirroring into `skills/`.
   Completion criteria:
   - `sync-skills.sh` uses `.agents/skills` as source
   - `skills/` is no longer a sync target
   - help text reflects the new model

2. Delete the top-level `skills/` directory from the repository.
   Completion criteria:
   - tracked and untracked content under `skills/` is removed
   - no active tooling path depends on `skills/`

3. Keep `skills-check` out of blocking CI-like flows.
   Completion criteria:
   - GitHub workflow files no longer invoke `make skills-check`
   - native/docker CI aggregate commands no longer include `skills-check`

4. Update docs/specs that still reference `skills/...`.
   Completion criteria:
   - README and core workflow docs no longer describe `skills/` as existing
   - active docs and canonical eval metadata point to `.agents/skills/...`

5. Refresh runtime mirrors and run targeted validation.
   Completion criteria:
   - mirrors are synced from `.agents/skills`
   - sync check passes under the new model
   - targeted searches show no active tooling/docs still depending on top-level `skills/`

## Validation

Executed:
- `test -d skills && echo present || echo absent`
- `bash ./scripts/dev/sync-skills.sh`
- `bash ./scripts/dev/sync-skills.sh --check`
- `rg -n "skills/" .agents/skills docs specs README.md scripts Makefile .github/workflows`
- `git diff --check -- scripts/dev/sync-skills.sh scripts/dev/docker-tooling.sh Makefile .github/workflows README.md docs specs/agents-skills-source-of-truth/spec.md`

## Outcome

Completed:
- `.agents/skills` remains the canonical source used by `scripts/dev/sync-skills.sh`
- top-level `skills/` was removed instead of kept as a compatibility mirror
- blocking `skills-check` stays removed from GitHub Actions workflows and CI-like aggregate commands
- active docs/specs and canonical eval metadata no longer point at `skills/...`
- runtime mirrors are now refreshed only from `.agents/skills` into provider-specific directories
