# Integration Tests

Store end-to-end, migration-backed, container-backed, and larger cross-package tests in this directory.

Integration tests use the `integration` build tag and are not executed by default.

Run locally:

```bash
make test-integration
# zero-setup mode:
make docker-test-integration
```

Placement rules:
- Put repository-local unit tests beside the package under `internal/...`; use `test/` only when the test needs a real external dependency, multi-package flow, or larger fixture.
- Put broad service integration tests at the `test/` root. Use `test/<feature>/` only when a feature owns enough scenarios or helpers that a subdirectory keeps the root readable.
- Use `package integration_test` by default so tests exercise exported package boundaries. Use a same-package integration test only when the test must prove unexported integration behavior.
- Every integration test file must start with `//go:build integration`.

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
| Generated drift for OpenAPI, SQLC, mocks, and stringer | Use the owning make targets instead of integration tests. |

Docker behavior:
- Local `make test-integration` skips when Docker is unavailable.
- `make docker-test-integration` runs the same tests through the Docker tooling image and passes the Docker socket when available.
- CI sets `REQUIRE_DOCKER=1`, so Docker unavailability fails the job instead of silently skipping.

Migration-backed helpers:
- Prefer real migration rehearsal targets such as `make migration-validate` / `make docker-migration-validate` when the claim is migration correctness.
- Test helpers that execute `env/migrations/*.up.sql` directly are schema bootstrap helpers for integration setup, not full migration rehearsal.
- Apply migration files in sorted order, fail on empty globs, use bounded contexts, and clean up containers and pools with `t.Cleanup`.
