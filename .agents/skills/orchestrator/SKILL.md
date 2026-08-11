---
name: orchestrator
description: "Ledger: Use when ready; Own Lead routing; Skip unit work."
---

# Ledger Orchestrator

Use in the Codex App only when the user requests execution of a ready
Implementation ledger through fresh tasks in the saved Git project. Bind
`LEDGER_ORCHESTRATOR`, start its routing Goal, and load the canonical
[Execution Role Tree](../../../docs/spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree),
[Artifact Model](../../../docs/spec-first-workflow/shared/artifact-model.md),
[Resume And Handoff](../../../docs/spec-first-workflow/shared/resume-and-handoff.md),
and [Agent Harness's native orchestration
protocol](../../../docs/agent-harness.md#codex-app-native-orchestration-protocol).

For each ready unit, select its Lead's model and effort, dispatch one fresh Lead
through the native bootstrap, wait for its event, and verify the canonical
transition and Git identity before routing again. Dispatch several Leads only
for a ledger-proven planned wave.

Remain routing-only. The Role Tree owns terminal conditions, bottom-up obstacle
handling, and Lead authority. Artifacts own readiness and receipts, Codex owns
task lifecycle, and Git owns candidates. Resolve native routing obstacles,
return a misrouted `NEEDS_PARENT` to its Lead, keep independent ready units
moving, and resume a blocked unit only after its reopen condition changes.
