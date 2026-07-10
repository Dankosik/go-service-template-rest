---
name: specification-session
description: "Own the user-started specification macro phase: author or repair task-local spec.md, run mandatory independent review, repair findings, and obtain fresh re-review before handing off to the next macro phase."
---

# Specification Session

## Eligibility And Outcome

Use when routing enters specification and the accepted brief plus required evidence are sufficient to decide the in-scope behavior. Use it for a new spec, repair after clarification, or repair after specification-review FAIL.

Do not use it while user-owned scope, an approval-changing fact, or a required provider/source-of-truth contract is unknown. It owns specification review internally; do not combine it with technical design, test design, planning, or implementation.

The outcome is one reviewed or explicitly blocked `spec.md` containing orchestrator-owned decisions, a current specification-review verdict, and the next macro-phase route.

## Canonical Owners

- [Specification phase](../../../docs/spec-first-workflow/phases/specification.md) owns the spec shape, review-ready bar, Risk Challenge, clarification disposition, and stop rule.
- [Artifact model](../../../docs/spec-first-workflow/shared/artifact-model.md) owns artifact depth, typed lifecycle/validity, routing identity, and reclassification.
- [Subagents and handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) owns lane gates, authorization wording, resume order, and final prompt rendering.
- [Specification review](../../../docs/spec-first-workflow/phases/specification-review.md) owns the internal read-only review method and verdict semantics.

Use `spec-document-designer` for the actual spec normalization. Load a bundled specification-session reference only when its named issue can change readiness: entry readiness, clarification flow, allowed writes, workflow-control projection, or review handoff.

## Allowed Side Effects

This session may create or repair task-local `spec.md`; record its internal specification-review cycle; update existing `workflow-plan.md`; and create or update specification/specification-review phase-control files only when ROUTING-PHASE-CONTROL allows those carriers.

It must not write design artifacts, `test-plan.md`, `tasks.md`, code, tests, generated output, implementation handoff, or later-macro-phase control files.

## Unique Method

1. Verify that intent, scope, evidence, contract/source ownership, and approval-changing questions are closed enough for specification.
2. Use `spec-document-designer` to turn accepted inputs into explicit behavior/contract delta, decisions, non-goals, risks, proof obligations, reopen conditions, and cleanup disposition.
3. Run formal spec-clarification-challenge when the canonical predicate requires it. Otherwise run the lean Risk Challenge and record only risk_challenge_outcome=PASS|CONCERNS|RECLASSIFY_FULL.
4. Reconcile lane evidence into the spec; raw lane output never becomes authority.
5. When the spec reaches review-ready, invoke a fresh read-only specification-review lane against the exact revision. Reconcile its advisory verdict, repair every specification-owned actionable finding, mark the old verdict stale, and launch fresh re-review at the same or stronger model tier.
6. If Risk Challenge or new protected evidence invalidates the route, use the guarded upward reclassification transaction before continuing.

Repository-standing authorization covers required read-only lanes. If the primary spawn surface is unavailable, use the configured independent read-only fallback; do not convert required review to local-only.

## Success, Blocked Stop, And Reopen

Success requires a current `PASS` or eligible `CONCERNS` specification-review verdict over the final spec revision, no TBD, open alternative, or implementation-time product choice, current artifact state/validity, required dependency/OSS and Pattern Fit decisions, legacy-surface disposition, and mapped accepted risks/proof obligations.

Stop blocked when a user decision, research fact, contract, source owner, required clarification lane, or routing transaction is missing. Record the blocker in the eligible owner rather than inventing the answer.

Reopen intake for user-owned scope, research for evidence, or workflow planning for routing. Perform specification-owned repair and fresh re-review inside this session. Render a next-session prompt only after the specification macro phase closes, through [Subagents and handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md); never store the full prompt in an artifact.
