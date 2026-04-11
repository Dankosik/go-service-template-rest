# Timeout, Deadline, And Cancellation Review

## Behavior Change Thesis
When loaded for symptom `request-scoped work, outbound I/O, polling, or detached context changed`, this file makes the model report dropped caller cancellation and unbounded waits instead of likely mistake `accept context.Background, arbitrary local timeouts, or context-unaware APIs as normal reliability hardening`.

## When To Load
Load when a Go diff touches request-scoped work, outbound HTTP/RPC calls, DB calls, sleeps, polling loops, long-running handlers, background work spawned from handlers, or code that adds `context.Background()`, `context.TODO()`, `context.WithTimeout`, `context.WithDeadline`, or `context.WithoutCancel`.

Keep the finding local: ask for the changed operation to preserve caller cancellation and have a bounded wait. Hand off global timeout-budget design to `go-reliability-spec`, DB cleanup depth to `go-db-cache-review`, and goroutine lifecycle depth to `go-concurrency-review`.

## Decision Rubric
- Request path replaces `r.Context()` or the caller `ctx` with `context.Background()` or `context.TODO()`.
- Timeout or deadline is derived from a root context instead of the inbound parent.
- `context.WithTimeout` or `context.WithDeadline` is used without `cancel()`.
- `time.Sleep`, `time.After`, channel receive, mutex wait, polling, or network I/O has no `ctx.Done()` escape.
- Outbound HTTP uses `http.NewRequest` where `http.NewRequestWithContext` is needed.
- DB work uses non-context methods or a context detached from the request.
- Cancellation errors are wrapped or logged as internal failures without preserving `context.Canceled` or `context.DeadlineExceeded` semantics.
- `context.WithoutCancel` appears in request-path work without an explicit owner and shorter replacement deadline.

## Imitate

Bad finding shape to copy: the defect is not "missing timeout"; it is "child work outlives the caller."

```go
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rows, err := h.db.QueryContext(ctx, "select id from jobs")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	req, _ := http.NewRequest(http.MethodPost, h.workerURL, nil)
	resp, err := h.client.Do(req)
	_ = resp
	_ = err
}
```

```text
[high] [go-reliability-review] internal/http/jobs.go:42
Issue: The changed path derives its timeout from context.Background and builds an outbound request without the caller context.
Impact: If the client disconnects or the inbound deadline expires, the DB query and worker call can continue consuming scarce downstream resources.
Suggested fix: Derive the timeout from r.Context, pass that context to DB and HTTP work, and preserve cancellation errors at the handler boundary.
Reference: Go context and database cancellation docs.
```

Good correction shape: derive a bounded child from the caller and use context-aware APIs.

```go
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	rows, err := h.db.QueryContext(ctx, "select id from jobs")
	if err != nil {
		writeError(w, err)
		return
	}
	defer rows.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.workerURL, nil)
	if err != nil {
		writeError(w, err)
		return
	}
	resp, err := h.client.Do(req)
	_ = resp
	_ = err
}
```

Copy the review move: do not prescribe `2*time.Second`; ask for the repository's existing budget source when one exists.

## Reject

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()
return s.remote.Do(ctx, req)
```

Reject because the child deadline can outlive caller cancellation. The local fix is deriving from caller `ctx`, not merely adding a timer.

```go
for {
	time.Sleep(time.Second)
	if ready() {
		return nil
	}
}
```

Reject because dependency blackholes or shutdown can leave the loop running until the caller's budget is exhausted somewhere else, or forever if no caller budget exists.

```go
ctx := context.WithoutCancel(r.Context())
go h.rebuild(ctx, key)
```

Reject in request-path review unless the async work has an explicit owner, deadline, and lifecycle handoff; otherwise the request can create unbounded orphan work.

## Agent Traps
- Do not invent a magic timeout duration. Use an existing repository budget or ask for a caller-derived operation budget when the package owns one.
- Do not treat `context.WithTimeout(context.Background(), ...)` as equivalent to `context.WithTimeout(ctx, ...)` on request paths.
- Do not demand cancellation for intentionally detached durable work; require the owner, deadline, and hand off to async or concurrency review when that proof is non-local.
- Do not load this file just to mention DB `Rows` cleanup; prefer `go-db-cache-review` when cursor/resource cleanup is the primary defect.
- Do not turn cancellation classification into error-style nitpicks; only flag it when callers/operators lose a meaningful canceled/deadline signal.

## Validation Shape
- `go test ./... -run 'Test.*(Context|Cancel|Canceled|Cancellation|Timeout|Deadline)'`
- `go test ./... -run 'Test.*(ClientDisconnect|RequestAbort|SlowDependency)'`
- `go test -race ./...` when the change starts or cancels goroutines.
- `go vet ./...` to catch some missing `CancelFunc` paths.

If the repo has integration tests, add a canceled-context or short-timeout case for the changed DB/HTTP path rather than relying on a happy-path test.
