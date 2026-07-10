---
name: test-design-session
description: "Own the user-started test-design macro phase: produce or repair test-plan.md, run independent QA review, repair findings, and obtain fresh re-review before planning."
---

# Test Design Session

## Eligibility And Outcome

Use when routing enters test-design, the reviewed spec and required approved design exist, and risk or proof complexity requires scenario decisions before task breakdown. The phase sequence is:

`technical design review -> test-design -> planning`

Skip SHAPE-DIRECT, missing or stale upstream approval, ordinary test implementation from an approved ledger, and tasks where the owning upstream artifact already records a current test-plan not-expected decision.

The outcome is a reviewed scenario contract with stable TD-* IDs and a current verdict, or an explicit current not-expected disposition with trigger rationale, proof carrier, and reopen condition.

## Canonical Owners

- [Test Design](../../../docs/spec-first-workflow/phases/test-design.md) owns triggers, test-plan shape, scenarios, fan-out, planning handoff, and stop rule.
- [Artifact model](../../../docs/spec-first-workflow/shared/artifact-model.md) owns typed state, no-artifact semantics, routing identity, and phase-control eligibility.
- [Subagents and handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) owns fan-out gates, authorization wording, resume order, and final prompt rendering.
- `go-qa-tester-spec` supplies the scenario-design method; `go-qa-tester` belongs to later implementation.

Load repository command docs or narrow source/test files only when they can change proof level, observability, determinism, or fixture ownership.

## Allowed Side Effects

This session may create or repair task-local `test-plan.md`, update existing `workflow-plan.md`, and create or update `workflow-plans/test-design.md` only when ROUTING-PHASE-CONTROL allows it.

It must not edit approved behavior/design, `tasks.md`, production code, test code, migrations, fixtures, generated output, implementation readiness, or validation closeout.

## Unique Method

1. Extract behavior deltas, invariants, failure semantics, compatibility constraints, accepted risks, and upstream proof obligations.
2. Decide whether a separate test plan is expected using the canonical trigger. Do not edit an approved upstream carrier merely to add a missing not-expected decision; reopen its owner.
3. Run or record `Test-design fan-out: complete | scoped_down | local_only | blocked | not_expected`.
4. Use `go-qa-tester-spec` to define stable TD-* scenarios with source anchor, setup, proof level, pass/fail observable, fail-before expectation or explicit waiver, determinism constraint, owner layer, and reopen target.
5. Keep task order, helper names, fixture internals, and test-code skeletons out of test design.
6. Mark an expected test plan review-ready, invoke a fresh read-only `qa-agent` review, repair test-design-owned findings, invalidate the old verdict, and obtain fresh re-review at the same or stronger tier.

Repository-standing authorization covers required read-only lanes. Use the configured fallback before blocking; do not treat missing runtime capability as local-only review.

## Success, Blocked Stop, And Reopen

Success requires either an approved `test-plan.md` with current `PASS` or eligible `CONCERNS` review over its final revision, or a current typed not-expected decision in an eligible owner. Workflow control, when present, must agree on artifact expectation, procedural gate state, review verdict, fan-out result, validity, and next action.

Stop blocked when upstream approval is missing/stale, behavior or ownership is undecided, scenario proof cannot be made observable/deterministic, a required lane cannot run, or no writable/current owner can carry a needed not-expected decision.

Reopen specification for behavior, technical design for ownership or mechanism, or workflow planning for routing. Perform scenario repair and fresh re-review inside test design. Stop before `tasks.md`; render the planning prompt only after the test-design macro phase closes.
