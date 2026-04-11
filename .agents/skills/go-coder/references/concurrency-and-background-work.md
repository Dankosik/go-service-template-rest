# Concurrency And Background Work

## When To Load
Load this when implementation work starts goroutines, uses channels, adds worker pools, changes shutdown, adds timers/tickers, touches shared state, or makes request-scoped work asynchronous.

## Good/Bad Examples

Bad: unbounded goroutines can block forever on early return.

```go
func ProcessAll(ctx context.Context, jobs []Job) error {
	results := make(chan error)
	for _, job := range jobs {
		go func() {
			results <- process(ctx, job)
		}()
	}
	for range jobs {
		if err := <-results; err != nil {
			return err
		}
	}
	return nil
}
```

Good: keep lifecycle, cancellation, buffering, and bounded concurrency visible. Use `WaitGroup.Go` only when the module's Go version supports it.

```go
func ProcessAll(ctx context.Context, jobs []Job, limit int) error {
	if limit < 1 {
		limit = 1
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, limit)
	errs := make(chan error, len(jobs))

	var wg sync.WaitGroup
	for _, job := range jobs {
		job := job
		wg.Go(func() {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			if err := process(ctx, job); err != nil {
				cancel()
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			return err
		}
	}
	return ctx.Err()
}
```

Good when an approved dependency already exists: `errgroup` owns cancellation and first-error collection more directly.

```go
func ProcessAll(ctx context.Context, jobs []Job, limit int) error {
	if limit < 1 {
		limit = 1
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)
	for _, job := range jobs {
		job := job
		g.Go(func() error {
			return process(ctx, job)
		})
	}
	return g.Wait()
}
```

Bad: receiver closes a channel it does not own.

```go
func consume(ch <-chan Event) {
	defer close(ch)
	for event := range ch {
		handle(event)
	}
}
```

Good: the sender that owns the channel closes it.

```go
func produce(ctx context.Context, ch chan<- Event, events []Event) {
	defer close(ch)
	for _, event := range events {
		select {
		case ch <- event:
		case <-ctx.Done():
			return
		}
	}
}
```

Bad: sleeping to "wait" for background work.

```go
go worker.Run()
time.Sleep(100 * time.Millisecond)
```

Good: expose a readiness or completion signal.

```go
ready := make(chan struct{})
go worker.Run(ready)

select {
case <-ready:
case <-ctx.Done():
	return ctx.Err()
}
```

## Common False Simplifications
- Starting a goroutine inside a function just to make synchronous work look faster. Callers can add concurrency more easily than remove hidden concurrency.
- Using unbounded fan-out for I/O or CPU-heavy work.
- Returning on first error without proving other goroutines cannot block on send, receive, locks, or semaphores.
- Closing a channel from the receiver side.
- Ignoring `Ticker.Stop`, `Timer.Stop`, derived context cancellation, or worker shutdown paths.
- Replacing synchronization with sleeps in tests.
- Assuming the race detector proves absence of races on code paths tests do not execute.

## Validation Or Test Patterns
- Run `go test -race` for changed concurrent code.
- Run targeted repeated tests, for example `go test ./pkg/foo -run TestWorkerShutdown -count=100`.
- Use canceled contexts and blocked fake dependencies to prove goroutines exit.
- Prefer channels, hooks, or fake clocks over sleeps in tests.
- If the repository already uses a leak checker such as `goleak`, add or update the relevant leak proof.
- For worker pools, test the concurrency limit and early-error cancellation path.

## Source Links Gathered Through Exa
- [sync package](https://pkg.go.dev/sync)
- [context package](https://pkg.go.dev/context)
- [Go memory model](https://go.dev/ref/mem)
- [Data Race Detector](https://go.dev/doc/articles/race_detector)
- [Go Concurrency Patterns: Context](https://go.dev/blog/context)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Go 1.26 release notes](https://go.dev/doc/go1.26)
- [golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup)
