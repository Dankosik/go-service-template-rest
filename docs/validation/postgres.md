# PostgreSQL Validation

Use real PostgreSQL for transaction, locking, concurrency, visibility,
migration, or adapter-integration claims. Run the focused tagged selector while
iterating and `REQUIRE_DOCKER=1 make test-integration` for the completed surface.

Migration source shape uses `make migration-check`; reversible runtime rehearsal
uses `make migration-validate`. A missing container runtime is an unverified
remainder, not a successful skip.
