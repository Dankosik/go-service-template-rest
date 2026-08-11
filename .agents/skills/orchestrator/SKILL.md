---
name: orchestrator
description: "Orchestrate a ready Codex ledger through fresh App tasks."
---

# Orchestrator

Use only in the Codex App after explicit user authorization to create tasks.
Load [Artifact Model](../../../docs/spec-first-workflow/shared/artifact-model.md),
[Resume And Handoff](../../../docs/spec-first-workflow/shared/resume-and-handoff.md),
and [Agent Harness](../../../docs/agent-harness.md). Require a ready next owner
and external-effect envelope; otherwise return the blocker.

If needed, create a dedicated App Worktree task from the accepted integration
base, using working-tree state when that base is not the project default; pass
this skill and the user's envelope, then stop here. In the coordinator, start
one orchestration-only Goal and keep one writer. Fork one fresh same-directory
App task per macro phase or acceptance unit. Pass the `Next
Session Prompt` at a phase boundary; during Implementation pass the accepted
ledger path, exact unit IDs, and only live facts absent from it. The receiver
owns its workflow and stops before the next owner. It starts a Goal only
during Implementation and persists its transition and receipt.

Require `UNIT_COMPLETE|UNIT_BLOCKED`, owner ID, transition or movement evidence,
candidate ref when applicable, and blocker. Wait on App task events. Route
onward only when the returned identities agree with durable artifacts. Return
an incomplete envelope to the same task; resume that task once after a transport
interruption. A block, user attention, changed scope, or missing authority stops
the chain and preserves the blocker.

The coordinator reads lifecycle fields and named receipts only. It does not
inspect diffs, review, rerun proof, repair, replan, or replace a semantic
failure with a fresh owner. Reuse the ledger's `Active wave` only when durable
task mapping is needed; create no scheduler artifact.
