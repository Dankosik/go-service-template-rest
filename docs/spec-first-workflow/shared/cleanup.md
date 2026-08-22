# Cleanup

Use when a task bundle has reached proven closeout.

Remove execution-only `tasks.md`, `tasks/`, `workflow-plan.md`, and worktrees
created for the bundle.

A bundle worktree is removed only when it is clean. Uncommitted or untracked
paths stop cleanup: name them and leave the tree. Closeout does not invent a
clean tree.

Retain a completed spec, design, research note, or rollout record only while a
live authority names it as a durable decision source. Otherwise move the
durable rule, deciding commit or pull request when provenance matters, and
objective reopen condition to its canonical owner, then delete the completed
bundle.

Git is the archive. Do not keep completed ledgers as a second history.
