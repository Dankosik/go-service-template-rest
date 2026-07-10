---
name: planning-session
description: "Create or repair a compact executable tasks.md and review readiness when the implementation needs a durable ledger."
---

# Planning Session

Use [planning](../../../docs/spec-first-workflow/phases/planning.md) when work has multiple dependent steps, actors, generated-source order, or proof checkpoints.

Write outcome-oriented tasks with source, owner/surface, dependencies, proof, and reopen condition. Add fields only when they change execution. Include one observable successful completion condition and required cleanup.

For structured or orchestrated work, invoke [task review/readiness](../../../docs/spec-first-workflow/phases/task-review-readiness.md) before implementation. Direct work uses a focused self-check unless the user or risk requires independent review. Repair ledger defects; reopen earlier owners for missing behavior/design/test decisions.

Task review, planning-owned repair, and fresh re-review stay in the same root session and never produce a next-session prompt. Stop before implementation only when the user named planning as the boundary; otherwise continue directly into implementation in the same authorized request.
