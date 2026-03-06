# Context

CI `docs-drift-check` fails because the canonical Docker build files changed without a corresponding documentation update.

# Scope / Non-goals

In scope:
- document the repository rule that Go toolchain bumps must update both Docker Go base-image pins together with `go.mod`;
- satisfy `docs-drift-check` for the current Dockerfile changes.

Non-goals:
- additional CI, Docker, or runtime behavior changes;
- reintroducing deleted spec artifacts into tests.

# Constraints

- Keep the documentation change minimal and aligned with existing CI policy documentation.
- Do not change docs-drift policy itself.

# Decisions

- Update `docs/build-test-and-development-commands.md` because it already documents docker-tooling image sourcing and docs-drift policy.
- Document the invariant in version-agnostic form so future Go patch bumps do not require repeated doc wording changes.

# Open Questions / Assumptions

- Assumption: one meaningful docs update is sufficient because `docs-drift-check` only requires docs presence when behavior/CI-sensitive files change.

# Implementation Plan

1. Add a note that `go.mod`, `build/docker/Dockerfile`, and `build/docker/tooling-images.Dockerfile` must stay on the same Go patch line.
2. Clarify the docs-drift exception wording so it remains explicit that only isolated tooling-image updates are exempt.
3. Re-run the docs-drift logic against the current worktree and report the outcome honestly.

# Validation

- `bash ./scripts/ci/docs-drift-check.sh "4ad60291ac6d57727a68e6771f5ecae15a267b23" "<ephemeral-worktree-commit>"`
  - result: pass (`docs drift check passed`)
- `git diff --name-only 4ad60291ac6d57727a68e6771f5ecae15a267b23 -- .`
  - result: `docs/build-test-and-development-commands.md` is now part of the change set alongside the Dockerfile updates.

# Outcome

- Docs now explicitly state that Docker Go base-image pins must stay aligned with the Go patch line in `go.mod`.
- Current Dockerfile changes no longer violate the repository docs-drift policy.
