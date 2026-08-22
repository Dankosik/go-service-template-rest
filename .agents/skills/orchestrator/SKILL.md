---
name: orchestrator
description: "Ledger routing: Use only as LEDGER_ORCHESTRATOR for a ready persisted Implementation ledger. Own routing; Skip unit work."
metadata:
  invocation: role
  kind: carrier
disable-model-invocation: true
---

# Ledger Orchestrator

Apply the [Planning Ledger
Contract](../../../docs/spec-first-workflow/phases/planning/ledger-contract.md)
and the current adapter selected by [Agent
Harness](../../../docs/agent-harness.md). Treat `tasks.md` as a dependency
graph, not an ordered list.

Only this carrier writes canonical ledger state during orchestrated execution.
At each cycle, compute the ready frontier from packet mutable owners, exclusive
locks, and accepted dependencies, then dispatch every ready unit to one fresh
`acceptance-unit-lead` before waiting, within current capacity counted across
live Leads and their in-flight children.

Each Lead returns one immutable [Acceptance Result
V1](../../../docs/spec-first-workflow/interfaces/acceptance-result-v1.md). Land
the candidate serially, record the Lead-owned verdict without re-adjudicating
it, then immediately refill the frontier before landing the next waiting
result.

Route the smallest upstream repair back to the same unit. Do not cancel
unrelated running units when one unit reopens or discovers a lock. An agent-owned
technical, proof, review, or phase reopen is owner-held recovery: open the named
fresh task, wait for its canonical transition, repair ledger status through
Planning when needed, and resume without asking the user to confirm routing,
delegation, or reopen. Ask only when an `AGENTS.md` user-owned decision or
authority boundary remains unresolved. Continue until the ledger is done or no
ready unit or owner-held recovery remains.

Use the adapter's full-ledger carrier only when its required native identities,
messaging, and wait controls are callable. Otherwise return that exact carrier
gap before dispatch; never silently contract the ledger into a different
workflow. Do not implement unit work or keep a parallel artifact, task, or Git
journal.
