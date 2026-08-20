# Persistence Architecture

Load for a PostgreSQL pool, repository, transaction, migration, query, or
durable-schema change.

`internal/infra/postgres/` owns strict connection admission, template defaults,
commit-outcome policy, and repository code over `pgxpool`.
`internal/infra/postgresmigrate/` owns migration execution for `cmd/migrate`.
Canonical Goose files under `migrations/` own schema; SQLC query sources own
generated access code. Runtime repositories map generated rows into
feature-facing types and join a caller transaction only through the existing
transaction seam.

Feature packages own persistence ports and business invariants. Bootstrap owns
the concrete pool, repository wiring, partial-startup cleanup, readiness
participation, and shutdown. Pool mechanics do not own process lifecycle, HTTP
behavior, config precedence, or feature truth.

Load a capability document only when that pack is affected:

<!-- profile:http-idempotency-postgres:start -->
- [HTTP idempotency](../postgres-http-idempotency.md)
<!-- profile:http-idempotency-postgres:end -->
<!-- profile:outbox-postgres:start -->
- [Transactional outbox](../postgres-transactional-outbox.md)
<!-- profile:outbox-postgres:end -->
<!-- profile:jobs-postgres:start -->
- [Durable background jobs](../postgres-durable-background-jobs.md)
<!-- profile:jobs-postgres:end -->

A new durable behavior evolves `migrations/` first, then regenerates or adapts
access code from that schema.
