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

Config validates PostgreSQL acquire/statement bounds against the request
timeout. `Pool.Acquire` distinguishes a live caller whose acquire sub-budget
expired (`ErrSaturated`, 503) from an already-spent caller budget (504). Preserve
that distinction in metrics and errors.

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
