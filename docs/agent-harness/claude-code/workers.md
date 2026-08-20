# Claude Code Workers

## Read When

Read immediately before dispatching, monitoring, correcting, or accepting an
Implementation Worker in Claude Code.

Dispatch a background `Agent` with `isolation: "worktree"`, the exact frozen
base, accepted ledger revision, and the [Worker
Contract](../../spec-first-workflow/phases/implementation-worker-contract.md#dispatch-contract).
Pass isolation at dispatch rather than frontmatter so the base follows the
parent's current `HEAD`. Collapse any dependency chain whose successor base the
harness cannot materialize before dispatch.

A Worker receives its dispatch, worktree files, repository instructions, the
parent-start status snapshot, and explicitly loaded skills. It does not receive
the root conversation, prior command output, output style, or auto-memory. Put
every execution-changing fact absent from canonical files in the dispatch.

Record the completing Worker's agent ID and keep it reachable through the unit
transition. Send one batched correction to that same ID; it retains context.
Message an active Worker only for a safety stop or accepted-input invalidation.
Replace it only for a true stall or invalidated base. `Explore` and `Plan` are
one-shot read-only lanes and never Workers.
