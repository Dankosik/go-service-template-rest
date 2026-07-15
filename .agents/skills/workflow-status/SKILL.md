---
name: workflow-status
description: "Read-only status and next-action summary for one identified task path."
---

# Workflow Status

Use [the workflow router](../../../docs/spec-first-workflow.md) and [artifact-model review/freshness](../../../docs/spec-first-workflow/shared/artifact-model.md#review-and-freshness) plus [resume order](../../../docs/spec-first-workflow/shared/artifact-model.md#resume-order). Inspect current workspace and Git drift before reporting ready, done, or implementation readiness. Return status, owner, evidence, next action, and readiness without changing state.
