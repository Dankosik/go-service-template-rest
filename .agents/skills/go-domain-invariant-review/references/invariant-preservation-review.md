# Invariant Preservation Review Examples

## When To Load
Load this when a review touches constructors, factories, mutators, aggregate methods, application-service guards, repository saves, direct field mutation, value object creation, totals, limits, quotas, eligibility, ownership, or other rules that must remain true after every accepted operation.

Use approved repo specs, domain docs, task artifacts, and tests as the business-rule authority. External links below only calibrate the review lens: aggregates and domain objects protect consistency boundaries, but the actual invariant comes from the local repo.

## Review Lens
Ask which operation can leave the domain in a state the business says must never be observable. Look for bypass paths, guard movement, direct field assignment, guard-only-in-handler drift, invalid object construction, and saves that persist invalid state.

## Bad Finding Example
```text
[medium] internal/orders/service.go:72
This looks like an anemic domain model. Move the logic into the aggregate.
```

Why it fails: it redesigns shape without naming the local invariant, the bypass path, or the business impact.

## Good Finding Example
```text
[critical] [go-domain-invariant-review] internal/orders/service.go:72
Issue:
Approved behavior says submitted orders are price-locked, but `AddLine` appends the line and recalculates `Total` before checking `order.Status`. A caller can persist a changed total on an already submitted order.
Impact:
Customer-visible and billing totals can diverge after checkout commitment, which can create undercharge, overcharge, or fulfillment disputes for an order that should be immutable.
Suggested fix:
Check `order.Status == StatusDraft` before mutating `Lines` or `Total`, or route the update through the existing domain method that already enforces the draft-only rule.
Reference:
Local order lifecycle spec or tests for price lock; external aggregate guidance is calibration only.
```

## Non-Findings To Avoid
- Do not flag public fields, setters, or service-layer checks by themselves when no local invariant bypass is shown.
- Do not require a new aggregate, value object, or DDD pattern because the code is not textbook DDD.
- Do not invent rules such as "orders can never change after creation" unless a repo artifact, test, or clear code-visible contract supports it.
- Do not report a style concern as an invariant finding when the bad state cannot be accepted or persisted.

## Smallest Safe Correction
Prefer a local correction that restores the invariant:
- move the guard before mutation and persistence;
- use the existing domain method or constructor that enforces the rule;
- add a missing validation branch at the one bypass path;
- reject or roll back before any invalid state is saved;
- keep the correction in the current owner when the invariant ownership is already clear.

## Escalation Cases
Escalate instead of redesigning in review when:
- the approved invariant is absent, contradictory, or now disputed;
- multiple packages own incompatible versions of the same rule;
- preserving the invariant needs a new consistency boundary, transaction model, or cross-service process;
- the smallest fix would change acceptance behavior for public clients;
- a data repair or migration is required because invalid states already exist.

## Source Links From Exa
- [Microsoft Learn: Use tactical DDD to design microservices](https://learn.microsoft.com/en-us/azure/architecture/microservices/model/tactical-domain-driven-design)
- [Martin Fowler: Domain Model](https://martinfowler.com/eaaCatalog/domainModel.html)
- [Martin Fowler: Test Invariant](https://martinfowler.com/bliki/TestInvariant.html)
- [Vaughn Vernon: Effective Aggregate Design, Part I](https://www.dddcommunity.org/wp-content/uploads/files/pdf_articles/Vernon_2011_1.pdf)
