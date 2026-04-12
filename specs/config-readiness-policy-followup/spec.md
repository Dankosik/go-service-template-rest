# Config Readiness Policy Follow-up Spec

## Context

The `internal/config` review found that most of the package is idiomatic and well covered, but several findings need a pre-implementation decision record:
- `internal/config` validates `http.readiness_timeout` against `postgres.healthcheck_timeout`, `redis.dial_timeout`, and `mongo.connect_timeout`, while runtime Postgres readiness currently uses the outer readiness context only.
- Redis and Mongo config include future adapter/cache/store fields even though repository docs describe Redis/Mongo as guard-only extension stubs in the baseline template.
- `isLocalEnvironmentHint` hides file-policy behavior behind a local-environment predicate.
- `validateRedis` computes a normalized mode and then uses `cfg.StoreMode()` for the same branch.
- A low-risk Mongo probe-address branch can be simplified if `internal/config/validate.go` is already being edited.

## Scope / Non-goals

In scope:
- Preserve the existing aggregate readiness-budget contract and make runtime Postgres readiness obey it.
- Clarify that unused Redis/Mongo adapter/cache/store fields are reserved extension keys with no baseline runtime behavior.
- Improve local readability in config-file policy selection and Redis mode validation.
- Optionally simplify the redundant bare-IPv6 predicate in `normalizeMongoProbeAddress`.

Non-goals:
- Do not implement Redis or Mongo adapters.
- Do not remove existing Redis/Mongo config keys in this pass.
- Do not change REST API behavior, OpenAPI output, database migrations, telemetry label names, or dependency criticality.
- Do not change readiness probe ordering or liveness semantics.

## Constraints

- `internal/config` remains responsible for building and validating the immutable config snapshot.
- `cmd/service/internal/bootstrap` remains responsible for runtime dependency admission and traffic-readiness policy binding.
- `internal/app/health.Service` continues to run enabled dependency probes sequentially under one outer readiness context.
- `/health/live` remains process-only; external dependency checks remain readiness-only.
- Redis/Mongo extension fields must not imply runtime adapter behavior until an app feature owns a real adapter under `internal/infra/redis` or `internal/infra/mongo`.
- Changes should be backward-compatible with existing config files and env examples.

## Decisions

1. Keep the current config validation rule that `http.readiness_timeout` must cover the aggregate sequential readiness probe budget.
2. Fix the Postgres mismatch in bootstrap by wrapping the runtime Postgres readiness probe with `withStageBudget(ctx, cfg.Postgres.HealthcheckTimeout)` before calling `pg.Check`. Do not change `internal/app/health.Service` sequencing.
3. Keep `postgres.Pool.Check(ctx)` as a context-driven adapter probe for this pass. Bootstrap owns applying the config-derived per-probe readiness budget when it registers the probe.
4. Keep existing Redis/Mongo config keys for compatibility, but document unused adapter/cache/store fields as reserved extension API with no baseline runtime effect. If the desired decision is to remove those keys instead, reopen this spec before implementation.
5. Rename/restructure `isLocalEnvironmentHint` into a policy-named helper that returns `configFilePolicy`, so the fail-closed behavior for explicit config files without an env hint is visible at the call site.
6. In `validateRedis`, use the local normalized `mode` for the store-mode guard branch.
7. If `internal/config/validate.go` is already being edited, simplify the bare-IPv6 branch in `normalizeMongoProbeAddress` after the bracket-handling block. This is behavior-preserving cleanup, not a parsing redesign.

## Open Questions / Assumptions

- Assumption: preserving the existing Redis/Mongo config surface is preferred over removing unused extension keys because these keys already appear in defaults, env examples, docs, and tests.
- Assumption: no new runtime Redis/Mongo semantics should be introduced while documenting reserved keys.
- Assumption: focused tests can cover the Postgres readiness budget wrapper with a fake `health.Probe` or a small bootstrap helper, without requiring a live Postgres instance.

## Plan Summary / Link

Use `plan.md` and `tasks.md` for the implementation handoff.

Implementation should start from `workflow-plans/implementation-phase-1.md` in a later session.

## Validation

Fresh validation evidence after implementation:
- `go test ./cmd/service/internal/bootstrap -run 'Test.*Postgres.*Readiness|TestRunDependencyProbe|TestInitStartupDependenciesAllDisabled' -count=1`: passed.
- `go test ./internal/config -run 'Test(LocalAllowsSymlinkConfig|ConfigFileWithoutEnvironmentHintFailsClosed|NonLocalRejectsSymlinkConfig|RedisStoreGuard|RedisModePolicyHelpers|MongoProbeAddress)' -count=1`: passed.
- `go test ./internal/config ./cmd/service/internal/bootstrap ./internal/infra/postgres -count=1`: passed.

## Outcome

Implemented and validated.

- Runtime Postgres readiness is now bounded in bootstrap by `postgres.healthcheck_timeout` before calling the Postgres pool check, without changing `internal/app/health.Service` sequencing or global `postgres.Pool.Check` semantics.
- Redis/Mongo docs now distinguish active guard/probe controls from reserved future adapter/cache/store keys.
- Config-file policy selection now uses a policy-returning helper, Redis store-mode validation uses the local normalized mode, and the Mongo bare-IPv6 branch was simplified without changing covered behavior.
