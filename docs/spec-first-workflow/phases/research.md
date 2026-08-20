# Research

Use when current or external evidence can change a named task decision. Research
owns evidence closure and downstream implications, not the final behavior or
mechanism choice.

## Inputs

- the exact question, affected decision/criterion, and owner;
- leading hypothesis or live alternatives and their falsifiers;
- authoritative source boundary, applicable versions, freshness needs, and
  downstream input shape.

## Method

1. Route policy, target, or risk-tolerance choices to their owner; mechanism to
   Design; proof strategy to Test Design. Label each quantity as measured,
   external limit, forecast, accepted target, or assumption.
2. Before searching, fix the smallest falsifying evidence, authority and
   applicability boundaries, aliases/failure modes, counter-evidence surfaces,
   absence/conflict handling, and stop condition.
3. Follow decision-changing claims to primary authority. Test the leading
   implication against material counter-evidence and the strongest viable
   alternative. An unsearched or unavailable relevant surface is unknown, not
   absence.
4. Load only the triggered branch below. Load another only when its independent
   pressure can change a named disposition.
5. Stop when further authoritative inspection or a safe discriminating probe is
   unlikely to change, narrow, or reopen the decision.

## Conditional Methods

- repository/runtime/consumer divergence -> [Current-state baseline](research-branches.md#current-state-or-semantic-baseline);
- unfamiliar or current external behavior -> [Current external contract](research-branches.md#current-external-contract);
- a live solution family -> [Solution discovery](research-branches.md#solution-discovery-evidence);
- an empirical claim -> [Empirical claim or probe](research-branches.md#empirical-claim-or-probe);
- material conflict/freshness -> [Conflict or freshness](research-branches.md#conflict-or-freshness);
- a concrete later input/proof gap -> [Downstream input closure](research-branches.md#downstream-input-closure);
- an independent read-only question -> [Delegation](../shared/delegation.md).

## Output

Return a claim-organized synthesis separating fact, inference, conflict,
assumption, and unknown. Each material claim carries its primary locator,
scope/date/limits, counter-evidence disposition, affected downstream decision,
and refresh/reopen condition. When a solution choice is live, include viable
same-level candidates, local-fit evidence, exclusions, decision-flip conditions,
and saturation rationale without selecting the architecture. Persist under
`research/` only through [Artifacts](../shared/artifacts.md).

## Review

For an accepted `research only` boundary, apply shared
[Review](../shared/review.md) before returning `ready`. The reviewer checks
question/lens coverage, authority, applicability, freshness, falsification,
conflicts, and downstream disposition. A missing Research-owned answer is
`FAIL`; a bounded later proof risk may be `CONCERNS`.

## Exit And Reopen

Exit when the named next owner can act without repeating the search or
re-synthesizing sources. Reopen the smallest evidence or decision owner for a
required gap; do not write the final specification or design decision here.
