# Cache Value

Load when deciding whether a cache should exist.

Pin the user-visible operation, target, correctness/isolation invariants,
environment, representative workload, time window, and cache state. Establish a
comparable baseline for end-to-end latency/throughput/errors and the origin's
request rate, concurrency, latency, and limiting resource. Count only repeated
work the proposed cache can remove.

Do not subtract independently reported percentiles as though they describe one
trace, and do not call a percentile a maximum. Require a measured
counterfactual or joint trace evidence for removable tail latency. If the
avoidable work cannot repay new key, freshness, invalidation, outage, and
operational surfaces, return no-cache. If evidence is unavailable, return a
bounded measurement plan without a performance claim.

Reopen when the workload, target, origin cost, or protected invariant changes
enough to reverse the value decision.
