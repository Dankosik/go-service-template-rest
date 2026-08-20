# Review Independence

Shared trigger for deciding whether a fixed artifact or implementation
acceptance unit needs an independent reviewer.

## Read When

- Deciding whether to open any independent review.
- Re-evaluating that decision after a material candidate or risk change.

## Trigger

Use an independent reviewer when the fixed boundary is high-impact,
hard-to-reverse, protected-domain, cross-owner, or materially contested and its
owning actor cannot credibly falsify it alone. The execution path,
artifact presence, and `tasks.md` alone do not trigger review; ordinary work
uses root self-review. An explicit user request for independent review also
triggers it.

## Route

- A required complementary panel explicitly owned by a phase is selected by
  that phase rather than by this trigger. Apply this trigger only to any
  remaining broader artifact boundary after consuming the panel's current
  receipts.
- When the trigger does not apply, continue with root self-review and do not
  load a review-specific branch.
- For a triggered non-implementation review, continue through the current
  phase's review adapter and [Review Findings And Convergence](review-findings-and-convergence.md#convergence).
- For implementation, evaluate this trigger only after a fixed unit candidate
  passes bounded root review and mapped validation. When it applies, load [Independent
  Implementation Review](implementation-review.md).

## Review Router

| Fixed boundary | Phase-owned review |
| --- | --- |
| Standalone research synthesis | [Research Review](../phases/research.md#review) |
| Completed specification | [Specification Review](../phases/specification-review.md) |
| Technical and Go-ownership design | [Technical Design Review](../phases/technical-design-review.md) |
| Non-obvious test design | [Test Design Review](../phases/test-design.md#review) |
| Executable ledger | [Task Review / Readiness](../phases/task-review-readiness.md) |
| Fixed implementation acceptance unit | [Independent Implementation Review](implementation-review.md) |

The artifact-owning phase repairs findings and receives the verdict. Review
method and finding shape stay with the selected branch and [Review Findings And
Convergence](review-findings-and-convergence.md), not the workflow router.

## Stop Rule

The decision names one fixed review boundary and selects no more than one
phase-owned review branch. A material candidate or risk change re-evaluates only
the affected boundary.
