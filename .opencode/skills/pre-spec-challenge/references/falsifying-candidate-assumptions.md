# Falsifying Candidate Assumptions

## Behavior Change Thesis
When loaded for a convenience assumption, this file makes the model ask failure-oriented falsification questions instead of broad checklist prompts or alternate designs.

## When To Load
Load this when candidate synthesis depends on claims like "clients will not retry," "TTL cleanup is enough," "operators can fix this manually," "UUID secrecy is sufficient," "frontend disables the button," or "v1 can ignore the edge case."

## Decision Rubric
- Isolate the assumption carrying the candidate path. If it fails and no planning choice changes, drop it.
- Invert the assumption into a concrete production failure: retry after timeout, TTL lag, object storage expiry mismatch, tenant guessing, skipped manual fix, stale cache after commit.
- Ask what invariant, actor promise, data contract, or validation proof breaks if the assumption is false.
- Prefer one question that falsifies the path over three questions that request elaboration.
- Do not propose the replacement design. Name what the answer would change in planning.

## Imitate
- "What breaks if the client retries after a timeout and the first request commits after the response is lost?"
  - Copy the specific failure timing; it is harder to wave away than "what about retries?"
- "If TTL cleanup lags by 24 hours, which user-visible or operator-visible state becomes wrong?"
  - Copy the move from storage mechanics to visible correctness.
- "If support skips the manual DB fix during an incident, which invariant remains violated and who owns recovery?"
  - Copy the pressure on manual workarounds as part of the design, not an escape hatch.

## Reject
- "What about retries?"
  - Fails because it names a category but not the failure that could change the API or data contract.
- "Is TTL safe?"
  - Fails because it does not say what correctness promise TTL delay might violate.
- "Consider using idempotency keys."
  - Fails because it jumps to design authorship before the challenged assumption is answered.
- "Do we need more scale testing?"
  - Fails unless it names the metric or repository fact that would falsify the scale assumption.

## Agent Traps
- Restating the candidate assumption as a question: "Are we sure frontend disabling is enough?"
- Challenging every edge case equally instead of preserving only the ones that change planning.
- Treating low probability as low impact without checking irreversibility or tenant exposure.
- Replacing challenge with a preferred architecture pattern.
