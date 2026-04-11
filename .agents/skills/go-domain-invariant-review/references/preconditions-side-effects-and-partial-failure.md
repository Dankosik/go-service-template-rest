# Preconditions, Side Effects, And Partial Failure Review Examples

## When To Load
Load this when a review touches payment capture, refunds, emails, inventory reservation, entitlement changes, event publication, webhooks, external API calls, repository saves, transaction boundaries, compensation, or any side effect that can outlive a rejected domain operation.

Use approved repo specs, task artifacts, tests, and local side-effect contracts as the authority. External outbox and workflow sources calibrate partial-failure questions; they do not authorize changing the domain process during review.

## Review Lens
Preconditions should guard side effects, not explain them afterward. A finding is warranted when an irreversible or externally visible effect happens before the domain operation is accepted, when a save and side effect can split into contradictory outcomes, or when a failure path leaves a mixed state the local contract forbids.

## Bad Finding Example
```text
[medium] internal/order/service.go:41
Payments should use a saga or outbox.
```

Why it fails: it jumps to architecture without first naming the local precondition and the concrete partial-failure outcome.

## Good Finding Example
```text
[critical] [go-domain-invariant-review] internal/order/service.go:41
Issue:
Approved behavior says only `authorized` orders may complete and rejected completion must not charge the customer, but `Complete` calls `payments.Capture` before checking `order.Status`.
Impact:
A non-authorized order can be charged and then rejected, leaving the customer with an irreversible payment side effect for an order the domain refused to complete.
Suggested fix:
Move the `StatusAuthorized` guard before `payments.Capture`, and only perform capture after all local completion preconditions pass. If capture must be coordinated with persistence, escalate to the existing reliability/data design rather than inventing a new workflow in review.
Reference:
Local order completion spec or test; outbox and partial-failure sources are calibration only.
```

## Non-Findings To Avoid
- Do not flag every external call as a domain issue; tie it to a violated precondition, partial state, duplicate effect, or forbidden mixed outcome.
- Do not prescribe sagas, outbox tables, or distributed transactions when a local guard move fixes the bug.
- Do not require compensation for a side effect that is provably after acceptance and locally idempotent.
- Do not treat logging, metrics, or tracing as domain side effects unless they change business obligations or leak sensitive semantics into user-visible behavior.

## Smallest Safe Correction
Prefer the local ordering fix:
- evaluate all domain preconditions before calling external systems;
- persist accepted state before non-critical notification side effects when local contract allows;
- use an existing outbox or idempotent effect path when one already exists;
- make side effects conditional on the successful domain transition;
- return the original domain rejection without touching external systems.

## Escalation Cases
Escalate when:
- there is no local order that can make state change and side effect safe;
- a new compensation, saga, outbox, or reconciliation policy is required;
- the side effect is irreversible and the approved business contract does not define the failure state;
- database transaction, cache, or event-publish mechanics are the root cause;
- the fix changes public acceptance or retry semantics.

## Source Links From Exa
- [Microservices.io: Transactional outbox](https://microservices.io/patterns/data/transactional-outbox.html)
- [Microservices.io: Idempotent Consumer](https://microservices.io/patterns/communication-style/idempotent-consumer.html)
- [Microsoft Learn: Use tactical DDD to design microservices](https://learn.microsoft.com/en-us/azure/architecture/microservices/model/tactical-domain-driven-design)
- [NILUS: State Machines in Microservices Workflows](https://www.nilus.be/blog/state_machines_in_microservices_workflows/)
