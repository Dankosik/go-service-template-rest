# Review Findings And Convergence

Load the Finding Envelope for a non-implementation or specialist review. Apply
Convergence only to a triggered non-implementation independent review. Review
lanes use the [Lane Brief](subagents-and-handoff.md#lane-brief); the owning root
retains repair, acceptance, and movement.

## Finding Envelope

Lead with surviving findings in severity order. Each actionable finding names
its anchor, impact on the accepted outcome, blocker/concern/non-blocking
classification, and smallest action or reopen owner. If no finding survives,
state the evidence boundary without padding the review.

Falsify a finding before classifying it. A blocker names the observation that
would disprove it and that observation's current result. When disproof was not
attempted or could not run, report a concern with that gap rather than a
blocker.

## Convergence

One triggered reviewer reads the fixed artifact or diff and returns anchored,
falsified findings, evidence boundary, and verdict without editing or approving
its repair. Add one bounded specialist only for a concrete high-impact question
the reviewer cannot credibly close; an unchanged candidate gets no second broad
reviewer.

A phase may instead define one required complementary review panel when its
contract partitions one fixed artifact into disjoint, jointly exhaustive
evidence boundaries and names the overlap exclusions, synthesis owner, verdict
threshold, and focused re-review rule. That panel replaces the default broad
reviewer for the same boundary; each lane stays read-only and the root retains
acceptance.

A review moves on `PASS`. `CONCERNS` moves only after each concern has a proof
or risk owner, observable, and reopen condition and leaves no behavior or
mechanism for Implementation to invent. Everything else is `FAIL` and receives
repair or upstream reopen plus focused fresh review. Reuse unaffected findings.
A material mutation invalidates only affected findings and proof; wording-only
edits and concern dispositions do not trigger another review.
