# Qwen Code Harness Adapter

Use installed Qwen task and agent controls as native authority.

## Native Map

- `todo_write`, or team `task_create` and `task_update`, tracks the root
  execution tree; repository `tasks.md` remains the acceptance ledger.
- `/orchestrator` binds the current session as Ledger Orchestrator when Agent
  Team controls are callable. Dispatch every mutually independent ready unit
  before waiting, within current capacity. Create one named Acceptance-Unit
  Lead teammate per ready unit; the Lead owns proof, review, and the
  acceptance verdict, while the team lead lands only `Accepted` candidates
  serially from the
  [Acceptance Result](../spec-first-workflow/interfaces/acceptance-result-v1.md)
  and records that verdict without re-adjudicating it.
- Bind that teammate to the canonical `acceptance-unit-lead` carrier in its
  spawn brief; do not substitute generic worker semantics.
- If Agent Team controls are unavailable, full-ledger invocation returns the
  exact carrier gap before dispatch instead of silently becoming single-unit
  work.
- Agent Team enablement is user or machine configuration. Name that condition
  when absent; do not mutate settings as part of ledger routing.
- The Lead may implement directly or delegate implement, investigate, verify,
  or review mode to a built-in or project agent.
- Use `isolation: "worktree"` when [Agent Harness](../agent-harness.md)
  selects isolation for a concurrent mutable Lead. Workers inside one Lead
  may share the Lead checkout when writable responsibility and exclusive
  locks are disjoint. Do not create worktrees for sequential work, cheap
  disjoint units, or bounded read-only review.
- Independent review uses one fresh `reviewer-agent`, names its Method, and
  keeps the fixed candidate unchanged.

## Models And Dispatch

Project agent frontmatter accepts `inherit`, `fast`, a model ID, or
`authType:modelId`. Use `fast` for closed mechanical work when configured,
`inherit` for ordinary work and a closed, strongly owned Lead unit, and the
strongest configured model when remaining uncertainty, protected-risk surface,
or high-consequence reasoning requires it. Qwen has
no per-agent reasoning-effort field, so lanes inherit session effort. Preserve
a user-selected model.

Pass the [delegation interface](../agent-harness.md#delegation-interface) through
native fields where available. Retain the returned agent identity before
waiting or following up. Continue the same agent while its context is useful;
use a fresh agent when independence, an invalidated base, a stall, or a changed
strategy makes that more reliable. A missing identity is a carrier failure,
not a completed lane.

Retain team, task, agent, and continuation identities. Use `send_message` for a
same-unit correction and a fresh `reviewer-agent` for independent review. When
Review requires integrated-candidate review, the team lead binds one fresh
`reviewer-agent` to that boundary and still does not accept units. Team
tasks mirror execution only; this session records the Lead-owned
`Accepted`/`Blocked` verdict in the repository ledger without re-adjudicating
it.
