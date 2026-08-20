# Qwen Code Harness Adapter

Use installed Qwen task and agent controls as native authority.

## Native Map

- `todo_write`, or team `task_create` and `task_update`, tracks the root
  execution tree; repository `tasks.md` remains the acceptance ledger.
- Qwen has no Ledger Orchestrator carrier; the current root owns one
  Acceptance-Unit Lead.
- The Lead may implement directly or delegate implement, investigate, verify,
  or review mode to a built-in or project agent.
- Use `isolation: "worktree"` only when separate writable state prevents a real
  collision or preserves a useful candidate.
- Independent review uses one fresh `task-acceptance-agent` and keeps the fixed
  candidate unchanged.

## Models And Dispatch

Project agent frontmatter accepts `inherit`, `fast`, a model ID, or
`authType:modelId`. Use `fast` for closed mechanical work when configured,
`inherit` for ordinary work, and the strongest configured model for the Lead or
complex, weak-oracle, protected-domain, or high-consequence reasoning. Qwen has
no per-agent reasoning-effort field, so lanes inherit session effort. Preserve
a user-selected model.

Pass the [delegation interface](../agent-harness.md#delegation-interface) through
native fields where available. Retain the returned agent identity before
waiting or following up. Continue the same agent while its context is useful;
use a fresh agent when independence, an invalidated base, a stall, or a changed
strategy makes that more reliable. A missing identity is a carrier failure,
not a completed lane.
