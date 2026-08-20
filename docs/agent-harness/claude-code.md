# Claude Code Harness Adapter

## Read When

Read after the [Agent Harness](../agent-harness.md) selects Claude Code and a
native Agent lane, Goal, model, effort, peer message, or programmatic run is
needed.

## Native Map

- `/goal <condition>` is the durable execution control; its evaluator sees only
  the conversation.
- An isolated Implementation Worker is a background `Agent` lane with
  `isolation: "worktree"` and an effort-carrier `subagent_type`.
- Read-only lanes use fresh background `Agent` contexts without worktree
  isolation.
- Claude Code has no Ledger Orchestrator carrier; the current root binds one
  Acceptance-Unit Lead and stops at that unit's canonical transition.

## Conditional Controls

| Trigger | Read before |
| --- | --- |
| Start, resume, or clear a Goal. | [Goals](claude-code/goals.md) |
| Dispatch, monitor, or correct an Implementation Worker. | [Workers](claude-code/workers.md) |
| Dispatch or wait on research, challenge, or review. | [Read-Only Lanes](claude-code/read-only-lanes.md) |
| Discover or message another session. | [Cross-Session Messaging](claude-code/cross-session.md) |
| Drive this harness programmatically. | [Claude Agent SDK](claude-code/sdk.md) |

## Model And Effort

Apply the shared [selection policy](shared/model-selection.md).
Map it to Sonnet for mechanical and ordinary lanes and Opus for
Acceptance-Unit Leads and complex or high-consequence lanes. Carry effort
through the installed role carrier; remove that carrier when Claude gains a
per-dispatch effort field.
