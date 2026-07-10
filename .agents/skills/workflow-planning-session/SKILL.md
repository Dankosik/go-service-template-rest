---
name: workflow-planning-session
description: "Create or repair a compact workflow-plan.md only when a task genuinely needs cross-session or multi-lane coordination."
---

# Workflow Planning Session

Use after intake when the task is unfinished and cannot be resumed safely from `spec.md` or `tasks.md` alone.

Read `AGENTS.md`, [the workflow router](../../../docs/spec-first-workflow.md), and [artifact model](../../../docs/spec-first-workflow/shared/artifact-model.md). Write only the smallest `workflow-plan.md` that records goal, current phase, active artifacts, blockers/assumptions, next action, and completion proof.

Do not duplicate specification, design, test, or task content. Do not create phase-control files by default. A separate read-only adequacy challenge is optional when routing is high-impact, contested, or explicitly requested.

Success means a context-blind next session can resume from one clear artifact. Stop blocked when the goal or next action depends on a missing user/external decision.
