# Timeout, Deadline, And Cancellation Review

## When To Load
Load this reference when a Go diff touches request-scoped work, outbound HTTP/RPC calls, DB calls, sleeps, polling loops, long-running handlers, background work spawned from handlers, or code that adds `context.Background()`, `context.TODO()`, `context.WithTimeout`, `context.WithDeadline`, or `context.WithoutCancel`.

Keep the finding local: ask for the changed operation to preserve caller cancellation and have a bounded wait. Hand off global timeout-budget design to `go-reliability-spec`, DB cleanup depth to `go-db-cache-review`, and goroutine lifecycle depth to `go-concurrency-review`.

## Review Smells
- A request path replaces `r.Context()` or the caller `ctx` with `context.Background()`.
- A new timeout is derived from a root context instead of the inbound parent.
- `context.WithTimeout` or `context.WithDeadline` is used without `cancel()`.
- A loop uses `time.Sleep`, `time.After`, channel receive, mutex wait, or network I/O without a `ctx.Done()` escape.
- `http.NewRequest` is used where `http.NewRequestWithContext` is needed.
- DB work uses non-context methods or a context detached from the request.
- Cancellation errors are wrapped or logged as internal failures without preserving `context.Canceled` or `context.DeadlineExceeded` semantics.
- `context.WithoutCancel` is used on request-path work without an explicit, shorter replacement deadline.

## Failure Modes
- Client disconnects or deadline expiry do not stop downstream work, tying up DB connections, workers, or HTTP sockets.
- A child timeout outlives the caller and continues work after the caller has already abandoned the request.
- Missing `cancel()` leaks timers and child contexts until the parent is canceled.
- A blackholed dependency causes handler pileup and then overload.
- A retry or fallback loop ignores cancellation and turns a small dependency stall into resource exhaustion.

## Review Examples

Bad: the handler drops the request context before a DB call and outbound HTTP call.

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

Review finding shape:

```text
[high] [go-reliability-review] internal/http/jobs.go:42
Issue: The changed path derives its timeout from context.Background and builds an outbound request without the caller context.
Impact: If the client disconnects or the inbound deadline expires, the DB query and worker call can continue consuming scarce downstream resources.
Suggested fix: Derive the timeout from r.Context, pass that context to DB and HTTP work, and preserve cancellation errors at the handler boundary.
Reference: Go context and database cancellation docs.
```

Good: derive a bounded child from the caller and use context-aware APIs.

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

Do not prescribe `2*time.Second`; ask for the repository's existing budget source when one exists. The reliability finding is the detached or unbounded wait.

## Smallest Safe Fix
- Replace request-path `context.Background()` with the caller's `ctx` or `r.Context()`.
- Derive operation deadlines from the caller context and `defer cancel()`.
- Use context-aware APIs such as `QueryContext`, `ExecContext`, `BeginTx`, and `http.NewRequestWithContext`.
- Add `select { case <-ctx.Done(): return ctx.Err() }` around changed blocking loops or sends.
- Preserve cancellation classification with `errors.Is(err, context.Canceled)` or `errors.Is(err, context.DeadlineExceeded)` where the boundary maps errors.
- If work must survive the request, require an explicit owner, explicit deadline, and handoff to concurrency or async-review lanes.

## Validation Commands
- `go test ./... -run 'Test.*(Context|Cancel|Canceled|Cancellation|Timeout|Deadline)'`
- `go test ./... -run 'Test.*(ClientDisconnect|RequestAbort|SlowDependency)'`
- `go test -race ./...` when the change starts or cancels goroutines.
- `go vet ./...` to catch some missing `CancelFunc` paths.

If the repo has integration tests, add a canceled-context or short-timeout case for the changed DB/HTTP path rather than relying on a happy-path test.

## Exa Source Links
- Go `context` package documentation: https://pkg.go.dev/context
- Go database cancellation guide: https://go.dev/doc/database/cancel-operations
- Go `net/http` package documentation, including request contexts and server behavior: https://pkg.go.dev/net/http
- Google SRE cascading-failure chapter, blackholed dependency and missing deadline failure modes: https://sre.google/sre-book/addressing-cascading-failures/

