# Integration Tests

This Codex-native repository keeps broad executable proof in this directory.

Store end-to-end, migration-backed, container-backed, and larger cross-package tests in this directory.

Integration tests use the `integration` build tag and are not executed by default.

Run locally:

```bash
make test-integration
# require rather than conditionally skip Docker-backed cases:
REQUIRE_DOCKER=1 make test-integration
```

Placement rules:
- Put repository-local unit tests beside the package under `internal/...`; use `test/` only when the test needs a real external dependency, multi-package flow, or larger fixture.
- Put broad service integration tests at the `test/` root. Use `test/<feature>/` only when a feature owns enough scenarios or helpers that a subdirectory keeps the root readable.
- Use `package integration_test` by default so tests exercise exported package boundaries. Use a same-package integration test only when the test must prove unexported integration behavior.
- Every integration test file must start with `//go:build integration`.
- Put real-database `BenchmarkXxx` functions in the same integration package so
  they can reuse the PostgreSQL 17 Testcontainers and migration helpers. Keep
  in-process benchmarks beside their owning package instead.

Feature-author placement:

| Surface | Prefer tests |
| --- | --- |
| Handler mapping, OpenAPI contract policy, Problem responses, generated-route ownership, and route labels | Beside `internal/infra/http`. |
| App use-case behavior and app-owned ports | Beside `internal/app/<feature>`. |
| Runtime config keys, defaults, snapshot construction, validation, and secret-source policy | Beside `internal/config`. |
| Repository mapping and SQLC adapter behavior | Beside `internal/infra/postgres`; use `test/` only for container-backed behavior. |
| Feature bootstrap wiring for a real adapter | Beside `cmd/service/internal/bootstrap`; prove disabled, ready, policy-denied, and partial-initialization cleanup paths before adding broad integration coverage. |
| Telemetry instruments and lifecycle/bootstrap behavior | Beside `internal/infra/telemetry` or `cmd/service/internal/bootstrap`, matching the owner. |
| Endpoint plus real persistence plus bootstrap composition | Target the owning packages first, then use `test/` with the `integration` tag when a real database-backed flow is required to prove the combined contract. |
| Generated drift for OpenAPI and SQLC | Use the owning make targets instead of integration tests. |

Docker behavior:
- Local `make test-integration` skips when Docker is unavailable.
- CI sets `REQUIRE_DOCKER=1`, so Docker unavailability fails the job instead of silently skipping.

Migration-backed helpers:
- Prefer `make migration-validate` when the claim is migration correctness.
- Test helpers that execute `env/migrations/*.up.sql` directly are schema bootstrap helpers for integration setup, not full migration rehearsal.
- Apply migration files in sorted order, fail on empty globs, use bounded contexts, and clean up containers and pools with `t.Cleanup`.

Database benchmark behavior:
- `make bench-db` sets `REQUIRE_DOCKER=1`, selects the `integration` build tag,
  and fails rather than skipping when Docker, `BENCH_DB_WORKLOAD_ID`, or a
  matching benchmark is absent.
- The shared PostgreSQL image is digest-pinned; result metadata records that
  digest, the migration fingerprint, and the named fixture/workload.
- Seed and `ANALYZE` representative fixtures before timing. Name row-count,
  selectivity, cache state, and concurrency cases separately.
- Keep production-owned transactions, round trips, decoding, and commit inside
  the measured boundary. Reset mutable fixtures outside it and clean up with
  `b.Cleanup`.
- Use `make bench-db-baseline` and `make bench-db-compare` for repeated A/B
  evidence. See [Benchmarking](../docs/benchmarking.md) for the full protocol.
