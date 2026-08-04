# Allocation And Pooling

## When To Load

Load this when a change reuses memory to reduce allocation: `sync.Pool`, a
shared buffer, a reset-and-refill struct, a preallocated slice kept between
calls, or a claim about GC pressure.

## Behavior Change Thesis

Fewer allocations per operation reads as proof that GC pressure fell, and a
pool's benchmark reads as proof that reuse happens. Both are weaker than they
look: `sync.Pool` is documented as a hint that may return nothing it was given,
so a tight benchmark loop measures a hit rate a bursty request path will not
see, and per-operation churn and retained heap are different measurements that
can move in opposite directions.

## Decision Rubric

- `sync.Pool`'s contract is explicit: any item may be removed automatically at
  any time without notification, and `Get` may ignore the pool and treat it as
  empty — callers may assume no relation between what `Put` stored and what
  `Get` returns. Correctness that depends on receiving a pooled object back is a
  defect, and a benchmark that never misses is not the production shape.
- Churn and retention need different evidence: `-benchmem` and `alloc_space`
  show allocation churn; `inuse_space` shows what stays live.
  `make bench-profile BENCH_PROFILE=memory` prints both views, and a smaller
  in-use heap is not evidence that churn fell.
- Two hazards are worth a finding on their own. A pooled object returned while
  still holding request data leaks it into the next request — this is a
  correctness finding, not a performance one. A grow-only buffer that once held
  a rare large payload is retained at that size forever, converting a GC win
  into a permanent heap increase.
- The claim under review is that allocation is the bottleneck, not that
  allocations went down. When the profile puts the cost elsewhere, reuse is
  added complexity with no budget behind it.

## Reject

- Reaching for a pool before a cheaper structural fix has been ruled out:
  materializing once instead of per item, avoiding a conversion in the loop, or
  sizing a slice at its known length usually removes the same allocations
  without reuse discipline to maintain.

## Validation Shape

Old-versus-new `-benchmem` through `make bench-baseline` / `make bench` /
`make bench-compare`, plus a case that does not keep the pool hot when the claim
is about reuse. For a live GC or heap claim, use the runtime metrics
`otelruntime` already exports rather than extrapolating from a benchmark.
