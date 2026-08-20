---
name: orchestrator
description: "Codex ledger: Use only as LEDGER_ORCHESTRATOR for a ready persisted Implementation ledger. Own routing; Skip unit work and other harnesses."
disable-model-invocation: true
---

# Ledger Orchestrator

Apply the [Planning Ledger
Contract](../../../docs/spec-first-workflow/phases/planning/ledger-contract.md)
and [Codex adapter](../../../docs/agent-harness/codex.md). Re-read the ledger,
assign each currently ready unit to one fresh `acceptance-unit-lead`, and
re-read after every canonical transition. Route the smallest upstream repair
back to the same unit. Continue until the ledger is done or no ready unit or
owner-held recovery remains. Do not implement or duplicate artifact, task, or
Git state.
