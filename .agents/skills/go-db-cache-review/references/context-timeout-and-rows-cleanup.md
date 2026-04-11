# Context Timeout And Rows Cleanup

## Behavior Change Thesis
When loaded for dropped cancellation, unbounded DB/cache calls, or resource lifecycle leaks, this file makes the model choose caller-derived deadlines and explicit cleanup instead of likely mistakes such as `context.Background`, arbitrary timeout budgets, or assuming `QueryContext` alone returns connections safely.

## When To Load
Load this reference when lifecycle and cancellation are the primary symptoms: DB/cache calls in request paths drop caller context, use `context.Background()` or `context.TODO()`, add local query deadlines, change `Rows` iteration, prepare statements, reserve connections, or alter transaction cleanup.

If query construction, dynamic SQL, or round-trip shape is the primary symptom, prefer `sql-query-and-resource-safety-review.md`. Keep findings local: ask for context propagation, bounded blocking calls, and cleanup in the changed code. Escalate global timeout budgets, retry policy, or overload behavior to reliability design/review.

## Decision Rubric
- Request path drops `ctx` and creates `context.Background()` before a DB or cache call.
- Code creates `context.WithTimeout` but does not call `cancel`.
- Code uses context-aware DB/cache methods but with an unbounded parent context on critical paths.
- `rows.Close()` is missing or placed after code that can return early.
- `rows.Err()` is not checked after iteration.
- `QueryContext` returns `Rows`, but the code only wants one row.
- `Stmt` or `Conn` is created/reserved and never closed.
- A transaction is left open if `Commit` is skipped or context cancellation occurs.

## Bad Example: Dropped Request Cancellation

```go
func (s *Store) ActiveUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(context.Background(), `
		select id, email from users where active = true`)
	if err != nil {
		return nil, err
	}

	var out []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email); err != nil {
			return nil, err
		}
		out = append(out, user)
	}
	return out, nil
}
```

Review finding shape:

```text
[medium] [go-db-cache-review] store/users.go:55
Issue: The changed query replaces the caller's request context with context.Background and does not close/check Rows.
Impact: A canceled request can keep the DB query and cursor alive, tying up a pooled connection and hiding iteration errors.
Suggested fix: Derive any timeout from the caller context, defer cancel, close Rows after a successful query, and check rows.Err after iteration.
Reference: Go context cancellation and database/sql Rows cleanup guidance.
```

## Good Example: Derived Deadline And Cursor Cleanup

```go
func (s *Store) ActiveUsers(ctx context.Context) ([]User, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(queryCtx, `
		select id, email from users where active = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email); err != nil {
			return nil, err
		}
		out = append(out, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
```

Do not mandate `2*time.Second`; use the repository's existing timeout source when one exists. The review finding is that blocking I/O needs a bounded, propagated context and cursor cleanup.

## Bad Example: Prepared Statement Leaked

```go
func (s *Store) EmailByID(ctx context.Context, id string) (string, error) {
	stmt, err := s.db.PrepareContext(ctx, `select email from users where id = $1`)
	if err != nil {
		return "", err
	}

	var email string
	if err := stmt.QueryRowContext(ctx, id).Scan(&email); err != nil {
		return "", err
	}
	return email, nil
}
```

## Good Example: Close Or Avoid Per-Request Prepare

```go
func (s *Store) EmailByID(ctx context.Context, id string) (string, error) {
	var email string
	err := s.db.QueryRowContext(ctx,
		`select email from users where id = $1`, id,
	).Scan(&email)
	if err != nil {
		return "", err
	}
	return email, nil
}
```

If the statement is intentionally reused, require the owning lifecycle to close it. If it is per request, the smaller fix is often to avoid preparing at all.

## Agent Traps
- Do not invent a magic timeout duration. Use an existing repository budget, caller deadline, or ask for the local operation to derive a deadline where the package already owns one.
- Do not replace caller cancellation with `context.Background()` for "cleanup" in a request path unless the code is deliberately detaching an owned async operation.
- Do not say `PrepareContext` bounds statement execution; the prepare context covers prepare work, while each query still needs its own execution context.
- Do not load both this file and the SQL query reference just to mention `Rows.Close`; pick the one that changes the finding.

## Smallest Safe Fix
- Pass the caller `ctx` to DB/cache methods instead of `context.Background()`.
- Derive operation-specific timeouts from the caller context and `defer cancel()`.
- Add `defer rows.Close()` after successful `QueryContext`.
- Check `rows.Err()` after iteration.
- Close `Stmt` and `Conn` when created in the changed path.
- Use `BeginTx(ctx, ...)` and rollback on all non-commit paths.

## Validation Shape
- Add a test with an already-canceled context and assert DB/cache work returns promptly.
- Add a fake `Rows` or integration test that surfaces an iteration error and assert it is returned.
- Run `go test -race` for changed cache wrappers that share state.
- Use a short PostgreSQL `statement_timeout` or test query such as `pg_sleep` in integration tests only when the repository already supports DB-backed tests.
