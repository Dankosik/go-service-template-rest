# Acceptance And Rejection Semantics

## Decide
- Success must mean the operation was accepted, or was already accepted by an explicit contract. A finding exists when the diff reports success for a forbidden command, rejects an approved one, converts a rejection into a silent no-op, or changes a deterministic domain error so callers read a different business result.
- Commands and events are different: a command may be rejected; an event is a fact whose local policy is ignore, quarantine, compensate, or investigate. Rejecting an event as though it were a command discards a fact that already happened.
- Vocabulary that separates outcomes is part of the contract. `cancelled` and `expired`, `owner` and `creator`, `authorized` and `captured`, `available` and `reserved` are distinctions a rename can collapse; a finding needs the local rule that reads them differently, not a preference. Readability-only naming goes to `go-language-simplifier`, exported-API naming to `go-idiomatic`.
- Ask what a caller, an audit reader, or a support workflow would now believe. That reading is the impact; the changed line is only the anchor.
- Keep a no-op where the accepted behavior makes duplicate commands idempotent success, and keep the local error convention the surrounding code already establishes.
- Escalate when the accepted contract never says whether duplicate or stale input is success, rejection, no-op, or asynchronous — that is a missing decision, not a review finding.

## Reject
```text
This should return a better error.
```
Failure: no acceptance rule named and no business consequence, so nothing distinguishes it from taste.

```text
`ReservedCredit` would be a clearer name than `AvailableCredit`.
```
Failure: a naming preference unless a local rule treats reserved credit as unavailable. With that rule cited, the same line becomes a double-spend finding.

## Prove
The finding is complete when it classifies the result as accepted, rejected, ignored, or already applied both before and after the diff, and names the caller or audit consequence of the difference.
