# Postgres Integration Proof

## Behavior Change Thesis
When loaded for repository, migration, transaction, or cache proof, this file
makes the model run the cases the engine decides against a real PostgreSQL — the
common defect is a fake querier standing in for the database on behavior the
database owns, where `SKIP LOCKED` exclusion, constraint violation, and query
ordering are whatever the fake was written to return, so the test agrees with
itself and rejects nothing.

## When To Load
Symptom: the obligation involves SQL, migrations, transactions, row locks,
concurrent claims, tenant isolation, or a cache whose backend behavior matters.

## Decision Rubric
- Split by owner. The engine decides lock exclusion, `ORDER BY`, unique and
  foreign-key violations, serialization failures, rollback visibility, and
  redelivery after a claim expires; none survive a fake. Go code decides error
  wrapping and category, row mapping, side-effect suppression, and validation;
  those need no container and get a fake querier.
- `internal/infra/postgres/pgtest` owns the fixture. `pgtest.Main(m, "")` from
  `TestMain` shares one container per test binary; `pgtest.Migrated(tb, fsys,
  "migrations")` or `pgtest.DSN(tb)` returns a freshly created, migrated database
  per test. Cases cannot see each other's rows, so parallel packages need no
  `TRUNCATE` step and no `-p 1`.
- Without Docker the fixture skips; with `REQUIRE_DOCKER=1` it fails instead. CI
  sets that variable so a missing daemon cannot report a green suite that ran
  nothing. Reproduce a CI-only failure by setting it locally rather than by
  changing the fixture.
- Prove exclusion by holding the row. Open a second transaction, `SELECT ... FOR
  UPDATE` the contended row, then assert the claim under test skips to the next
  one — the pattern in `test/postgres_outbox_claim_integration_test.go`. Two goroutines
  racing for the same claim proves timing, not the lock.
- Prove tenant isolation and cache freshness through behavior — a value that
  cannot cross tenants, a read that does not observe stale state — never through
  the key string the current implementation builds.
- A TTL with no injectable clock is an escalation. Sleeping through a real
  expiry buys a slow test and a flaky one.
- Tag the file `//go:build integration`. `make test-integration` runs
  `./test/...` plus `./internal/infra/natsjs` and `./cmd/worker/internal/bootstrap`
  with `-p=1 -count=1`, so a package outside that set is not covered by the gate
  a claim of integration proof implies.

## Reject
- A container started to reach behavior a fake query result already decides: it
  buys minutes of latency and the same assertion.
- Frozen SQL text or a pinned cache key standing in for durable behavior.
- Integration proof reported as skipped-because-Docker without saying so; the run
  produced no evidence for the claim it was cited under.

## Validation Shape
- Repository unit test: focused package command with `-count=1 -vet=off`.
- Integration test: `make test-integration`, or the tagged package directly while
  iterating.
- Migration-sensitive change: pair the integration command with the repository's
  migration validation, since `pgtest.Migrated` runs migrations per database and
  a broken migration fails every case at setup.
