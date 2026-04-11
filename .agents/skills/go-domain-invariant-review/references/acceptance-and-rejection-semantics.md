# Acceptance And Rejection Semantics

## Behavior Change Thesis
When loaded for symptom "code changed whether a command is accepted, rejected, ignored, or already applied", this file makes the model preserve deterministic business semantics instead of likely mistake "comment on error style or validation placement generically."

## When To Load
Load this when a review touches command handlers, validation placement, domain errors, no-op behavior, duplicate handling, event consumers, request acceptance, rejection paths, or code that changes whether the system treats input as accepted, rejected, ignored, or already applied.

## Decision Rubric
- Acceptance is domain behavior: success must mean the operation was accepted or was already accepted by explicit contract.
- Report a finding when the diff reports success for a forbidden command, rejects an approved command, converts a hard rejection into a silent no-op, or changes deterministic domain errors in a way that alters business meaning.
- Separate commands from events: commands may be rejected; events are facts whose local policy may be ignore, quarantine, compensate, or investigate.
- Escalate when the approved contract does not say whether duplicate or stale input is success, rejection, no-op, or asynchronous handling.

## Imitate
```text
[high] [go-domain-invariant-review] internal/applications/service.go:57
Issue:
The approved lifecycle makes `rejected` terminal, but `Approve` now returns `nil` when `StatusRejected` is present. That converts a forbidden approval command into a successful no-op instead of a deterministic rejection.
Impact:
Callers and audit logs can record the approval command as successful even though the application remained rejected, hiding failed manual review and breaking downstream obligation tracking.
Suggested fix:
Return the existing terminal-state domain error before any success path, and keep duplicate-success behavior only for states the local contract explicitly marks idempotent.
Reference:
Local application lifecycle spec or tests for terminal rejection.
```

Copy the shape: accepted/rejected/ignored classification, changed observable result, caller or audit consequence.

## Reject
```text
[low] internal/applications/service.go:57
This should return a better error.
```

Failure: this does not name the acceptance rule or why the changed result matters to the business.

## Agent Traps
- Do not demand exceptions instead of result errors, or result errors instead of exceptions, unless local code already defines that contract.
- Do not flag a no-op as wrong when the approved behavior says duplicate commands are idempotent success.
- Do not reject domain events merely because their payload is surprising; review the local event policy.
- Do not invent a validation rule from field names alone when no local source supports business rejection.
