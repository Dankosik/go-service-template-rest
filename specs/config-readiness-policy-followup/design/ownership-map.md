# Ownership Map

## Source-Of-Truth Rules

| Surface | Owns | Must Not Own |
| --- | --- | --- |
| `internal/config` | Config key shape, defaults, parsing, validation, and helper policy decisions local to config loading. | Runtime dependency behavior, adapter implementation, request handling, or traffic-admission sequencing. |
| `cmd/service/internal/bootstrap` | Composition root, dependency startup/admission, runtime readiness policy binding, and config-to-runtime wiring. | Feature behavior, persistence semantics, or generic config parsing rules. |
| `internal/app/health` | The consumer-owned `Probe` contract and sequential readiness runner. | Per-dependency budget selection or dependency-specific adapter details. |
| `internal/infra/postgres` | Postgres pool creation, ping/check behavior under a caller-owned context, and Postgres-specific connection mechanics. | Bootstrap traffic-admission policy or Redis/Mongo extension semantics. |
| Repository docs | Operator-visible config and extension-point contracts. | Hidden runtime behavior not implemented in code. |

## Redis/Mongo Extension Ownership

Active baseline guard/probe behavior:
- Redis/Mongo enabled flags and probe addresses participate in bootstrap dependency checks.
- Redis/Mongo readiness flags decide whether guard probes join runtime readiness.
- Redis store mode remains a guarded critical path and still requires explicit opt-in.

Reserved extension behavior:
- Redis cache/store knobs such as key prefix, TTLs, singleflight, fallback concurrency, username/password, and DB are reserved for future adapter work unless currently consumed by bootstrap probes.
- Mongo database, server-selection timeout, and max-pool-size settings are reserved for future adapter work unless currently consumed by bootstrap probes.
- These keys may remain validated for compatibility, but docs must say they do not create cache/store/database runtime behavior in the baseline template.

## Reopen Rules

Reopen `spec.md` if:
- The desired fix is to remove reserved Redis/Mongo keys instead of documenting them.
- Runtime readiness must change from sequential to parallel probes.
- Postgres readiness timeout must become globally owned by `postgres.Pool.Check` rather than the bootstrap readiness wrapper.

Reopen technical design if:
- A future feature needs real Redis/Mongo adapter behavior.
- A future change introduces a shared health-probe timeout abstraction across dependencies.
