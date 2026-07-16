---
name: workflow-planning-session
description: "Coordination: Use when a task genuinely needs a compact workflow-plan.md across sessions or independent lanes. Own only the routing, handoffs, and reopen conditions needed for that coordination; Skip when one lane is sufficient or process would exist only for documentation."
---

# Workflow Planning Session

Must read [the Artifact Model](../../../docs/spec-first-workflow/shared/artifact-model.md); it owns the persistence trigger, workflow-plan shape, status, and resume order. Build a control plane only when work spans sessions or independent lanes: reconstruct only the coordination state needed from canonical artifacts, current status, lane dependencies, and reopen conditions, never a second source of task decisions. Give every transition an actionable chain of custody—owner, consumed artifact/evidence, next action, produced checkpoint, next owner, and reopen condition—and disposition missing inputs as open decision, external input, or blocker. Stop when every transition is actionable or blocked by its named owner without chat archaeology.
