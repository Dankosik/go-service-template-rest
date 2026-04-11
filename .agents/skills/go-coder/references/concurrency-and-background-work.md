# Concurrency And Background Work

## Behavior Change Thesis
When loaded for goroutine, channel, worker, or shutdown pressure, this file makes the model make lifecycle, cancellation, bounds, and proof visible instead of hiding unbounded background work, blocking on early returns, or synchronizing tests with sleeps.

## When To Load
Load this when implementation work starts goroutines, uses channels, adds fan-out or worker pools, changes shutdown, adds timers/tickers, touches shared state, or makes request-scoped work asynchronous.

## Decision Rubric
- Do not add hidden concurrency just to make synchronous work look faster.
- Give every goroutine a lifecycle owner, cancellation path, and result/error path.
- Bound fan-out and queue growth; unbounded concurrency is a correctness risk, not just a performance risk.
- Prefer `errgroup.WithContext` when the dependency already exists and first-error cancellation is the desired contract.
- Close channels from the sender side, and only after all sends are done.
- Stop timers and tickers and cancel derived contexts when their lifetime ends.
- Preserve ordering if callers or tests depend on it.

## Imitate
Use `errgroup` when it already fits the task and dependency policy.

```go
func ProcessAll(ctx context.Context, jobs []Job, limit int) error {
	if limit < 1 {
		limit = 1
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)
	for _, job := range jobs {
		g.Go(func() error {
			return process(ctx, job)
		})
	}
	return g.Wait()
}
```

Make channel ownership explicit at the sender.

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

Expose readiness or completion rather than sleeping.

```go
ready := make(chan struct{})
go worker.Run(ready)

select {
case <-ready:
case <-ctx.Done():
	return ctx.Err()
}
```

## Reject
Reject unbounded goroutines that can block after early return.

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

Reject receiver-side channel close.

```go
func consume(ch <-chan Event) {
	defer close(ch)
	for event := range ch {
		handle(event)
	}
}
```

Reject sleep-as-synchronization.

```go
go worker.Run()
time.Sleep(100 * time.Millisecond)
```

## Agent Traps
- Returning on first error without proving other goroutines cannot block on sends, receives, locks, or semaphores.
- Assuming closure captures are safe when the code uses a reused mutable variable outside the range loop or a pre-Go 1.22 module.
- Treating a buffered channel as a complete lifecycle plan.
- Replacing test sleeps with a longer timeout rather than a readiness/completion signal.
- Assuming the race detector proves paths the tests never exercise.
- Using `sync.WaitGroup.Go` without checking the module Go version or handling the "function must not panic" contract.

## Validation Shape
- Run `go test -race` for changed concurrent code.
- Run targeted repeated tests for lifecycle-sensitive code, for example `go test ./pkg/foo -run TestWorkerShutdown -count=100`.
- Use canceled contexts and blocked fake dependencies to prove goroutines exit.
- Prefer channels, hooks, or fake clocks over sleeps in tests.
- If the repository already uses a leak checker such as `goleak`, add or update the relevant leak proof.
- For worker pools, test the concurrency limit and early-error cancellation path.
