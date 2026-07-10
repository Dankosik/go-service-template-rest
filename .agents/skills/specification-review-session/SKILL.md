---
name: specification-review-session
description: "Run the internal read-only specification-review checkpoint for an active specification session, returning anchored advisory findings and a fresh verdict recommendation without editing the spec."
---

# Specification Review Session

## Eligibility And Outcome

Use after a non-trivial task-local `spec.md` is review-ready, when its owning specification session requests review, or when a repaired spec needs a fresh verdict. It may also serve an explicitly read-only user review request. Skip direct work with no expected spec and draft or missing specs.

The outcome is a distinct, read-only advisory result anchored to the exact spec revision, with recommended `review_verdict=PASS|CONCERNS|FAIL`, proof obligations, finding owners, and the smallest reopen target. The specification root records authoritative gate state.

## Canonical Owners

- [Specification review](../../../docs/spec-first-workflow/phases/specification-review.md) owns eligibility, lens coverage, verdict semantics, carrier choice, and the phase stop rule.
- [Artifact model](../../../docs/spec-first-workflow/shared/artifact-model.md) owns typed state, routing identity, lifecycle/validity, and phase-control eligibility.
- [Subagents and handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) owns lane gates, authorization wording, resume order, and final prompt rendering.
- [Specification](../../../docs/spec-first-workflow/phases/specification.md) owns every content repair.

Read only the review packet named by the canonical phase. Add a domain review skill or narrow source/contract read only when that lens can change the verdict.

## Allowed Side Effects

This checkpoint is read-only with respect to every repository file and external state. It returns evidence to the specification root; only the root records workflow state or repairs `spec.md`.

## Unique Method

1. Validate the review anchor: spec path, routing identity, review-ready lifecycle, accepted brief, and required evidence bundle.
2. Preserve the mandatory independent review with at least one focused lane. Add lanes only for additional concrete independent bounded falsification questions that can change readiness and materially benefit from separate context. Default to no more than three concurrently active subagent lanes.
3. Require every material finding to include a `Spec anchor`, evidence, classification, and proposed owner; do not patch the spec during review.
4. Reconcile findings as the orchestrator:
   - PASS: no approval blocker survives;
   - CONCERNS: no blocker survives, and accepted risks plus downstream proof obligations are named;
   - FAIL: at least one blocker survives and the smallest specification, research, or intake reopen target is named.
5. Return the reviewed revision, model route, fresh-context attestation, finding closure table when this is a follow-up, and verdict recommendation separately from spec lifecycle.

Repository-standing authorization covers this read-only checkpoint. If no independent execution surface is available, return that capability blocker; do not self-review locally.

## Success, Blocked Stop, And Reopen

Success requires complete required lens coverage, anchored findings, a fresh-context result, an advisory verdict, explicit readiness consequence, and a return target to the specification root. Mandatory specification review has no not-expected or waiver route for a non-trivial spec.

Stop blocked when the spec is not review-ready, routing identity is stale or conflicting, required review evidence or lanes are unavailable, or a finding needs an upstream decision before a verdict can be honest.

Return every verdict to the active specification root. `PASS` or eligible `CONCERNS` lets it close the macro phase; `FAIL` or actionable concerns trigger same-session repair and fresh re-review unless an earlier macro phase owns the blocker. This checkpoint never renders a user handoff prompt.
