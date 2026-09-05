# Cache Runtime

Load when cache layer, concurrent fill, hot key, outage, or recovery can change
the decision.

Use the narrowest sharing boundary that removes the measured work: request,
process, distributed cache, then HTTP/CDN. Each extra layer must remove a
separate measured cost. Request/process caches inherit process memory and deploy
cold-start limits; distributed caches add network latency, partitions,
eviction, and ambiguous outcomes; HTTP/CDN caches require correct audience,
`Cache-Control`, validators, `Vary`, key, purge, and propagation semantics.

On miss, bound origin concurrency, queueing, waiters, and fill deadline. Collapse
only equivalent fills. Caller cancellation ends that caller's wait, not every
waiter's fill. Publish only a current authority generation. Cache authoritative
not-found separately; never turn timeout/dependency failure into a negative hit.

Budget cache calls inside the request deadline. During timeout, eviction, hot
key, mass expiry, cold fleet, or partition, keep fallback demand within origin
capacity. Fail open only when correctness and headroom permit; otherwise bypass,
serve explicitly safe stale, or fail closed. Recovery ramps demand; broad flush
is a cold-load event, not a harmless reset.
