---
name: orchestrator
description: "Codex ledger: Use only as LEDGER_ORCHESTRATOR for a ready persisted Implementation ledger. Own routing; Skip unit work and other harnesses."
disable-model-invocation: true
---

# Ledger Orchestrator

Route one ready Implementation ledger in its saved Codex App Git project to
exhaustion or a true external boundary.

Load the [Planning Ledger
Contract](../../../docs/spec-first-workflow/phases/planning/ledger-contract.md),
[Implementation](../../../docs/spec-first-workflow/phases/implementation.md),
and the [Codex adapter](../../../docs/agent-harness/codex.md). Re-read the
ledger after every canonical transition. Assign each ready acceptance unit to a
fresh `$acceptance-unit-lead`; run independent units concurrently when current
dependencies and surfaces make that safe.

The Lead owns its internal strategy. When implementation invalidates an
upstream decision, route the smallest owning repair and resume the same unit;
do not create a semantic reopen role. Continue until the ledger is done or no
ready unit or agent-owned recovery remains. Remain routing-only: artifacts own
semantic state, Codex owns task lifecycle, and Git owns candidates.
