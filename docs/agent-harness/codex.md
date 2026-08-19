# Codex Harness Adapter

## Read When

Read after the [Agent Harness](../agent-harness.md) selects Codex and a native
task, Worktree, subagent, Goal, model, effort, Handoff, or recovery control is
needed. Callable installed schemas outrank public prose.

## Native Map

- A Goal is thread-local and evidence-aware.
- An isolated `IMPLEMENTATION_WORKER` is a separate top-level App task created
  in Worktree mode. A built-in `collaboration.spawn_agent` child is read-only
  for this workflow, regardless of its prompt or tools.
- Read-only specialists and reviewers use fresh project subagents with no
  inherited turns unless irreproducible user context requires the smallest
  bounded recent set.
- A Ledger Orchestrator is `$orchestrator` in a dedicated saved-project task.

The Worktree task's `threadId`, `hostId`, backing, and checkout must satisfy the
shared [Write-Carrier Gate](../agent-harness.md#write-carrier-gate).

## Conditional Controls

| Trigger | Read before |
| --- | --- |
| Create, wait on, correct, Handoff, or clean up a known task. | [Codex Orchestration](codex/codex-orchestration.md) |
| Reconcile unknown create/Handoff state, terminalize a known Lead, or resume an upstream reopen. | [Codex Recovery](codex/codex-recovery.md) |
| Start, inspect, resume, or clear a Goal. | [Codex Goals](codex/codex-goals.md) |

## Model And Effort

Use installed model names and supported fields. Preserve a user-named model.
Otherwise:

| Lane | Model |
| --- | --- |
| Closed mechanical Worker | Luna with `low` effort |
| Ordinary or complex Worker or review | Terra at `medium`, raised to `high` or `xhigh` for closed-route complex, cross-cutting, protected-domain, or high-consequence work |
| Acceptance-Unit Lead or hardest critical review | Sol with `xhigh`, falling back to Sol `high` only on a recorded unsupported or rejected `xhigh` override |

Apply the shared effort policy through the installed supported fields. When the
installed schema exposes `model` or `thinking`, omitting it is an invalid
dispatch: inherited settings and a model or effort named only in prompt text do
not satisfy selection. Treat a field as unavailable only when the installed
schema lacks it, and as rejected only from the control's actual rejection;
then continue on the effective configured value and retain that capability
evidence. [Codex Orchestration](codex/codex-orchestration.md) owns the dispatch
configuration receipt. Only after that receipt do corrections and continuations
omit overrides so the established configuration stays intact.
