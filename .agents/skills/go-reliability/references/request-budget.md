# Request Budget

## Load When

Load when a request path adds or changes a dependency call, pool acquire, query,
poll, or wait.

## Decide

`RequestTimeout` already installs the enclosing request deadline and maps its
expiry to 504 while the write timeout can still carry that response. Add a
smaller sub-budget only when the dependency must fail early enough to leave time
for remaining work. A handler observing `ctx.Err()` returns; it does not invent
another status.

PostgreSQL pool acquisition currently waits within the caller context. The
template provides no separate acquire timeout or saturation error. A live
caller waiting for the pool is therefore bounded by its enclosing operation or
request deadline; do not map an arbitrary acquire failure to 503.

Add a dependency-specific admission or acquire sub-budget only when measured
pool contention shows that waiting consumes unrelated request capacity. The
mechanism must cover every relevant repository path and preserve the difference
between parent deadline expiry and local saturation.

Cancellation alone may not bound the server during network failure: pgx sends a
separate cancel request, so server-side statement and idle-transaction bounds
remain part of the contract. A new dependency similarly needs a transport/server
bound when client cancellation can be lost.

Configured linters catch many missing contexts and cancel funcs, but a handler
can still root `context.Background()` instead of `r.Context()`. Reject local
timeouts at or above the enclosing budget and `http.TimeoutHandler`, which can
return while its handler goroutine continues.

## Prove

Assert the caller deadline reaches the dependency, sub-budget expiry retains the
correct status identity, and work actually stops; timer existence alone is not
proof.
