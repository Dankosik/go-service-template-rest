# Falsifying Candidate Assumptions

## Behavior Change Thesis
When loaded for a convenience assumption, this file makes the model ask failure-oriented falsification questions instead of broad checklist prompts or alternate designs.

## When To Load
Load this when candidate synthesis depends on claims like "clients will not retry," "TTL cleanup is enough," "operators can fix this manually," "UUID secrecy is sufficient," "frontend disables the button," or "v1 can ignore the edge case."

## The Move
Isolate the assumption carrying the candidate path — if its failure changes no planning choice, drop it. Invert it into a concrete production failure (a retry after timeout whose first request commits late, TTL lag, object-storage expiry mismatch, tenant guessing, a skipped manual fix, stale cache after commit) and ask what invariant, actor promise, data contract, or validation proof breaks. One question that falsifies the path beats three that request elaboration; the question names what the answer would change in planning and leaves the replacement design unproposed. Weigh irreversibility and tenant exposure before dismissing a low-probability failure, and preserve only the edge cases that change planning.

## Imitate
- "What breaks if the client retries after a timeout and the first request commits after the response is lost?"
  - Copy the specific failure timing; it is harder to wave away than "what about retries?"
- "If TTL cleanup lags by 24 hours, which user-visible or operator-visible state becomes wrong?"
  - Copy the move from storage mechanics to visible correctness.
- "If support skips the manual DB fix during an incident, which invariant remains violated and who owns recovery?"
  - Copy the pressure on manual workarounds as part of the design, not an escape hatch.

## Reject
- "What about retries?"
  - Names a category but not the failure that could change the API or data contract; restating the assumption as a question ("Are we sure frontend disabling is enough?") is the same miss.
- "Consider using idempotency keys."
  - Jumps to design authorship before the challenged assumption is answered.
