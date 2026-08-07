# Review Independence

Shared trigger for deciding whether a fixed artifact or implementation
acceptance unit needs an independent reviewer.

## Read When

- Deciding whether to open any independent review.
- Re-evaluating that decision after a material candidate or risk change.

## Trigger

Use an independent reviewer when the fixed boundary controls an orchestrated,
high-impact, hard-to-reverse, protected-domain, cross-owner, or materially
contested decision that its author cannot credibly falsify alone. Artifact or
`tasks.md` presence alone does not trigger a reviewer. Other structured
artifacts and ordinary implementation use root self-review. An explicit user
request for independent review also triggers it.

## Route

- A required complementary panel explicitly owned by a phase is selected by
  that phase rather than by this trigger. Apply this trigger only to any
  remaining broader artifact boundary after consuming the panel's current
  receipts.
- When the trigger does not apply, continue with root self-review and do not
  load a review-specific branch.
- For a triggered non-implementation review, continue through the current
  phase's review adapter and [Subagents And Review](subagents-and-handoff.md#non-implementation-review-convergence).
- For implementation, evaluate this trigger only after a fixed candidate passes
  bounded root review and mapped validation. When it applies, load [Independent
  Implementation Review](implementation-review.md).

## Stop Rule

The decision names one fixed review boundary and selects no more than one
phase-owned review branch. A material candidate or risk change re-evaluates only
the affected boundary.
