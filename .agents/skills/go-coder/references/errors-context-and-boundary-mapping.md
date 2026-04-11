# Errors, Context, And Boundary Mapping

## When To Load
Load this when implementation work touches error wrapping, sentinel or typed errors, context propagation, cancellation, HTTP or RPC status mapping, repository error translation, or log-and-return decisions.

## Good/Bad Examples

Bad: string matching and boundary behavior mixed into the repository.

```go
func (r *Repo) Find(ctx context.Context, id OrderID) (Order, int, error) {
	row := r.db.QueryRowContext(ctx, "select id, total from orders where id = $1", id)
	var order Order
	if err := row.Scan(&order.ID, &order.Total); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return Order{}, http.StatusNotFound, nil
		}
		return Order{}, http.StatusInternalServerError, err
	}
	return order, http.StatusOK, nil
}
```

Good: repository returns domain-shaped errors; the transport boundary maps them.

```go
var ErrOrderNotFound = errors.New("order not found")

func (r *Repo) Find(ctx context.Context, id OrderID) (Order, error) {
	row := r.db.QueryRowContext(ctx, "select id, total from orders where id = $1", id)

	var order Order
	if err := row.Scan(&order.ID, &order.Total); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Order{}, fmt.Errorf("find order %s: %w", id, ErrOrderNotFound)
		}
		return Order{}, fmt.Errorf("find order %s: %w", id, err)
	}
	return order, nil
}

func writeOrderError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.Canceled):
		return
	case errors.Is(err, ErrOrderNotFound):
		http.Error(w, "order not found", http.StatusNotFound)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
```

Bad: request-scoped code detaches work by accident.

```go
func (s *Service) Sync(_ context.Context, id OrderID) error {
	return s.client.Push(context.Background(), id)
}
```

Good: keep the caller's context flowing unless detachment is explicitly approved.

```go
func (s *Service) Sync(ctx context.Context, id OrderID) error {
	if err := s.client.Push(ctx, id); err != nil {
		return fmt.Errorf("push order %s: %w", id, err)
	}
	return nil
}
```

Bad: typed error checks only see the outer error.

```go
if pathErr, ok := err.(*fs.PathError); ok {
	return pathErr.Path
}
```

Good: inspect wrapped errors. In Go 1.26+, `errors.AsType` is concise for error types.

```go
if pathErr, ok := errors.AsType[*fs.PathError](err); ok {
	return pathErr.Path
}
```

## Common False Simplifications
- Replacing `errors.Is` with direct equality after adding `%w`.
- Matching error text in tests when the contract is error identity or type.
- Wrapping every error with `%w` even when exposing the wrapped error would accidentally become part of an API contract.
- Logging and returning the same error at every layer. Add context to the returned error; log once where the operation is handled or abandoned.
- Treating `context.Canceled` and `context.DeadlineExceeded` as generic internal errors at the transport boundary.
- Storing `context.Context` in long-lived structs instead of passing it as the first argument to request-scoped operations.
- Using context values for optional parameters instead of request-scoped data that crosses APIs.

## Validation Or Test Patterns
- Write boundary tests that assert status mapping through `errors.Is`, not through exact error strings.
- Add cancellation tests with a canceled or timed-out context and prove the repository/client receives that context.
- For typed errors, add a test where the typed error is wrapped once and still found by `errors.As` or `errors.AsType`.
- If a wrapper intentionally hides an internal error, test that callers cannot match it with `errors.Is`.
- Verify log changes by behavior when possible; avoid brittle tests on log text unless log format is part of the contract.

## Source Links Gathered Through Exa
- [errors package](https://pkg.go.dev/errors)
- [context package](https://pkg.go.dev/context)
- [database/sql package](https://pkg.go.dev/database/sql)
- [Go Concurrency Patterns: Context](https://go.dev/blog/context)
- [Contexts and structs](https://go.dev/blog/context-and-structs)
- [Canceling in-progress database operations](https://go.dev/doc/database/cancel-operations)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Go 1.26 release notes](https://go.dev/doc/go1.26)
