# Preconditions, Side Effects, And Partial Failure

## Behavior Change Thesis
When loaded for symptom "an external or durable effect can happen before the domain operation is accepted", this file makes the model review guard-before-effect ordering and mixed outcomes instead of likely mistake "prescribe sagas, outbox, or compensation before proving a local partial-failure bug."

## When To Load
Load this when a review touches payment capture, refunds, emails, inventory reservation, entitlement changes, event publication, webhooks, external API calls, repository saves, transaction boundaries, compensation, or any side effect that can outlive a rejected domain operation.

## Decision Rubric
- A domain finding exists when an irreversible or externally visible effect can happen before all local preconditions pass, or when a save and side effect can split into contradictory outcomes the local contract forbids.
- Name the rejected operation and the lasting effect together; one without the other is usually not enough for this lane.
- Prefer moving guards before effects, making effects conditional on accepted state, or using an existing idempotent/outbox path when one already exists.
- Escalate when no local ordering can make state and side effect safe, or a new compensation, saga, outbox, transaction, or reconciliation policy is required.

## Imitate
```text
[critical] [go-domain-invariant-review] internal/order/service.go:41
Issue:
Approved behavior says only `authorized` orders may complete and rejected completion must not charge the customer, but `Complete` calls `payments.Capture` before checking `order.Status`.
Impact:
A non-authorized order can be charged and then rejected, leaving the customer with an irreversible payment side effect for an order the domain refused to complete.
Suggested fix:
Move the `StatusAuthorized` guard before `payments.Capture`, and only perform capture after all local completion preconditions pass. If capture must be coordinated with persistence, escalate to the existing reliability/data design.
Reference:
Local order completion spec or rejected-completion test.
```

Copy the shape: precondition, effect, partial outcome, local reorder or explicit escalation.

## Reject
```text
[medium] internal/order/service.go:41
Payments should use a saga or outbox.
```

Failure: this jumps to architecture without naming the violated precondition or partial outcome.

## Agent Traps
- Do not flag every external call as a domain issue; tie it to a rejected operation, duplicate effect, forbidden mixed outcome, or violated side-effect contract.
- Do not require compensation for a side effect that is provably after acceptance and locally idempotent.
- Do not treat logging, metrics, tracing, or queued-but-undispatched in-memory domain events as domain side effects unless they change business obligations, persist, dispatch, or trigger user-visible behavior before acceptance.
- Do not treat an outbox or event relay as full protection when duplicate delivery can repeat the business effect; pair it with an existing idempotency or transition guard, or escalate.
- Do not hide a needed data/reliability escalation inside a review-local suggested fix.
