# API, Data, And Cache Test Patterns

## When To Load
Load this when tests touch HTTP/API contracts, strict request parsing, idempotency, async operation resources, SQL/repositories, migrations, tenant isolation, caching, stale-data prevention, fallback/degradation, or integration with Postgres/testcontainers.

## API Patterns
- Use `httptest.NewRequest` and `httptest.NewRecorder` for handler-level tests.
- Assert status, content type, headers, and response body only to the exactness approved by the contract or existing code.
- Cover malformed input, unknown fields, trailing JSON, missing required fields, unsupported media type, size limits, idempotency, retry classification, and request ID behavior when relevant.
- Keep semantic service tests separate from transport mapping when exact status/header/body semantics are not approved.

## Data And Cache Patterns
- Use fakes for repository mapper and error wrapping behavior.
- Use integration tests for real SQL, migration, transaction, lock, query ordering, or driver behavior.
- Test `QueryRow`/`Scan`, `Rows.Next`, `Rows.Err`, and `Rows.Close` behavior at the layer where it matters.
- For cache behavior, cover hit, miss, stale, expired, bypass, fallback, corruption, tenant isolation, and stampede suppression only when approved or present in code.
- Do not invent cache key dimensions, TTL, jitter, negative-cache stability rules, or migration resume checkpoints.

## Good API Example
```go
func TestCreateWidgetRejectsUnknownJSONField(t *testing.T) {
	handler := NewWidgetHandler(fakeWidgetService{})
	req := httptest.NewRequest(http.MethodPost, "/widgets", strings.NewReader(`{"name":"a","surprise":true}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
	if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
		t.Fatalf("content type = %q, want problem JSON", got)
	}
}
```

Use this only when strict unknown-field rejection and status mapping are approved or already established.

## Bad API Example
```go
func TestCreateWidgetRejectsUnknownJSONField(t *testing.T) {
	resp := callCreate(`{"name":"a","surprise":true}`)
	if resp.Code >= 400 {
		return
	}
	t.Fatal("request failed")
}
```

Why it is bad: any error status passes and the failure message does not preserve the contract split.

## Good Data Example
```go
func TestListRecentReturnsRowsInDescendingCreateOrder(t *testing.T) {
	pool := setupPostgresPoolWithMigrations(t)
	repo := postgres.NewPingHistoryRepository(pool.DB())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	first, err := repo.Create(ctx, "first")
	if err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	second, err := repo.Create(ctx, "second")
	if err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	recent, err := repo.ListRecent(ctx, 2)
	if err != nil {
		t.Fatalf("ListRecent() error = %v", err)
	}
	if recent[0].ID != second.ID || recent[1].ID != first.ID {
		t.Fatalf("recent IDs = [%d,%d], want [%d,%d]", recent[0].ID, recent[1].ID, second.ID, first.ID)
	}
}
```

Why it is good: real query ordering is an integration concern and uses bounded context plus cleanup through the setup helper.

## Bad Cache Example
```go
func TestAccountSummaryCacheKey(t *testing.T) {
	if got := key("tenant-a", "acct-1"); got != "summary:v3:tenant-a:acct-1" {
		t.Fatalf("key = %q", got)
	}
}
```

Why it is bad unless explicitly approved: it freezes key internals instead of proving tenant isolation or stale-data prevention.

## Assertion Patterns
- API: `status`, `Content-Type` prefix, required headers, stable response fields, and absence of side effects after rejection.
- Idempotency: same key and same payload returns existing operation; same key and different payload conflicts only when conflict semantics are approved; concurrent same-key starts create one side effect.
- SQL: row count, order, transaction outcome, wrapped query errors, `sql.ErrNoRows`/driver categories, and migration compatibility.
- Cache: source call counts, hit/miss outcomes, tenant separation, stale fallback category, and recovery from cache error without assuming internals.

## Deterministic Coordination Patterns
- For handler tests, avoid real network when `httptest.NewRecorder` proves the contract.
- For integration tests, use `context.WithTimeout`, `t.Cleanup`, build tags, and Docker-unavailable skip policy consistent with the repo.
- For concurrent idempotency or stampede tests, use a gated fake store so all goroutines overlap at the reservation/read boundary.
- For cache TTL tests, use an injected clock when available. If no clock exists and TTL exactness is required, escalate the missing testability seam.

## Repository-Local Cues
- `internal/infra/http/router_test.go` covers 404/405/OPTIONS/CORS, request ID, security headers, and body limit behavior.
- `internal/infra/http/openapi_contract_test.go` covers runtime contract endpoints with `httptest`.
- `test/postgres_sqlc_integration_test.go` applies migrations and tests generated-query repository behavior through Postgres.
- `test/postgres_integration_test.go` skips or fails on Docker unavailability based on `REQUIRE_DOCKER`.

## Exa Source Links
- [net/http/httptest package](https://pkg.go.dev/net/http/httptest)
- [Go database querying documentation](https://go.dev/doc/database/querying)
- [database/sql example tests](https://go.dev/src/database/sql/example_test.go)
- [database/sql fake driver tests](https://go.dev/src/database/sql/fakedb_test.go)
- [Go testing package](https://pkg.go.dev/testing)

