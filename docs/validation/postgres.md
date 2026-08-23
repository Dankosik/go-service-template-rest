# PostgreSQL Validation

Use real PostgreSQL for transaction, locking, concurrency, visibility,
migration, or adapter-integration claims. Run the focused tagged selector while
iterating and `REQUIRE_DOCKER=1 ALLOW_HEAVY=1 make test-integration` for the
completed surface. CI sets `CI=true`, which satisfies the heavy-target guard.

Migration source shape uses `make migration-check`; reversible runtime rehearsal
uses `ALLOW_HEAVY=1 make migration-validate`. A missing container runtime is an unverified
remainder, not a successful skip.
