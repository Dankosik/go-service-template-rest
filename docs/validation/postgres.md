# PostgreSQL Validation

Use real PostgreSQL for transaction, locking, concurrency, visibility,
migration, or adapter-integration claims. Execute focused selectors during
final validation or standalone debugging, within the applicable heavy-run
authority. A completed database surface uses
`REQUIRE_DOCKER=1 ALLOW_HEAVY=1 make test-integration-db`.
Add `test-integration-messaging` or `test-integration-process` only when those
surfaces changed; `test-integration` remains the explicit full non-race pack.
CI sets `CI=true`, which satisfies the heavy-target guard.

Migration source shape uses `make migration-check`; reversible runtime rehearsal
uses `ALLOW_HEAVY=1 make migration-validate`. A missing container runtime is an unverified
remainder, not a successful skip.
