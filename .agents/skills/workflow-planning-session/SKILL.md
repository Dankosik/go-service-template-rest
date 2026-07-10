---
name: workflow-planning-session
description: "Own a dedicated workflow-planning session after Phase 0 and the orchestrator's SHAPE-* classification when durable cross-phase or multi-session control is required. Record or repair workflow control, run the required adequacy check, and stop before research or specification."
---

# Workflow Planning Session

## Eligibility And Outcome

Use only after Phase 0 has an accepted brief and the orchestrator has selected a shape under the ordered SHAPE-* rules. Enter this session when durable master control is required or materially stale and the task has not yet entered research or specification.

Do not use it to decide user-owned scope, classify a shape before intake, repair a later phase artifact, or add ceremony to a direct or lean task that does not need durable control.

The outcome is current workflow routing that tells a context-blind next session exactly which one phase starts next, without duplicating specification, design, test, planning, or implementation authority.

## Canonical Owners

- [Workflow router](../../../docs/spec-first-workflow.md) owns the loading path and phase selection.
- [Artifact model](../../../docs/spec-first-workflow/shared/artifact-model.md) owns shape consequences, durable-control triggers, typed state, routing identity, transitions, and phase-control eligibility.
- [Subagents and handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) owns lane gates, authorization wording, resume order, and final prompt rendering.
- [Intake](../../../docs/spec-first-workflow/phases/intake.md) owns accepted-brief readiness.

Read only those owners whose decision is active. Load a bundled reference only when its named issue can change the result:

- `execution-shape-selection.md` for a disputed recorded classifier result;
- `artifact-expectation-matrix.md` for unclear artifact consequences;
- `research-mode-and-fanout-lanes.md` for research routing;
- `control-file-authoring-split.md` for master versus phase-control ownership;
- `adequacy-challenge-and-stop-boundary.md` for adequacy or boundary failures.

## Allowed Side Effects

This session may create or repair task-local `workflow-plan.md` and only the phase-control files allowed by ROUTING-PHASE-CONTROL. It may update routing status, artifact expectations, blockers, reopen targets, and the next-session context bundle.

It must not write `spec.md`, research notes, design artifacts, `test-plan.md`, `tasks.md`, code, tests, generated files, or implementation handoff. It must not classify or reclassify shape through an adequacy challenger.

## Unique Method

1. Verify the accepted brief, selected SHAPE-* rule, complete FULL-*/DIRECT-*/LEAN-* audit, agent_request, actor, proof obligation, routing identity, and transition history against the canonical owners.
2. Resolve research expectation independently from phase-control eligibility and same-session collapse.
3. Write the smallest durable routing record. Phase files carry only phase-local control; they never become a second spec, design, test plan, or ledger.
4. Run workflow-plan-adequacy-challenge whenever a canonical ADEQUACY-* trigger applies. The challenger may falsify the recorded route but never classify, edit, approve, or reclassify state.
5. Reconcile findings as the orchestrator and stop before the next phase.

Repository-standing `capability_only` authorization covers the required read-only adequacy check. Missing runtime capability cannot justify local-only, scoped-down, waived, or not-expected treatment until the configured independent fallback also fails.

## Success, Blocked Stop, And Reopen

Success requires current routing identity, proportional artifact expectations, honest typed state, the required adequacy result, one next phase, a selective context bundle, and a reached session boundary. Research skip routes a new session to specification; it does not authorize same-session specification.

Stop blocked when intake is not accepted, shape evidence is incomplete or contradictory, a required adequacy lane cannot run, routing identities conflict, or a required owner decision is missing. Name the evidence gap and smallest reopen target.

Reopen intake for user-owned framing, workflow planning for routing or transition repair, or the owning downstream phase for artifact-local repair. Render any actionable next-session prompt only through [Subagents and handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md); do not store the full prompt in repository artifacts.
