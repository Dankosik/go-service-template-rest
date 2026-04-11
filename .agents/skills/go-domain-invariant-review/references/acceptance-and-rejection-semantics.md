# Acceptance And Rejection Semantics Review Examples

## When To Load
Load this when a review touches command handlers, validation placement, domain errors, no-op behavior, duplicate handling, event consumers, request acceptance, rejection paths, or code that changes whether the system treats a request as accepted, rejected, ignored, or already applied.

Use approved repo specs, API contracts, task artifacts, tests, and domain wording as the authority. External CQRS and DDD sources only help separate commands that can be rejected from events that represent facts.

## Review Lens
Acceptance semantics are domain behavior. A finding is warranted when changed code reports success for a rejected command, rejects an approved command, converts a hard rejection into a silent no-op, treats an event fact like a user command, or changes deterministic domain errors in a way that alters business meaning.

## Bad Finding Example
```text
[low] internal/applications/service.go:57
This should return a better error.
```

Why it fails: it does not say which acceptance rule changed or why the business outcome matters.

## Good Finding Example
```text
[high] [go-domain-invariant-review] internal/applications/service.go:57
Issue:
The approved lifecycle makes `rejected` terminal, but `Approve` now returns `nil` when `StatusRejected` is present. That converts a forbidden approval command into a successful no-op instead of a deterministic rejection.
Impact:
Callers and audit logs can record the approval command as successful even though the application remained rejected, which can hide failed manual review and break downstream obligation tracking.
Suggested fix:
Return the existing terminal-state domain error before any success path, and keep duplicate-success behavior only for states the local contract explicitly marks idempotent.
Reference:
Local application lifecycle spec or tests for terminal rejection; command/event distinction is external calibration only.
```

## Non-Findings To Avoid
- Do not demand exceptions instead of result errors, or result errors instead of exceptions, unless the local contract requires it.
- Do not flag a no-op as wrong when the approved behavior says duplicate commands are idempotent success.
- Do not reject domain events merely because their payload is surprising; if events are facts, the local policy may be ignore, quarantine, compensate, or investigate.
- Do not invent a validation rule from field names alone when no local source supports business rejection.

## Smallest Safe Correction
Prefer a local semantics fix:
- restore the existing domain error or result type for forbidden commands;
- return success only after the domain operation is actually accepted or already accepted by contract;
- keep duplicate and stale inputs on their documented path;
- add a small branch that preserves terminal-state rejection;
- update the changed test expectation only when an approved artifact already authorizes the semantics change.

## Escalation Cases
Escalate when:
- the approved contract does not say whether duplicate or stale commands are success, rejection, or no-op;
- an API-visible status code, error code, or response body must change;
- commands and events are mixed enough that ownership of rejection policy is unclear;
- the change implies a new business acceptance rule rather than restoring an existing one;
- a consumer needs a dead-letter, quarantine, or reconciliation policy not present in local design.

## Source Links From Exa
- [Enterprise Craftsmanship: When to validate commands in CQRS?](https://enterprisecraftsmanship.com/posts/validate-commands-cqrs)
- [Microsoft Learn: Use tactical DDD to design microservices](https://learn.microsoft.com/en-us/azure/architecture/microservices/model/tactical-domain-driven-design)
- [Martin Fowler: Domain Model](https://martinfowler.com/eaaCatalog/domainModel.html)
- [NILUS: State Machines in Microservices Workflows](https://www.nilus.be/blog/state_machines_in_microservices_workflows/)
