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
At each cycle, apply the contract's ready-frontier and capacity rules; dispatch
every ready unit to an `acceptance-unit-lead` before waiting, using only the
contract's permitted Lead reuse.

Process each Lead's immutable [Acceptance Result
V1](../../../docs/spec-first-workflow/interfaces/acceptance-result-v1.md) through
the contract's acceptance transition. Integrate `Implemented` candidates
serially into the local development tree and immediately refill the frontier;
do not wait for tests or reviews. Route `Blocked` to recovery. After assembly,
assign one Lead the final delivery boundary, consume its final proof and
acceptance result, and record it without repeating validation.

Route the smallest upstream repair back to the same unit. Do not cancel
unrelated running units when one unit reopens or discovers a lock. Apply
[Parent-Owned Recovery](../../../docs/spec-first-workflow/shared/transition.md#parent-owned-recovery)
to technical, proof, review, and phase gaps; repair ledger status through
Planning when needed. Continue until the ledger is done and
[Cleanup](../../../docs/spec-first-workflow/shared/cleanup.md) has closed
execution-only state, or no ready unit, owner-held recovery, or authorized
cleanup path remains.

Use the adapter's full-ledger carrier only when its required native identities,
messaging, and wait controls are callable. Otherwise return that exact carrier
gap before dispatch; never silently contract the ledger into a different
workflow. Do not implement unit work or keep a parallel artifact, task, or Git
journal.
