---
name: planning-session
description: "Own the user-started planning macro phase: produce or repair tasks.md, run independent task-review/readiness, repair findings, and obtain a fresh implementation-readiness verdict without coding."
---

# Planning Session

## Eligibility And Outcome

Use when routing enters planning with a specification-review-approved `spec.md`, all triggered design and test-design inputs current, and the task needs an executable ledger. Skip direct work, missing upstream approvals, test-design authoring, and any request to code. It owns task-review/readiness internally.

The outcome is an approved or blocked `tasks.md` whose dependency order, reviewable diff stories, source traceability, proof, risk checkpoints, cleanup, Goal-ready completion condition, and current readiness verdict support the next macro phase.

## Canonical Owners

- [Planning](../../../docs/spec-first-workflow/phases/planning.md) owns entry readiness, ledger shape, task authoring, traceability, checkpoints, implementation handoff fields, and the planning stop rule.
- [Task Review / Readiness](../../../docs/spec-first-workflow/phases/task-review-readiness.md) owns ledger approval and implementation authorization.
- [Artifact model](../../../docs/spec-first-workflow/shared/artifact-model.md) owns typed state, routing identity, artifact expectations, and phase-control eligibility.
- [Subagents and handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) owns lane gates, authorization wording, resume order, and final prompt rendering.
- `planning-and-task-breakdown` supplies the task-decomposition method.

Load only the bundled reference that can change the result: entry readiness, allowed writes, implementation-readiness gate, phase-control examples, workflow-plan projection, or session boundary. Do not load the full reference set by default.

## Allowed Side Effects

This session may create or repair task-local `tasks.md`, record internal task-review/readiness cycles, update existing `workflow-plan.md`, and create or update planning/readiness phase-control files only when ROUTING-PHASE-CONTROL allows them.

It must not create or repair `spec.md`, design, `test-plan.md`, runtime code, tests, migrations, generated output, implementation patches, or closeout evidence.

## Unique Method

1. Confirm the mandatory specification review is current PASS or eligible CONCERNS with named obligations.
2. Confirm every triggered design checkpoint and technical design review is current, and every required checkpoint has a completed design fan-out result.
3. Consume the approved test plan or the current typed not-expected decision. Planning never creates or repairs test design.
4. Use `planning-and-task-breakdown` to produce dependency-ordered tasks with:
   - one reviewable diff story or explicit coupling rationale;
   - source/owner anchors and approved design/test scenario mapping;
   - implementation obligations, legacy cleanup disposition, and explicit quality constraints;
   - fresh proof commands and pass/fail observables proportional to the claim;
   - material-risk checkpoints;
   - Goal-ready completion condition distinct from blocked-stop behavior.
5. Run the read-only ledger review required by the canonical planning/task-readiness contract. Reconcile and repair planning-local defects, mark prior readiness stale, and launch fresh re-review at the same or stronger model tier. A finding that changes behavior, design, test strategy, or ownership reopens that owner.
6. Close only with current `PASS`, eligible `CONCERNS`, eligible `WAIVED`, or an honest blocker. Planning handoff readiness authorizes only the next implementation macro phase.

Repository-standing authorization covers read-only ledger review. Missing runtime capability is not a valid `Ledger-review fan-out rationale:`; use the configured fallback before blocking.

## Success, Blocked Stop, And Reopen

Success requires a complete approved ledger, current source traceability, scenario and proof mapping, explicit cleanup, checkpoint coverage, no TBD or hidden design work, and a current readiness verdict that supports implementation handoff.

Stop blocked when upstream approval is missing/stale, a design fan-out or test-design obligation is unresolved, proof is too narrow or unavailable without a named handling rule, ledger review cannot run, or tasking exposes a decision not owned by planning.

Reopen specification for behavior/scope, technical design for mechanism/ownership, test design for scenarios/proof levels, or workflow planning for routing. Perform ledger-local repair and fresh task review inside planning. Stop before implementation begins; render its Goal prompt only after planning closes.
