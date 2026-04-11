# Error, Context, And Cancellation Tests

## When To Load
Load this when tests touch wrapped errors, sentinel or typed error categories, context propagation, cancellation, deadlines, shutdown, fail-fast behavior, context values, or accidental replacement with `context.Background()`.

## Error Rules
- Use `errors.Is` for sentinel errors and context cancellation categories.
- Use `errors.As` for typed errors or structured error details.
- Compare raw strings only when exact text is part of public behavior.
- If an error should include human context and preserve a cause, assert both the category and a minimal context fragment.
- Do not require internal error wording at process boundaries unless the contract exposes it.

## Context Rules
- Pass parent context through; do not hide a lost parent behind top-level cancellation checks.
- Verify `context.Canceled` and `context.DeadlineExceeded` remain recognizable.
- For derived contexts, call the cancel function and prove bounded exit when work blocks.
- Use context values in tests only when the approved behavior depends on propagation of request-scoped data.
- Use `context.WithoutCancel` only when the approved lifecycle behavior requires parent cancellation to be ignored.

## Good Example
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

## Bad Example
```go
func TestRepositoryCreateWrapsQueryError(t *testing.T) {
	_, err := repo.Create(context.Background(), "payload")
	if err.Error() != "create record: write failed" {
		t.Fatal("wrong error")
	}
}
```

Why it is bad: it can panic on nil and overfits string formatting while missing error inspectability.

## Good Context Propagation Example
```go
func TestHelperReceivesParentContextDeadlineAndValue(t *testing.T) {
	type contextKey struct{}
	key := contextKey{}

	parent, cancel := context.WithTimeout(context.WithValue(context.Background(), key, "request-1"), time.Minute)
	defer cancel()

	var gotDeadline bool
	var gotValue any
	repo := fakeRepo{
		call: func(ctx context.Context) error {
			_, gotDeadline = ctx.Deadline()
			gotValue = ctx.Value(key)
			return ctx.Err()
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

This catches accidental `context.Background()` replacement inside the call chain.

## Assertion Patterns
- `if !errors.Is(err, context.Canceled) { ... }`
- `if !errors.Is(err, context.DeadlineExceeded) { ... }`
- `var target *SomeError; if !errors.As(err, &target) { ... }`
- For shutdown: assert event order, timeout/deadline presence, ignored or honored parent cancellation per approved behavior, and wrapped root cause.
- For fail-fast: assert sibling cancellation and original fatal cause preservation.

## Deterministic Coordination Patterns
- Use an already-canceled context to prove early cancellation behavior.
- Use an expired deadline context to prove deadline category without waiting.
- Use a fake dependency that records the context it receives.
- Use `readyToBlock` channels before canceling a running goroutine.
- Use `synctest.Test` for timeout behavior when fake-time support fits the code under test.

## Repository-Local Cues
- `cmd/service/internal/bootstrap/main_shutdown_test.go` checks shutdown ordering, deadline presence, ignored parent cancellation, and wrapped shutdown failures.
- `cmd/service/internal/bootstrap/startup_dependencies_additional_test.go` checks context cancellation and deadline categories.
- `internal/config/config_test.go` checks wrapped load and parse categories with `errors.Is`.

## Exa Source Links
- [Go errors package](https://pkg.go.dev/errors)
- [Go context package](https://pkg.go.dev/context)
- [testing/synctest package](https://pkg.go.dev/testing/synctest)
- [Go race detector article](https://go.dev/doc/articles/race_detector.html)

