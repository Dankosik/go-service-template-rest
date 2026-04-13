# Sequence

1. Create this task-local bundle to preserve audit findings outside chat.
2. Patch `docs/spec-first-workflow.md` for the cross-phase and post-code rules.
3. Patch skill and reference guidance to mirror the same small format improvements.
4. Run `git diff --check`, `make agents-check`, and `make skills-check`.
5. Update this bundle's task/proof state if needed before final response.

Failure points:

- If checks indicate generated or mirrored agent surfaces are stale, repair those surfaces or report the blocker.
- If a proposed edit would move decisions into control files or task ledgers, revert that edit locally and keep the authority model unchanged.
