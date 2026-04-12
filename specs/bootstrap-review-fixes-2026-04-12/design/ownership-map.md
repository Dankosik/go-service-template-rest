# Ownership Map

## Source-Of-Truth Ownership

`cmd/service/internal/bootstrap`

- Owns service composition, startup/shutdown flow, dependency admission, runtime network policy enforcement, and the new degraded dependency status/log helper.
- Owns reading the `NETWORK_*` operator-policy channel after the typed config snapshot is built.
- Must not grow Redis or Mongo runtime adapter semantics.

`internal/config`

- Owns ordinary immutable runtime config from code defaults, YAML, `APP__...`, and loader flags.
- Continues to own Redis/Mongo guard-only typed config and `MongoProbeAddress`.
- Does not gain `NETWORK_*` in this fix.

`internal/infra/telemetry`

- Owns the Prometheus metric implementation.
- Does not need a metric API change unless implementation proves `SetStartupDependencyStatus` cannot express degraded-ready state.

`docs/configuration-source-policy.md`

- Owns the explanation that `NETWORK_*` is a deliberate operational policy channel and not ordinary app config.

`env/.env.example`

- Owns discoverability examples. It must not silently enable or assert production network policy by default.

## Dependency Direction

- Bootstrap may depend on `internal/config` and `internal/infra/telemetry`.
- `internal/config` must not depend on bootstrap.
- Docs may describe the bootstrap-owned operator-policy channel, but runtime source-of-truth for ordinary config remains the typed config snapshot.

## Reopen Conditions

Reopen technical design before implementation if:

- the team decides `NETWORK_*` must become normal typed config after all,
- the fix would require changing the Prometheus metric contract in `internal/infra/telemetry`,
- the chosen network policy label fix requires a broad logging helper redesign,
- implementation uncovers a need for Redis/Mongo runtime adapters rather than guard-only probes.
