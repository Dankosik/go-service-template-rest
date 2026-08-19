---
name: orchestrator
description: "Codex ledger: Use only as LEDGER_ORCHESTRATOR. Own routing; Skip units/other harnesses."
disable-model-invocation: true
---

# Ledger Orchestrator

Route one agent-selected ready orchestrated Implementation ledger in its saved
Codex App Git project to exhaustion or a Role Tree terminal condition.

1. Bind `LEDGER_ORCHESTRATOR` and load
   the [Role Tree](../../../docs/spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree),
   the [Planning Ledger Contract](../../../docs/spec-first-workflow/phases/planning/ledger-contract.md),
   the [Agent Harness](../../../docs/agent-harness.md), and its [Codex
   adapter](../../../docs/agent-harness/codex.md). Load only the adapter branch
   required by the next native action.
2. Load the Goal branch, start the routing Goal, and re-read the ledger after
   every canonical transition. Dispatch
   `$acceptance-unit-lead` for one ready unit, or several only for a
   ledger-proven wave; dispatch `$upstream-reopen-lead` for one canonical
   agent-owned reopen.
3. Apply the native protocol through dispatch, wait, Handoff, recovery, and
   terminal cleanup.

Completion: the ledger is exhausted, or the exact Role Tree terminal condition
is recorded with no ready unit or authorized recovery. Remain routing-only:
artifacts own semantic state, Codex owns task lifecycle, and Git owns candidates.
