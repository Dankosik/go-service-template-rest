# Data, Cache, And Integration Test Patterns

## Behavior Change Thesis
When loaded for SQL, migrations, repository, cache, tenant isolation, or integration-test behavior, this file makes the model prove durable state, transaction/cache semantics, and failure categories at the right seam instead of likely mistake: overusing Postgres for mapper tests, freezing cache keys, or inventing migration/backfill mechanisms.

## When To Load
Load this when tests touch SQL repositories, generated queries, migrations, transactions, row scanning, tenant isolation, cache hit/miss/stale/fallback behavior, TTLs, stampede suppression, testcontainers, Docker-gated integration tests, or datastore/cache degradation.

## Decision Rubric
- Use fakes for repository mapper behavior, error wrapping, and local call-count side effects.
- Use integration tests for real SQL, migration compatibility, transaction rollback/commit behavior, locks, query ordering, driver errors, and generated query contracts.
- Test `QueryRow`/`Scan`, `Rows.Next`, `Rows.Err`, and `Rows.Close` behavior at the layer where the code owns those outcomes.
- For cache behavior, assert hit, miss, stale, expired, bypass, fallback, corruption, tenant separation, and stampede suppression only when approved or present in code.
- Prefer stale-data prevention, side-effect count, and tenant isolation over exact key strings.
- Escalate missing TTL clock seams, negative-cache stability, cache-key dimensions, migration resume checkpoints, and mixed-version rules instead of inventing them in tests.
- Follow repository integration policy: `//go:build integration`, Docker skip locally, `REQUIRE_DOCKER=1` fail in CI when relevant.

## Imitate
```go
func TestRepositoryCreateWrapsScanError(t *testing.T) {
	sentinel := errors.New("scan failed")
	repo := newRepositoryWithQuerier(fakeQuerier{
		create: func(context.Context, string) (Record, error) {
			return Record{}, sentinel
		},
	})

	_, err := repo.Create(context.Background(), "payload")
	if !errors.Is(err, ErrRepository) {
		t.Fatalf("Create() error = %v, want ErrRepository", err)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("Create() error = %v, want wrapped scan failure", err)
	}
}
```

Copy the shape: fake the query seam because the obligation is repository error category and wrapping.

```go
func TestAccountSummaryCacheDoesNotShareTenants(t *testing.T) {
	origin := newRecordingAccountOrigin(map[accountKey]Summary{
		{tenant: "tenant-a", account: "acct-1"}: {Balance: 10},
		{tenant: "tenant-b", account: "acct-1"}: {Balance: 20},
	})
	cache := newAccountSummaryCache(origin)

	first, err := cache.Get(context.Background(), "tenant-a", "acct-1")
	if err != nil {
		t.Fatalf("Get(tenant-a) error = %v", err)
	}
	second, err := cache.Get(context.Background(), "tenant-b", "acct-1")
	if err != nil {
		t.Fatalf("Get(tenant-b) error = %v", err)
	}
	if first.Balance == second.Balance {
		t.Fatalf("tenant summaries share value = %v, want isolated results", first.Balance)
	}
}
```

Copy the shape: tenant isolation is proven by behavior, not by a guessed cache-key format.

## Reject
```go
func TestAccountSummaryCacheKey(t *testing.T) {
	if got := key("tenant-a", "acct-1"); got != "summary:v3:tenant-a:acct-1" {
		t.Fatalf("key = %q", got)
	}
}
```

Reject unless the approved design or existing public contract explicitly names that key format.

```go
func TestRepositoryCreateWrapsScanErrorThroughDocker(t *testing.T) {
	// Starts Postgres and tries to force a mapper-only error through SQL setup.
}
```

Reject because SQL integration cost is not justified when a fake query result can prove the repository-owned behavior.

## Agent Traps
- Freezing SQL text or cache-key internals instead of proving durable behavior.
- Treating a cache hit test as tenant-isolation proof without a cross-tenant case.
- Sleeping through TTLs when an injected clock or escalation is the honest path.
- Testing Docker unavailability differently from the repo's local skip and CI fail policy.
- Adding migration resume or negative-cache semantics that the approved task ledger has not chosen.
- Ignoring `Rows.Err` or cleanup behavior when the code owns row iteration.

## Validation Shape
- Repository unit test: focused package command with `-count=1`.
- Integration test: `make test-integration` or `go test -tags=integration ./test/...` according to scope; use `REQUIRE_DOCKER=1` only when the desired proof should fail on missing Docker.
- Cache concurrency or stampede test: add race-aware validation when shared state or parallel calls are involved.
- Migration-sensitive test: pair the focused integration command with any repo migration validation target if the migration files changed.
