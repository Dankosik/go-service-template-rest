# Invalid State And Transition Review

## Behavior Change Thesis
When loaded for symptom "a construction, mutation, save, guard, or status update may admit a state the domain forbids", this file makes the model prove the reachable bad state instead of likely mistake "call the model anemic, or ask for a formal state machine, without showing a move the contract rejects."

## Decision Rubric
- A finding exists only when the diff can accept, persist, or expose state the local contract forbids — a guard bypassed, a transition the lifecycle rejects, an approved transition now blocked, or a terminal state weakened. A transition guard is an invariant guard; the granularity does not change the test.
- Name the rule in repository terms before naming the pattern: the accepted rule, the path that reaches around it, the bad state that survives, and what a caller or an audit reader then believes.
- Treat state names as business language when local specs, tests, fixtures, or callers give them product meaning. `retrying` and `published` are technical unless a product rule depends on them.
- Prefer the smallest owner-preserving fix: an existing domain method, a guard moved before the mutation, or a check the surrounding code already establishes.
- Escalate when the invariant is absent, contradictory, split across owners, or needs a new lifecycle state or consistency boundary — and when the guard can be lost to a concurrent writer, since that fix is a constraint rather than a reorder.
- Preserve approved no-op, rejection, and idempotent-success semantics exactly; `acceptance-and-rejection-semantics.md` owns the case where the diff changed which one applies.

## Reject
```text
This looks like an anemic domain model. Move the logic into the aggregate.
```
Failure: a reshaping request with no bad state and no local authority. It costs the same review attention as a real finding and cites nothing that can be checked.

```text
There are too many states here.
```
Failure: state-count taste. Without a broken transition rule there is nothing to preserve.

## Validation Shape
The finding is complete when a reader can name the accepted rule, the reachable bad state, and one assertion that fails today and passes after the smallest fix.
