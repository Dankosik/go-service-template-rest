# Resource Lifetime, I/O, And Transactions

## Behavior Change Thesis
When loaded for resource or transaction-lifetime pressure, this file makes the model keep acquisition, cleanup, terminal errors, and transaction scope explicit instead of hiding ownership in helpers, leaking resources, or leaving half-checked database/I/O paths.

## When To Load
Load this when work touches readers, writers, response bodies, files, `Rows`, scanners, statements, dedicated SQL connections, transactions, locks, timers, tickers, derived contexts, or cleanup helper extraction.

## Decision Rubric
- Keep acquire, use, and release in one obvious scope unless ownership is explicitly transferred.
- Put `defer` next to acquisition, but avoid `defer` inside long-running loops when cleanup would be delayed.
- Check terminal error surfaces: `rows.Err`, scanner errors, writer close errors when terminal, and `Commit` errors.
- Use context-aware calls such as `QueryContext`, `ExecContext`, and `BeginTx`.
- Keep slow network calls, queue publishes, and unrelated side effects outside transactions unless the approved design says otherwise.
- Bound reads at trust boundaries before expensive work or side effects.

## Imitate
Close rows and surface scan and cursor errors.

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

Use `database/sql` transaction ownership instead of raw SQL transaction strings.

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
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create order %s: %w", order.ID, err)
	}
	return nil
}
```

Use a helper only when it clarifies per-iteration ownership.

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

## Reject
Reject cursor code that loses cleanup and terminal errors.

```go
rows, err := r.db.QueryContext(ctx, "select id, total from orders")
if err != nil {
	return nil, err
}
for rows.Next() {
	var order Order
	_ = rows.Scan(&order.ID, &order.Total)
	orders = append(orders, order)
}
return orders, nil
```

Reject raw transaction management and unrelated side effects inside the transaction window.

```go
_, _ = db.ExecContext(ctx, "BEGIN")
_ = publishToQueue(ctx, event)
_, err := db.ExecContext(ctx, "COMMIT")
```

Reject unbounded reads on untrusted bodies.

```go
body, err := io.ReadAll(r.Body)
```

## Agent Traps
- Moving `Close`, `Rollback`, `Unlock`, or `cancel` into a helper that obscures who owns the resource.
- Using `defer` inside loops that can hold many resources until the outer function returns.
- Converting transaction logic into a callback helper before transaction scope, retry, and side-effect rules are stable.
- Ignoring `rows.Err`, scanner errors, writer close errors, or `Commit` errors.
- Treating rollback errors after a successful commit as a new failure.
- Adding `io.ReadAll` to simplify parsing without a size bound at a trust boundary.

## Validation Shape
- Use fake `io.Closer`, `httptest`, or repository fakes to prove close behavior when ownership changes.
- Add repository tests for scan and iteration failures when cursor handling changed.
- Test transaction failure points when order changes: begin, write, later write, commit.
- Use canceled contexts to prove caller cancellation reaches `QueryContext`, `ExecContext`, or `BeginTx`.
- Run `go test -race` when cleanup, locks, timers, or shared state changed.
