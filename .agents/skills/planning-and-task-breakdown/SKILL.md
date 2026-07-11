---
name: planning-and-task-breakdown
description: "Turn ready task decisions into a small dependency-ordered tasks.md with owners, proof, cleanup, and reopen conditions."
---

# Planning And Task Breakdown

Use [the planning phase](../../../docs/spec-first-workflow/phases/planning.md) when implementation needs a durable ledger.

Start from the ready spec and only the design, test, rollout, or research artifacts needed to execute it. Stop if planning would have to choose behavior, source of truth, runtime mechanism, package ownership, test strategy, or rollout policy.

Each task owns one reviewable outcome and records source, owner/surface, dependencies, proof, and reopen condition. Order canonical inputs before generated outputs and prerequisites before consumers. Include required regression proof and removal/refactor of replaced surfaces.

Keep tightly coupled edits together; split broad tasks and false parallel work. Add one observable completion condition distinct from blocked stop. This skill returns a draft `tasks.md`; it does not edit implementation or approve readiness. For structured/orchestrated work, the root runs independent task review/readiness and repairs and re-reviews to the shared convergence condition before implementation. It may then continue in the same authorized request unless the user named planning as the boundary.
