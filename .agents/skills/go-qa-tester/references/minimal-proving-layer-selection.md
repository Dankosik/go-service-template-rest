# Minimal Proving Layer Selection

## When To Load
Load this when you must choose between unit, integration, contract, fuzz, benchmark, or example tests for already-approved behavior.

## Selection Heuristics
- Use a unit test when local logic, validation, mapping, or state transition can prove the obligation.
- Use a contract or handler test when method, status, headers, payload, strict decode, idempotency, or transport error semantics are the behavior.
- Use an integration test when the behavior depends on a real datastore, migration, cache, network, generated SQL, process lifecycle, or multi-component seam.
- Use fuzz tests for parsers, decoders, serializers, validators, and protocol logic where many inputs can expose bugs beyond named examples.
- Use benchmarks only when latency, allocation, throughput, or contention behavior is an approved obligation.
- Use examples for public package usage documentation when executable docs are part of the value.

## Good Example
Approved obligation: "repository rejects a null `created_at` returned by the query layer."

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

Why it is good: a unit-level fake query result proves mapper behavior without starting Postgres.

## Bad Example
```go
func TestCreateRejectsNullCreatedAtThroughDockerPostgres(t *testing.T) {
	// Starts a real database, writes custom rows, and drives the full service
	// even though the obligation is mapper behavior.
}
```

Why it is bad: it pushes a local invariant to a slower, more failure-prone layer.

## Assertion Patterns By Layer
- Unit: assert returned values, sentinel or typed errors, dependency call counts, and local state transitions.
- Contract: assert method, path, status, headers, content type, body shape, and strict boundary behavior only where approved.
- Integration: assert durable state, transaction boundaries, migration compatibility, real query ordering, and cleanup.
- Fuzz: assert invariants such as round-trip, no panic, stable parse/format relation, and accepted/rejected categories.
- Benchmark: keep setup outside `b.Loop()` and assert only benchmark-specific setup errors.

## Deterministic Coordination Patterns
- Start small and add a higher layer only for behavior the smaller layer cannot observe.
- Use build tags like `//go:build integration` for Docker or external dependency tests when the repo already follows that pattern.
- Use seed corpora for fuzz tests so `go test` runs known regression inputs even when fuzzing is not enabled.
- Keep benchmarks free of network and wall-clock dependencies unless the approved performance obligation requires them.

## Repository-Local Cues
- `test/postgres_sqlc_integration_test.go` uses `//go:build integration` for real Postgres plus migrations.
- `internal/infra/http/openapi_contract_test.go` stays at handler/contract level for runtime HTTP behavior.
- `internal/infra/postgres/ping_history_repository_test.go` uses fakes for repository mapper and error wrapping paths.

## Exa Source Links
- [Go testing package](https://pkg.go.dev/testing)
- [Go fuzzing documentation](https://go.dev/doc/fuzz/)
- [Go command documentation](https://pkg.go.dev/cmd/go)
- [Go database querying documentation](https://go.dev/doc/database/querying)
- [database/sql example tests](https://go.dev/src/database/sql/example_test.go)

