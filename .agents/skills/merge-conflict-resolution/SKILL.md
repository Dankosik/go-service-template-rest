---
name: merge-conflict-resolution
description: "Active merge conflicts. Use when an in-progress merge, rebase, cherry-pick, or revert has conflicted hunks that require intent reconstruction, resolution, proof, and continuation."
metadata:
  invocation: model
  kind: method
---

# Merge Conflict Resolution

Inspect `git status`, the active operation, its base and sides, every conflicted
path, and unrelated work before editing. For each hunk, trace both sides to
their closest authority: accepted spec or task, canonical generated source,
commit and PR, current callers, and tests. State both intents before resolving.

Under read-only authorization, stop after reconstructing both intents and
return the hunk dispositions, proof plan, and exact blocker without editing,
staging, or continuing the operation.

Preserve compatible intents. When they conflict, follow the accepted outcome
and current source of truth; never invent behavior merely to remove markers or
select one whole side without hunk-level evidence. If authority cannot choose,
leave the operation intact and return the exact decision blocker.

Inspect the resolved diff from the operation base, run the smallest affected
proof, stage only resolved paths, and continue the already-authorized operation.
Return the operation and base, intent sources, hunk dispositions, proof, and
any unresolved blocker.
