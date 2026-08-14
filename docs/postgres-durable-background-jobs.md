<!-- profile:jobs-postgres:start -->
# PostgreSQL durable background jobs

`JOBS=postgres` retains the reusable default-off job pack and `/jobs-worker`.
The worker fails closed before database I/O until a derived service supplies a
concrete builder. It neither creates a producer nor enables operator controls,
periodic scheduling, multiple classes, deployment, or capacity claims.

Run `/migrate` for schema changes; application and worker startup never migrate.
<!-- profile:jobs-postgres:end -->
