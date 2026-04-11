# Transaction Boundary Review

## Behavior Change Thesis
When loaded for changed transaction scope or cache work around commit, this file makes the model choose the smallest atomic DB boundary and post-commit cache decision instead of likely mistakes such as per-statement fixes, Redis inside a transaction, or inventing an outbox design from review.

## When To Load
Load this reference when a Go diff starts, moves, retries, or removes a SQL transaction; splits dependent DB writes; mixes DB writes with cache updates; changes isolation options; or adds rollback/commit handling.

Keep findings local: prove why the changed operation must be atomic or why the transaction boundary is now too wide. Escalate schema ownership, saga/outbox design, API idempotency semantics, or a new retry policy instead of solving those here.

## Decision Rubric
- Dependent writes that must commit together are executed on `db` outside one `Tx`.
- A transaction is started with `Begin` instead of `BeginTx(ctx, opts)` in a request path.
- `Rollback` is missing on error paths or ignored before early return.
- `Commit` errors are dropped, masked, or followed by a success response.
- The transaction includes Redis, HTTP, queue publish, or other network calls before `Commit`.
- Cache invalidation happens before the DB commit and can publish data that later rolls back.
- Retry wraps only one statement inside a multi-statement transaction.
- Isolation level changes are made without a local invariant that needs them.

## Bad Example: Partial Commit Risk

```go
func (s *Store) CompleteOrder(ctx context.Context, orderID string) error {
	if _, err := s.db.ExecContext(ctx,
		`update orders set status = 'complete' where id = $1`, orderID,
	); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx,
		`insert into order_events(order_id, kind) values ($1, 'completed')`, orderID,
	); err != nil {
		return err
	}

	return s.cache.Del(ctx, "order:"+orderID).Err()
}
```

Review finding shape:

```text
[high] [go-db-cache-review] store/orders.go:88
Issue: The changed completion path updates order state and writes the completion event in separate implicit transactions, then invalidates cache afterward.
Impact: If the event insert fails after the update commits, readers can observe a completed order without the event history this path relies on.
Suggested fix: Run the dependent DB writes in one BeginTx block, commit once, then invalidate the exact cache key after commit if the existing cache contract allows best-effort invalidation.
Reference: database/sql requires transactions to end with Commit or Rollback; PostgreSQL isolation governs what concurrent transactions can observe.
```

## Good Example: One Local Transaction, Cache After Commit

```go
func (s *Store) CompleteOrder(ctx context.Context, orderID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`update orders set status = 'complete' where id = $1`, orderID,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`insert into order_events(order_id, kind) values ($1, 'completed')`, orderID,
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := s.cache.Del(ctx, "order:"+orderID).Err(); err != nil {
		return fmt.Errorf("invalidate order cache: %w", err)
	}
	return nil
}
```

This fix is local only if the existing contract says cache invalidation failure should fail the operation. If the contract says cache invalidation is best-effort or asynchronous, preserve that policy and ask for the smallest aligned change.

## Bad Example: Transaction Stretched Across Redis

```go
func (s *Store) RenameUser(ctx context.Context, id, name string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.cache.Del(ctx, "user:"+id).Err(); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `update users set name = $1 where id = $2`, name, id); err != nil {
		return err
	}
	return tx.Commit()
}
```

## Good Example: DB Atomicity First, Cache Boundary Afterward

```go
func (s *Store) RenameUser(ctx context.Context, id, name string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `update users set name = $1 where id = $2`, name, id); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return s.cache.Del(ctx, "user:"+id).Err()
}
```

The review finding should be precise: avoid holding DB locks while doing Redis I/O and avoid invalidating data that may roll back.

## Agent Traps
- Do not move Redis, HTTP, queue publish, or cache fill work inside a transaction just to make code "ordered"; that stretches locks across external I/O.
- Do not treat `defer tx.Rollback()` after `BeginTx` as a bug when the code checks `Commit`; rollback is normally a no-op after a successful commit.
- Do not invent outbox, saga, distributed transaction, or retry policy as the suggested fix unless the existing design already provides it.
- Do not report success after a `Commit` error. The safe review point is an unknown-state risk that needs returned or recorded failure handling.

## Smallest Safe Fix
- Use `db.BeginTx(ctx, opts)` when a request context exists.
- Add `defer tx.Rollback()` immediately after a successful `BeginTx`; let it no-op after `Commit`.
- Keep only DB statements needed for one invariant inside the transaction.
- Move cache invalidation after successful commit when the current contract allows it.
- Return or record `Commit` errors as unknown-state risks; never report success after a failed commit.
- If the fix needs outbox, saga, new idempotency keys, or API error semantics, escalate rather than inventing it in a review finding.

## Validation Shape
- Add a test that makes the second DB statement fail and verifies the first write is rolled back.
- Add a test that makes `Commit` fail and verifies no success response is returned.
- Add a fake cache that blocks or fails to prove Redis calls are not inside the DB transaction.
- Add a concurrency/integration case for the changed isolation assumption only when the invariant depends on it.
