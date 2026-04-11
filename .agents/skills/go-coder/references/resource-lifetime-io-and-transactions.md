# Resource Lifetime, I/O, And Transactions

## When To Load
Load this when work touches readers, writers, response bodies, files, rows, scanners, prepared statements, dedicated SQL connections, transactions, locks, timers, tickers, derived contexts, or cleanup helper extraction.

## Good/Bad Examples

Bad: rows are not closed and terminal cursor errors disappear.

```go
func (r *Repo) List(ctx context.Context) ([]Order, error) {
	rows, err := r.db.QueryContext(ctx, "select id, total from orders")
	if err != nil {
		return nil, err
	}

	var orders []Order
	for rows.Next() {
		var order Order
		_ = rows.Scan(&order.ID, &order.Total)
		orders = append(orders, order)
	}
	return orders, nil
}
```

Good: acquire, defer cleanup next to acquisition, and check terminal errors.

```go
func (r *Repo) List(ctx context.Context) ([]Order, error) {
	rows, err := r.db.QueryContext(ctx, "select id, total from orders")
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.Total); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}
	return orders, nil
}
```

Bad: transaction lifetime is spread across raw SQL and unrelated side effects.

```go
if _, err := db.ExecContext(ctx, "BEGIN"); err != nil {
	return err
}
if err := publishToQueue(ctx, event); err != nil {
	return err
}
_, err := db.ExecContext(ctx, "COMMIT")
return err
```

Good: use `database/sql` transaction APIs and keep transaction work narrow.

```go
func (r *Repo) Create(ctx context.Context, order Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create order: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, insertOrderSQL, order.ID, order.Total); err != nil {
		return fmt.Errorf("insert order %s: %w", order.ID, err)
	}
	if _, err := tx.ExecContext(ctx, insertAuditSQL, order.ID, "created"); err != nil {
		return fmt.Errorf("insert order audit %s: %w", order.ID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create order %s: %w", order.ID, err)
	}
	return nil
}
```

Bad: defers inside a long loop delay cleanup.

```go
for _, name := range names {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	process(f)
}
```

Good: make per-iteration ownership explicit.

```go
for _, name := range names {
	if err := processFile(name); err != nil {
		return err
	}
}

func processFile(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("open %s: %w", name, err)
	}
	defer f.Close()
	return process(f)
}
```

Bad: unbounded reads on request bodies.

```go
body, err := io.ReadAll(r.Body)
```

Good: bound reads at trust boundaries.

```go
const maxBody = 1 << 20
body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
```

## Common False Simplifications
- Moving `Close`, `Rollback`, `Unlock`, or `cancel` into a helper that hides who owns the resource.
- Using `defer` inside loops where many resources can remain open until the outer function returns.
- Converting transaction logic into a callback helper before transaction scope, retries, and side-effect rules are clear.
- Ignoring `rows.Err`, scanner errors, `Close` errors that are terminal for writers, or `Commit` errors.
- Performing network calls, queue publishes, or slow unrelated work inside a transaction without an approved design reason.
- Using `io.ReadAll` on untrusted input without a size bound.

## Validation Or Test Patterns
- Use tests with fake `io.Closer` or `httptest` responses to prove close behavior for new I/O ownership.
- Add a repository test that forces scan or iteration failure when cursor handling changed.
- Test transaction failure at each step when changing transaction order: begin, first write, later write, commit.
- Use canceled contexts to prove `QueryContext`, `ExecContext`, and `BeginTx` paths receive caller cancellation.
- Run `go test -race` when cleanup or locking changes can affect concurrency.

## Source Links Gathered Through Exa
- [database/sql package](https://pkg.go.dev/database/sql)
- [Executing transactions](https://go.dev/doc/database/execute-transactions)
- [Canceling in-progress database operations](https://go.dev/doc/database/cancel-operations)
- [Managing database connections](https://go.dev/doc/database/manage-connections)
- [io package](https://pkg.go.dev/io)
- [context package](https://pkg.go.dev/context)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
