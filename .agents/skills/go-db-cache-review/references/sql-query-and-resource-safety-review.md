# SQL Query And Resource Safety Review

## When To Load
Load this reference when a Go review touches `database/sql`, query construction, request-path SQL loops, dynamic sort/filter SQL, `QueryContext`, `QueryRowContext`, prepared statements, or result cursor cleanup.

Keep findings local: point at the changed query path, the observable DB/cache defect, and the smallest review-safe correction. Escalate schema ownership, query model changes, or API-visible filtering semantics instead of redesigning them here.

## Review Smell Patterns
- Changed code uses `db.Query` or `db.Exec` without a caller-provided `context.Context`.
- User input is concatenated into SQL values instead of passed as bind arguments.
- Dynamic identifiers, sort fields, or directions are interpolated without an allowlist.
- A handler loops over IDs and issues one query per item when one batched query would preserve behavior.
- `QueryContext` is used when `QueryRowContext` would avoid a cursor for a single-row lookup.
- Returned `*Rows` is not closed on all paths, or `rows.Err()` is never checked after iteration.
- Prepared statements are created per request and not closed, or preparation context is mistaken for execution context.

## Bad Example: Contextless Dynamic SQL And Leaked Rows

```go
func (s *Store) ListUsers(ctx context.Context, tenantID, sort string) ([]User, error) {
	query := "select id, email from users where tenant_id = '" + tenantID + "' order by " + sort
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}
```

Review finding shape:

```text
[high] [go-db-cache-review] store/users.go:42
Issue: The changed query concatenates tenant and sort input into SQL, runs without the request context, and leaks the returned Rows on early scan errors.
Impact: A bad tenant or sort value can alter the query, a canceled request can keep DB work running, and leaked cursors can pin connections under load.
Suggested fix: Bind tenantID as an argument, allowlist the sort token, call QueryContext, and close/check Rows.
Reference: Go database/sql QueryContext and Rows cleanup guidance.
```

## Good Example: Bound Values, Allowlisted Identifier, Cursor Cleanup

```go
var userSortColumns = map[string]string{
	"email":      "email",
	"created_at": "created_at",
}

func (s *Store) ListUsers(ctx context.Context, tenantID, sort string) ([]User, error) {
	sortColumn, ok := userSortColumns[sort]
	if !ok {
		return nil, fmt.Errorf("unsupported user sort %q", sort)
	}

	rows, err := s.db.QueryContext(ctx, `
		select id, email
		from users
		where tenant_id = $1
		order by `+sortColumn, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}
```

The local review point is not "rewrite sorting"; it is "dynamic identifiers need an allowlist, values need bind arguments, and the cursor needs cleanup."

## Bad Example: N+1 Lookup In A Changed Hot Path

```go
func (s *Store) LoadProfiles(ctx context.Context, ids []string) ([]Profile, error) {
	var out []Profile
	for _, id := range ids {
		var profile Profile
		err := s.db.QueryRowContext(ctx,
			`select id, display_name from profiles where id = $1`, id,
		).Scan(&profile.ID, &profile.DisplayName)
		if err != nil {
			return nil, err
		}
		out = append(out, profile)
	}
	return out, nil
}
```

## Good Example: Batch The Same Contract

```go
func (s *Store) LoadProfiles(ctx context.Context, ids []string) ([]Profile, error) {
	rows, err := s.db.QueryContext(ctx, `
		select id, display_name
		from profiles
		where id = any($1)`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Profile
	for rows.Next() {
		var profile Profile
		if err := rows.Scan(&profile.ID, &profile.DisplayName); err != nil {
			return nil, err
		}
		out = append(out, profile)
	}
	return out, rows.Err()
}
```

If the repository does not use `pq.Array`, adapt to its existing driver or query builder. The review finding should ask for one batched operation, not mandate a driver switch.

## Smallest Safe Fix
- Replace contextless `Query`, `Exec`, `Prepare`, or `Begin` with the context-aware form when the caller already has `ctx`.
- Bind data values as query arguments.
- Add an allowlist for dynamic identifiers, sort fields, or directions.
- Add `defer rows.Close()` immediately after a successful query and check `rows.Err()` after iteration.
- Replace `QueryContext` plus a one-row loop with `QueryRowContext` when the result contract is at most one row.
- Batch only when the changed path clearly introduced or worsened avoidable round trips; otherwise record the risk.

## Validation Ideas
- Unit-test allowlist rejection for unknown sort/filter identifiers.
- Add an integration test that cancels the request context before a long query and expects a context error or driver-specific cancellation error.
- Add a test that forces a scan/iteration error path and verifies the function returns the error.
- Use query-count instrumentation or a fake DB to prove per-item loops were removed when that is the finding.

## Source Links From Exa
- Go `database/sql` package: https://go.dev/pkg/database/sql
- Go SQLInterface usage examples: https://go.dev/wiki/SQLInterface
- Go canceling in-progress database operations: https://go.dev/doc/database/cancel-operations
- PostgreSQL transaction isolation reference: https://www.postgresql.org/docs/current/transaction-iso.html
