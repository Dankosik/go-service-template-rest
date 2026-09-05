# HTTP Resilience And Synchronization

Load for outbound HTTP deadlines, retry/rate pressure, polling, or paginated
synchronization after pinning the provider contract.

Carry one monotonic operation deadline through DNS/connect/TLS, request write,
headers/body, authentication, limiter wait, backoff, and parsing. Bound response
bytes, decompression, redirects, and idle behavior; each attempt receives only
remaining budget. Cancellation reaches I/O and body handling.

Retry only when the provider contract and stable operation identity make the
same immutable intent safe, the observed failure is eligible, and useful budget
remains. A side-effecting response loss or `5xx` may be ambiguous. `429` proves
rate pressure, not retry safety. Bound attempts and elapsed time, jitter delays,
honor applicable `Retry-After` inside the deadline, and limit concurrency at the
provider's quota scope. One shared retry budget prevents retries starving first
attempts.

Poll a documented status endpoint, never the original write. For pagination,
treat cursors as opaque and retain version, filters, stable order, account,
cursor/snapshot, and watermark. Apply items idempotently and advance the durable
checkpoint only after accepted page effects and permanent-failure records are
durable. Mutable offset pagination needs overlap/deduplication or a documented
snapshot; define deletion discovery and restart after cursor expiry.

Proof covers deadline/cancel at each blocking phase, connection loss before and
after possible send, throttle delays within/outside budget, shared limiter,
retryable then success, permanent zero-retry, crash on both sides of checkpoint,
duplicate/invalid item, expired cursor, deletion, and overlapping restart.
