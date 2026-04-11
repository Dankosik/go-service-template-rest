# Errors, Context, And Boundary Mapping

## Behavior Change Thesis
When loaded for error, context, or transport-boundary pressure, this file makes the model preserve inspectable error identity and caller cancellation at the correct boundary instead of string-matching errors, returning status codes from repositories, or accidentally detaching request work.

## When To Load
Load this when implementation work touches error wrapping, sentinel or typed errors, context propagation, cancellation, HTTP/RPC status mapping, repository error translation, or log-and-return behavior.

## Decision Rubric
- Return domain-shaped errors from domain or repository layers; map them to transport status at the transport boundary.
- Preserve inspectable identity with `%w`, `errors.Is`, and Go-version-appropriate `errors.AsType` or `errors.As` when callers branch on the error.
- Do not wrap an internal error with `%w` if exposing it would become an API contract.
- Pass `context.Context` as the first argument through request-scoped work; detach only when the approved design says so.
- For approved detached work that must retain request values, consider `context.WithoutCancel` plus an explicit new timeout or lifecycle owner; do not use it to silently ignore cancellation.
- Treat `context.Canceled` and `context.DeadlineExceeded` as control-flow signals, not generic internal failures.
- Add operation context to returned errors; avoid logging and returning the same error at every layer.

## Imitate
Keep repository errors domain-shaped and map at the boundary.

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

Pass caller context through request work.

```go
func (s *Service) Sync(ctx context.Context, id OrderID) error {
	if err := s.client.Push(ctx, id); err != nil {
		return fmt.Errorf("push order %s: %w", id, err)
	}
	return nil
}
```

## Reject
Reject boundary leakage and string matching in repositories.

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

Reject accidental detachment.

```go
func (s *Service) Sync(_ context.Context, id OrderID) error {
	return s.client.Push(context.Background(), id)
}
```

## Agent Traps
- Replacing `errors.Is` with direct equality after adding `%w`.
- Matching exact error text in tests when the contract is identity, type, or status mapping.
- Wrapping every error with `%w` even when callers must not inspect the internal cause.
- Storing `context.Context` in long-lived structs instead of passing it through request operations.
- Swapping request context for `context.Background()` in approved detached work and accidentally dropping trace, tenant, or auth values.
- Mapping cancellations, deadlines, validation failures, conflicts, and missing resources to one generic internal error.

## Validation Shape
- Add boundary tests that assert status mapping through `errors.Is`, `errors.AsType`, or `errors.As`, not string equality.
- Add a wrapped typed-error case when typed inspection matters.
- Add cancellation or timeout tests when context propagation changed, and prove the repository/client receives the caller context.
- If a wrapper intentionally hides an internal cause, test that callers cannot match it.
