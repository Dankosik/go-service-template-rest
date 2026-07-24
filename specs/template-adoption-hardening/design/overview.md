# Adoption hardening design

## Runtime and security

- `internal/infra/http.NewRouter` constructs only the client API handler.
- Bootstrap constructs an optional diagnostics `http.Server` around
  `metrics.Handler()`, binds it before readiness, and owns its shutdown.
- Application and diagnostics serve errors share one bounded result path.
  Application failure is terminal; configured diagnostics failure is terminal
  because silently losing the selected telemetry path is unsafe.
- Ingress admission still requires an explicit declaration for non-local
  wildcard application binds, but `true` is now valid because diagnostics are
  separate.
- Unknown configuration is rejected in `internal/config` for both service and
  migrator callers; the obsolete permissive CLI surface is removed.

## Lifecycle

Keep one application listener, at most one diagnostics listener, one buffered
result channel owned by bootstrap, and one goroutine per configured server.
Every goroutine reports once. Bootstrap closes listeners through bounded
`http.Server.Shutdown`; no sender can block after return. Preserve startup
rejection telemetry, readiness admission, readiness-first drain, propagation
delay, and telemetry cleanup while removing the outcome state object and split
pre/post-ready wait helpers.

## Derived-service profiles

`scripts/init-module.sh` is the existing profile owner. It validates profile
values before mutation, establishes identity, and then applies exact repository
transformations:

- `DATABASE=none` removes PostgreSQL runtime, migration, test, example,
  generated, tool, and delivery surfaces; `postgres` retains them. The shared
  typed config vocabulary remains so upstream profile changes are reviewable,
  but enabling PostgreSQL without the profile fails startup explicitly.
- `AGENT_WORKFLOW=none` removes reusable agent corpora, harness mirrors,
  historical template specs, and template-only workflow prose, then writes a
  compact service-local contract; `full` retains them.

The full-template CI creates complete temporary derived repositories for both
profile combinations and runs their narrow build/configuration proof. No new
templating dependency is added.

## Ownership and generated boundaries

- Service identity: initializer script plus exact initializer contract test.
- Config schema: `internal/config/types.go`; defaults and validation stay in
  their existing owners.
- Metrics collection: `internal/infra/telemetry`; listener lifecycle:
  bootstrap.
- PostgreSQL pool and migrations become separate packages so the service import
  graph does not compile the migrator.
- OpenAPI and sqlc authorities remain unchanged.
- GHCR admission remains in `.github/workflows/cd.yml`; existing digest-bound
  release steps are preserved.
- Tool versions live in their narrow owner with a consistency check for values
  duplicated by platform syntax.

## Rollout and rollback

The template repository retains every full capability. Only newly initialized
derived repositories receive minimal defaults, so existing consumers do not
lose files during an upstream merge. Runtime rollout requires setting
`APP__OBSERVABILITY__METRICS__ADDR` only when a different private diagnostics
bind is needed. Rollback is the preceding template commit; no persisted data or
external service contract changes.
