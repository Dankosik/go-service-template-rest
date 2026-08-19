# Conditional Planning Branches

Read only when one of these execution shapes is present.

## Integration-first slice

When integration is the primary uncertainty, make the next acceptance unit the
smallest production-grade end-to-end slice. The slice establishes one supported
behavior through the real production entry point, every uncertain integration
seam, and the final observable response, effect, or authoritative state,
together with the narrow failure or negative path required to falsify that
integration. Scaffolding, interface-only work, TODOs, mock success, and test-only
wiring do not satisfy the slice outcome; fixtures or test doubles may support
proof only behind an accepted seam. Prove that slice before expanding from it.
Otherwise keep local or already-proven work on its existing direct path.

## Expand, migrate, contract

When one mechanical contract change fans out so broadly that no bounded slice
can remain valid and green, plan `expand -> migrate -> contract`: add the
compatible new form, move bounded caller batches while both forms work, then
remove the old form after every consumer has moved. Keep the contract cleanup
in the same ledger and block it on every migration batch. Use one atomic task
when it can stay valid and provable; do not add compatibility machinery merely
to split work.
