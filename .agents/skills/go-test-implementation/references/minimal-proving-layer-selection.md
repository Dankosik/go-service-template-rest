# Minimal Proving Layer Selection

## Behavior Change Thesis
When loaded for test-layer ambiguity, this file makes the model choose the smallest layer that can observe the regression instead of likely mistake: defaulting to slow integration coverage or faking away behavior that only a boundary test can prove.

## When To Load
Load this when the task must choose among unit, handler or contract, integration, fuzz, benchmark, or example tests for already-approved behavior.

## Decision Rubric
- Start with the regression signal: what would be wrong if the behavior broke?
- Use a unit test when local validation, mapping, error wrapping, state transition, or side-effect suppression can prove the obligation.
- Use a handler or contract test when method, path, status, headers, content type, body shape, strict decode, idempotency, or generated/manual route integration is the behavior.
- Use an integration test when real SQL, migrations, transactions, locks, driver behavior, process lifecycle, cache backend behavior, or multi-component wiring is part of the obligation.
- Use fuzz tests for parsers, decoders, serializers, validators, and protocol logic where input variety matters beyond named examples.
- Use benchmarks only for approved latency, allocation, throughput, contention, or capacity obligations.
- Use examples only when executable public usage documentation is part of the value.

## Imitate
Mapper behavior belongs at unit level:

```go
func TestPingHistoryRepositoryCreateRejectsNullCreatedAt(t *testing.T) {
	t.Parallel()

	repo := newPingHistoryRepositoryWithQuerier(fakePingHistoryQuerier{
		create: func(context.Context, string) (sqlcgen.PingHistory, error) {
			return sqlcgen.PingHistory{ID: 9, Payload: "x"}, nil
		},
		list: func(context.Context, int32) ([]sqlcgen.PingHistory, error) { return nil, nil },
	})

	_, err := repo.Create(context.Background(), "x")
	if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("Create() error = %v, want ErrPingHistoryRepository", err)
	}
}
```

Copy the shape: fake the query result because the obligation is local mapper/error behavior, not Postgres.

Query ordering belongs at integration level:

```go
func TestPingHistoryRepositorySQLCReadWriteReturnsNewestFirst(t *testing.T) {
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

	got, err := repo.ListRecent(ctx, 2)
	if err != nil {
		t.Fatalf("ListRecent() error = %v", err)
	}
	if got[0].ID != second.ID || got[1].ID != first.ID {
		t.Fatalf("IDs = [%d,%d], want [%d,%d]", got[0].ID, got[1].ID, second.ID, first.ID)
	}
}
```

Copy the shape: real migration/query behavior is the reason to pay integration-test cost.

## Reject
```go
func TestCreateRejectsNullCreatedAtThroughDockerPostgres(t *testing.T) {
	// Starts a real database and drives the full service even though the
	// obligation is mapper behavior.
}
```

Reject because it pushes a local invariant to a slower and more failure-prone layer.

```go
func TestCreateWidgetAcceptsRequestByCallingServiceDirectly(t *testing.T) {
	err := NewWidgetService(fakeStore{}).Create(context.Background(), Widget{Name: "a"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}
```

Reject when the approved obligation is HTTP status, content type, strict decode, or generated contract behavior. A service unit test cannot prove transport mapping.

## Agent Traps
- Choosing integration tests because they feel more "real" even when they do not observe the targeted failure better.
- Mocking an HTTP handler's request parser or a SQL driver's ordering behavior, then claiming contract or integration proof.
- Adding fuzz tests for ordinary branch coverage instead of input-heavy invariants.
- Writing benchmarks without an approved performance obligation.
- Treating examples as tests for internal packages where executable docs add no user value.

## Validation Shape
- Unit or handler test: focused `go test <package> -run '^TestName$' -count=1`, then package-level command when helpers changed.
- Integration test: repository integration target or `go test -tags=integration ./test/...`, following the repo's Docker skip/fail policy.
- Fuzz target: plain `go test` for seed regressions plus a bounded fuzz command only when exploration is part of the obligation.
- Benchmark: run the named benchmark with stable setup and report that it is measurement evidence, not functional proof.
