# Reference Selector

State which evidence or correction choice the selected reference will change.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| A performance claim or blocker lacks a dominant narrower evidence question. | [performance-evidence-quality.md](performance-evidence-quality.md) | Write the exact missing proof or residual risk instead of accepting “faster” or demanding broad load tests reflexively. |
| Go benchmark or `benchstat` evidence is added or relied on. | [benchmark-and-benchstat-review.md](benchmark-and-benchstat-review.md) | Check workload, timer, `-benchmem`, repetition, and practical significance instead of trusting any table. |
| CPU, heap, allocs, goroutine, block, mutex, or live pprof artifact is central. | [pprof-and-profile-selection.md](pprof-and-profile-selection.md) | Match profile type and workload to the symptom. |
| Locks, channels, queues, fan-out/fan-in, or scheduler wait affects tail latency. | [trace-block-mutex-and-contention.md](trace-block-mutex-and-contention.md) | Ask for timeline/block/mutex evidence and smallest wait reduction instead of generic worker pools. |
| Loops, copies, serialization, batching, fan-out growth, or repeated transforms change. | [hot-path-cost-model.md](hot-path-cost-model.md) | Name the scaling dimension and structural fix instead of micro-optimization folklore. |
| DB/cache/query count, dependency calls, pagination, or I/O-in-loop amplifies the request path. | [db-cache-and-io-amplification.md](db-cache-and-io-amplification.md) | Quantify round trips and hand correctness ownership off instead of treating all DB/cache concerns as performance. |
| Allocation churn, GC, buffer reuse, retained backing arrays, or `sync.Pool` changes. | [allocation-gc-and-syncpool-review.md](allocation-gc-and-syncpool-review.md) | Require allocation/retention/reset evidence instead of pooling by default. |
| Retries, fallback, admission, queueing, or deadline behavior changes on a hot path. | [retry-overload-and-tail-latency.md](retry-overload-and-tail-latency.md) | Identify amplification and tail collapse while handing policy ownership to reliability. |
