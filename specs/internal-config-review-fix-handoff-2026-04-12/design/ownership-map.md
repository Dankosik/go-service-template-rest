# Ownership Map

## Source Of Truth

- Typed config shape: `internal/config/types.go`
- Accepted config keys: typed `Config` `koanf` tags after the fix
- Code defaults: `internal/config/defaults.go`
- Snapshot construction: `internal/config/snapshot.go`
- Validation and probe-address normalization: `internal/config/validate.go`
- Config source policy: `docs/configuration-source-policy.md`
- YAML baseline defaults: `env/config/default.yaml`
- Env examples and secret placeholders: `env/.env.example`

## Boundary Rules

- `internal/config` owns config loading, parsing, validation, and deterministic guard-only derived values such as `MongoProbeAddress`.
- `cmd/service/internal/bootstrap` consumes the validated snapshot and owns composition, startup, shutdown, dependency admission, and metrics/logging.
- `internal/infra/mongo` or `internal/infra/redis` should own real runtime adapters if those are introduced later.
- Tests may use reflection helpers to prove source-of-truth alignment, but production code should not depend on test-only helpers.

## Change Ownership

- Empty env semantics are a configuration policy decision; update both code and policy docs together.
- Known-key registry ownership moves from defaults to typed schema; update tests so they enforce the new ownership instead of the old accidental equality.
- YAML drift is fixed in `env/config/default.yaml`; `.env.example` already contains `APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_TRACES_ENDPOINT`.

## Reopen Conditions

Reopen specification or design before coding if:

- Empty env values should continue to mean “unset” for deployment compatibility.
- A full Mongo URI parser is desired instead of guard-only probe-address extraction.
- `ErrorType` fallback needs a new metric label contract.
- The implementation starts touching bootstrap lifecycle, telemetry metric names, or runtime dependency adapters.
