# PostgreSQL Validation

Use this branch for explicitly required database verification or a concrete
bounded diagnostic under the [Evidence Contract](../spec-first-workflow/shared/evidence-contract.md#required-and-optional-proof).
Changing a PostgreSQL file does not add a local integration gate; ordinary Go
completion still follows [Go Validation](go.md).

A claim of observed transaction, locking, concurrency, visibility, migration,
or adapter-integration behavior needs real PostgreSQL. Use existing focused
selectors and applicable heavy-run authority. The canonical database pack is
`REQUIRE_DOCKER=1 ALLOW_HEAVY=1 make test-integration-db`.
Add `test-integration-messaging` or `test-integration-process` only for their
explicitly required observations; `test-integration` is the full non-race pack.
Actual CI satisfies the heavy guard and retains its existing routing.

Migration source shape uses `make migration-check`; a required runtime rehearsal
uses `ALLOW_HEAVY=1 make migration-validate`. Missing Docker is not a pass for
that scenario. If the scenario is optional, disclose its material gap and stop
without building or repairing a test environment; it does not block local completion.
