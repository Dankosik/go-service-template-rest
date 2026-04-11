# Invariant Ownership And Consistency Contracts

## Behavior Change Thesis
When loaded for unclear invariant ownership, stale projection, or owner-unavailable symptoms, this file makes the model route the decision to the source-of-truth owner or a durable pending process instead of approving hard writes from cached read models or ownerless "eventual consistency."

## When To Load
Load when a cross-service flow must decide who owns an invariant, whether it is local-hard or process-eventual, how stale a projection may be, or what happens when the owner is unavailable.

## Decision Rubric
- If violating the invariant at commit time is unacceptable, keep the decision inside the owning service's local transaction.
- If the invariant spans owners, model it as a process invariant with explicit pending, rejected, compensated, or repair states.
- Use an owner query or command when a write path needs authoritative state; pay the latency/availability cost openly.
- Use projections for display or soft decisions only, unless the spec defines a staleness budget, lag check, and fallback.
- When the owner is unavailable, create a pending process state or fail by contract; do not make a hard decision from a stale cache.

## Imitate
- Order owns order lifecycle; Customer owns credit. Order creates `PENDING`, sends `ReserveCredit`, and transitions to `APPROVED` only from a version-checked reservation reply. Copy the separate lifecycle owner and invariant owner.
- Inventory owns reservation quantity. Checkout may display estimated availability from a projection, but final reservation is a command to Inventory with a business idempotency key. Copy the display-vs-authority split.
- Pricing owns price. Cart can cache price for display, but checkout revalidates with Pricing or fails with a documented stale-price response. Copy the explicit stale-read failure behavior.

## Reject
- Order reads a replicated customer row and approves the order without Customer owning the credit decision.
- Two services each maintain "remaining inventory" and race to decrement their local copies.
- A spec says "eventual consistency will fix it later" without naming the owner, maximum staleness, repair trigger, or terminal failure state.

## Agent Traps
- Treating the service that starts the flow as the owner of every invariant it touches.
- Calling a read model "eventually consistent" while using it to make irreversible writes.
- Naming a reconciliation job but not naming the authoritative owner it reconciles against.
- Letting availability pressure silently downgrade a hard invariant into a cached decision.

## Validation Shape
- Owner unavailable: the caller creates a pending process state or fails by contract; it does not make a hard decision from a stale cache.
- Stale projection: the write path detects projection lag over budget and queries the owner or returns a retryable/accepted response.
- Duplicate command: the invariant owner deduplicates by business key, returns the previous equivalent outcome, and does not reserve twice.
- Concurrent flows for one aggregate: state transitions use version checks or a durable uniqueness constraint so only one decision path wins.
