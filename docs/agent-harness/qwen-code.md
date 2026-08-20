# Qwen Code Harness Adapter

Use installed Qwen task and agent controls as native authority.

## Native Map

- `todo_write`, or team `task_create` and `task_update`, tracks the root
  execution tree; repository `tasks.md` remains the acceptance ledger.
- `/orchestrator` binds the current session as Ledger Orchestrator when Agent
  Team controls are callable. Create one named Acceptance-Unit Lead teammate
  per ready unit; the Lead owns proof, review, and the canonical transition,
  while the team lead only routes ledger-ready work.
- Bind that teammate to the canonical `acceptance-unit-lead` carrier in its
  spawn brief; do not substitute generic worker semantics.
- If Agent Team controls are unavailable, full-ledger invocation returns the
  exact carrier gap before dispatch instead of silently becoming single-unit
  work.
- Agent Team enablement is user or machine configuration. Name that condition
  when absent; do not mutate settings as part of ledger routing.
- The Lead may implement directly or delegate implement, investigate, verify,
  or review mode to a built-in or project agent.
- Use `isolation: "worktree"` only when separate writable state prevents a real
  collision or preserves a useful candidate.
- Independent review uses one fresh `reviewer-agent`, names its Method, and
  keeps the fixed candidate unchanged.

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

Retain team, task, agent, and continuation identities. Use `send_message` for a
same-unit correction and a fresh `reviewer-agent` for independent review. Team
tasks mirror execution only; only the Acceptance-Unit Lead writes
`Accepted`/`Blocked` in the repository ledger.
