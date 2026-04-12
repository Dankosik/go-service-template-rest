# Component Map

## Affected Components

| Component | Current Role | Planned Change |
| --- | --- | --- |
| `internal/config/validate.go` | Validates config values, readiness budgets, Redis mode policy, and Mongo probe-address parsing. | Keep aggregate readiness validation; use the local normalized Redis mode in `validateRedis`; optionally simplify the redundant Mongo bare-IPv6 predicate. |
| `internal/config/load_koanf.go` | Loads defaults/files/env and selects local versus hardened file policy. | Replace the environment-shaped predicate with a policy-shaped helper returning `configFilePolicy`. Preserve fail-closed behavior for explicit files without a local env hint. |
| `cmd/service/internal/bootstrap/startup_dependencies.go` | Registers runtime dependency probes and applies startup dependency policy. | Wrap the Postgres runtime readiness probe with the configured `Postgres.HealthcheckTimeout` before adding it to `health.Service`. |
| `cmd/service/internal/bootstrap/startup_dependencies_additional_test.go` | Holds focused bootstrap dependency-probe tests. | Add coverage proving the Postgres readiness wrapper caps the context deadline. |
| `docs/configuration-source-policy.md` | Owns config source, secret, runtime budget, and extension-point policy. | Clarify which Redis/Mongo keys are active guard/probe controls versus reserved future adapter/cache/store settings. |
| `docs/project-structure-and-module-organization.md` | Owns package-placement and extension guidance. | Keep Redis/Mongo extension guidance aligned with the reserved-key policy. |

## Stable Components

| Component | Stable Behavior |
| --- | --- |
| `internal/app/health.Service` | Still runs probes sequentially under one outer readiness context. |
| `internal/infra/postgres.Pool.Check` | Still accepts caller-owned context and pings Postgres. Bootstrap supplies the per-probe cap when registering runtime readiness. |
| `internal/config/defaults.go`, `internal/config/types.go`, `internal/config/snapshot.go`, `env/config/*.yaml`, `env/.env.example` | No required key removal or default changes for this pass. Touch only if implementation discovers docs cannot be made honest without a recorded spec reopen. |
| Redis/Mongo bootstrap probe behavior | Still guard/probe-only; no adapter runtime is introduced. |

## Cross-Component Contract

`internal/config` validates that the outer readiness timeout can cover sequential per-probe budgets. `cmd/service/internal/bootstrap` must ensure each runtime dependency probe actually honors its configured per-probe budget. `internal/app/health.Service` remains the sequential runner and does not need to know individual dependency budgets.
