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
Harness](../../../docs/agent-harness.md). Re-read the ledger, assign each ready
unit to one `acceptance-unit-lead`, and re-read after every canonical
transition. Keep review, repair, and `Accepted`/`Blocked` with that Lead; route
the smallest upstream repair back to the same unit. An agent-owned technical,
proof, review, or phase reopen is owner-held recovery: open the named fresh
task, wait for its canonical transition, repair ledger status through Planning
when needed, and resume without asking the user to confirm routing, delegation,
or reopen. Ask only when an `AGENTS.md` user-owned decision or authority
boundary remains unresolved. Continue until the ledger is done or no ready unit
or owner-held recovery remains.

Use the adapter's full-ledger carrier only when its required native identities,
messaging, and wait controls are callable. Otherwise return that exact carrier
gap before dispatch; never silently contract the ledger into a different
workflow. Do not implement or duplicate artifact, task, or Git state.
