# Runtime Sequence

## Current Mismatch

1. `internal/config.LoadDetailed` builds and validates the config snapshot.
2. `validateReadinessProbeBudgets` counts `postgres.healthcheck_timeout` when Postgres readiness is enabled.
3. Bootstrap initializes Postgres with `cfg.Postgres.HealthcheckTimeout` for startup ping.
4. Bootstrap adds the raw `*postgres.Pool` to runtime readiness probes.
5. `/health/ready` creates one outer context from `http.readiness_timeout`.
6. `internal/app/health.Service.Ready` runs probes sequentially.
7. `postgres.Pool.Check(ctx)` uses the outer context directly, so Postgres can consume more than `postgres.healthcheck_timeout`.

## Planned Runtime Sequence

1. `internal/config.LoadDetailed` keeps the existing validation rule.
2. Bootstrap initializes Postgres as today.
3. When `cfg.PostgresReadinessProbeRequired()` is true, bootstrap registers a bounded Postgres readiness probe instead of the raw pool:
   - `Name()` remains `postgres`.
   - `Check(ctx)` derives a child context with `withStageBudget(ctx, cfg.Postgres.HealthcheckTimeout)`.
   - The child context is canceled after `pg.Check(childCtx)` returns.
4. `/health/ready` still creates one outer context from `http.readiness_timeout`.
5. `health.Service.Ready` still runs probes sequentially.
6. Postgres runtime readiness now consumes no more than the smaller of the outer readiness remaining time and `postgres.healthcheck_timeout`.
7. Redis and Mongo readiness behavior remains unchanged:
   - Redis uses `redis.dial_timeout` through `probeRedisWithContext`.
   - Mongo uses `mongo.connect_timeout` through `probeMongoWithContext`.

## Failure Semantics

- If Postgres exceeds `postgres.healthcheck_timeout`, the Postgres probe fails and `/health/ready` returns not ready.
- If the outer readiness context is already shorter than `postgres.healthcheck_timeout`, the outer deadline wins.
- A Postgres readiness failure remains a readiness failure, not a liveness failure.
- Startup admission and startup dependency probing remain unchanged.

## Validation Hooks

Add a focused bootstrap test around the bounded Postgres readiness probe. The test should use a fake probe or helper-level seam to observe that:
- a parent context with no nearer deadline receives a child deadline near `postgres.healthcheck_timeout`;
- a parent context with a nearer deadline is not extended;
- cancellation is still propagated.
