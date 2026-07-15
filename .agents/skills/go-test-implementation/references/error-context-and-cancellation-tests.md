# Error, Context, And Cancellation Tests

## Behavior Change Thesis
When loaded for wrapped errors, sentinel or typed categories, context propagation, cancellation, or deadlines, this file makes the model test inspectable error and context contracts instead of likely mistake: raw string comparison, nil-only checks, swallowed cancellation, or accidental `context.Background()` replacement.

## When To Load
Load this when tests touch wrapped errors, sentinel or typed error categories, context propagation, cancellation categories, deadlines, shutdown error shape, fail-fast behavior, context values, or suspected parent-context loss. If goroutine scheduling or timer determinism is the hard part, load `deterministic-concurrency-and-time-tests.md` instead.

## Decision Rubric
- Use `errors.Is` for sentinel errors and cancellation categories.
- Use `errors.As` for typed errors or structured causes.
- Compare raw strings only when exact text is public behavior.
- When an error should include human context and preserve a cause, assert both the category and a minimal context fragment.
- At process or HTTP boundaries, assert the stable external category and behavior; do not leak private internal wording into the test unless the contract exposes it.
- Pass parent context through fakes that record deadline, value, cancellation, or cause when propagation is the obligation.
- For derived contexts, call the cancel function and prove bounded exit only if the code can block.
- Use `context.WithoutCancel` only when approved lifecycle behavior requires parent cancellation to be ignored.

## Imitate
```go
func TestRepositoryCreateWrapsQueryError(t *testing.T) {
	sentinel := errors.New("write failed")
	repo := newRepositoryWithQuerier(fakeQuerier{
		create: func(context.Context, string) (Record, error) {
			return Record{}, sentinel
		},
	})

	_, err := repo.Create(context.Background(), "payload")
	if err == nil {
		t.Fatal("Create() error = nil, want non-nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("Create() error = %v, want wrapped %v", err, sentinel)
	}
	if !strings.Contains(err.Error(), "create record") {
		t.Fatalf("Create() error = %q, want operation context", err.Error())
	}
}
```

Copy the shape: preserve the durable cause and a minimal operation clue without pinning exact formatting.

```go
func TestHelperReceivesParentContextDeadlineAndValue(t *testing.T) {
	type contextKey struct{}
	key := contextKey{}

	parent, cancel := context.WithTimeout(
		context.WithValue(context.Background(), key, "request-1"),
		time.Minute,
	)
	defer cancel()

	var gotDeadline bool
	var gotValue any
	repo := fakeRepo{
		call: func(ctx context.Context) error {
			_, gotDeadline = ctx.Deadline()
			gotValue = ctx.Value(key)
			return nil
		},
	}

	if err := runWithRepo(parent, repo); err != nil {
		t.Fatalf("runWithRepo() error = %v, want nil", err)
	}
	if !gotDeadline {
		t.Fatal("repository context has no deadline")
	}
	if gotValue != "request-1" {
		t.Fatalf("repository context value = %v, want request-1", gotValue)
	}
}
```

Copy the shape: the fake catches accidental replacement with `context.Background()` inside the call chain.

## Reject
```go
func TestRepositoryCreateWrapsQueryError(t *testing.T) {
	_, err := repo.Create(context.Background(), "payload")
	if err.Error() != "create record: write failed" {
		t.Fatal("wrong error")
	}
}
```

Reject because it can panic on nil, overfits private formatting, and misses error inspectability.

```go
func TestWorkerCancel(t *testing.T) {
	err := runWorker(context.Background(), fakeSource{})
	if err != nil {
		t.Fatal(err)
	}
}
```

Reject because it does not pass or trigger a canceled context, so it cannot prove cancellation behavior.

## Agent Traps
- Comparing `err == context.Canceled` when wrapping is allowed by contract.
- Testing only the top-level function's pre-canceled path while the real bug is lost cancellation in a downstream dependency.
- Requiring exact message text for internal errors just because it is currently convenient.
- Treating `context.WithTimeout` duration expiry as a deterministic test clock. Use already-expired contexts when possible.
- Forgetting that process shutdown tests often need both event order and error category.

## Validation Shape
- Focused test command for the named error/context scenario.
- Add race-aware validation only when the cancellation path includes goroutines or shared state.
- When context propagation is tested through a fake, package-level tests should still compile and exercise nearby real call sites.
