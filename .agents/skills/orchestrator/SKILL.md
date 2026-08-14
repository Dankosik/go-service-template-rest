---
name: orchestrator
description: "Ledger: Use when ready; Own routing/reopens; Skip work."
---

# Ledger Orchestrator

Use in the Codex App only when the user launches a ready Implementation ledger
in its saved Git project. Bind `LEDGER_ORCHESTRATOR`, start its routing Goal, and
load the [Role Tree](../../../docs/spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree),
[Artifact Model](../../../docs/spec-first-workflow/shared/artifact-model.md),
[Resume And Handoff](../../../docs/spec-first-workflow/shared/resume-and-handoff.md),
and [native protocol](../../../docs/agent-harness.md#codex-app-native-orchestration-protocol).

The one user invocation authorizes every reversible native routing decision
through ledger exhaustion. For each fresh task, choose its model and effort from
the fixed brief and installed controls. Create it once with a no-op role/scope
bootstrap and no overrides, wait for exactly `READY_FOR_DISPATCH`, then send one
technical handoff with the selected pair. If an override is unavailable,
continue on the effective configured value and record the capability gap; never
ask the user. Dispatch several Leads only for a ledger-proven wave.

Before reopening a Worktree unit, Handoff the same Lead and fixed candidate to
Local and let it persist the canonical blocker there. Then dispatch one fresh
Local `UPSTREAM_REOPEN_LEAD` per affected macro phase on an agent-selected pair,
wait through review and repair, and route only invalidated phases or prerequisite
units. Resume the original Local Goal when supported; after proven non-resume,
dispatch one replacement Local Lead for the same unit and candidate.

After every child terminal event, apply [Terminal task
cleanup](../../../docs/agent-harness.md#terminal-task-cleanup) before routing
again.

Remain routing-only. Artifacts own semantic state, Codex owns task lifecycle,
and Git owns candidates.
