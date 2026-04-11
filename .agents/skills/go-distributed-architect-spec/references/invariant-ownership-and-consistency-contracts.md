# Invariant Ownership And Consistency Contracts

## When To Load
Load this when the flow spans multiple services and the spec needs to decide who owns an invariant, whether the invariant is local-hard or process-eventual, how stale a projection may be, or what should happen when the owner is unavailable.

## Option Comparisons
- Local hard invariant: keep the decision inside one owner transaction. Choose this when business correctness requires a commit-time check, for example "credit may never be exceeded" at the moment an order is accepted.
- Cross-service process invariant: allow a durable process to converge through intermediate states. Choose this when pending, rejected, compensated, or repair states are acceptable to the business.
- Owner query or command: ask the owner for an authoritative decision. This costs latency and availability, but avoids writing from stale projections.
- Projection-based decision: use only for soft decisions or when the spec defines a staleness budget and fallback. Do not let a read model make an irreversible write decision.

## Good Flow Examples
- Order owns order lifecycle. Customer owns credit. Order creates `PENDING`, sends `ReserveCredit`, and only transitions to `APPROVED` from a version-checked reservation reply.
- Inventory owns reservation quantity. Checkout may show estimated availability from a projection, but final reservation is a command to Inventory with a business idempotency key.
- Pricing owns price. Cart can cache price for display, but checkout either revalidates with Pricing or fails with a documented stale-price response.

## Bad Flow Examples
- Order reads a replicated customer row and approves the order without Customer owning the credit decision.
- Two services each maintain "remaining inventory" and race to decrement their local copies.
- A spec says "eventual consistency will fix it later" without naming the owner, maximum staleness, repair trigger, or terminal failure state.

## Failure-Mode Examples
- Owner unavailable: the caller creates a pending process state or fails by contract; it does not make a hard decision from a stale cache.
- Stale projection: the write path detects projection lag over budget and queries the owner or returns a retryable/accepted response.
- Duplicate command: the invariant owner deduplicates by business key, returns the previous equivalent outcome, and does not reserve twice.
- Concurrent flows for one aggregate: state transitions use version checks or a durable uniqueness constraint so only one decision path wins.

## Exa Source Links
- [Microservices.io Saga pattern](https://microservices.io/patterns/data/saga.html)
- [Microservices.io saga consistency overview](https://microservices.io/post/microservices/2019/07/09/developing-sagas-part-1.html)
- [Dapr Workflow features and concepts](https://docs.dapr.io/developing-applications/building-blocks/workflow/workflow-features-concepts)
