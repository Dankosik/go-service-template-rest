# Component Map

## `internal/observability/otelconfig`

New package.

Owns:

- OTel sampler names such as `always_on`, `always_off`, `traceidratio`, and `parentbased_traceidratio`.
- OTel exporter protocol vocabulary such as `http/protobuf`.
- Defaults for OTel-specific config vocabulary such as the default sampler and default OTLP protocol.
- Pure helpers for normalizing or validating OTel config vocabulary and sample ratios when those helpers do not import OTel SDK packages.

Does not own:

- Loading config from files or environment.
- OTel SDK sampler/provider/exporter construction.
- HTTP request telemetry.
- Bootstrap lifecycle policy.

## `docs/repo-architecture.md` And `docs/project-structure-and-module-organization.md`

Changes:

- Record `internal/observability/otelconfig` as a narrow shared technical vocabulary package for OTel config values.
- Preserve the rule that `internal/config` owns config loading, defaults, snapshot construction, and validation while `internal/infra/telemetry` owns SDK setup.

Stable:

- Do not turn docs into a full list of every constant or config key.

## `internal/config`

Changes:

- Use `internal/observability/otelconfig` constants/helpers in defaults and validation for OTel sampler/protocol values.
- Own resource identity defaults and any validation needed for fields that `SetupTracing` consumes without fallback defaults.
- Preserve existing config key names, typed snapshot shape, and validation error category.

Stable:

- `internal/config` remains the source of truth for runtime config keys, precedence, secret-source policy, and validation.
- No dependency on `internal/infra/telemetry`.

## `internal/infra/telemetry`

Changes:

- Remove `resource.WithFromEnv()`.
- Remove config-owned resource identity fallback defaults from `SetupTracing`.
- Use `internal/observability/otelconfig` for OTel vocabulary in sampler/protocol handling.
- Reject non-finite sampler ratios in `buildTraceSampler`.
- Add intent-named startup dependency metric methods and keep numeric gauge encoding private.
- Add telemetry init failure reason constants used by bootstrap and `normalizeTelemetryFailureReason`.

Stable:

- SDK-specific OTel setup remains here.
- Prometheus instruments and handler behavior remain here.
- Metric names and labels remain unchanged.

## `cmd/service/internal/bootstrap`

Changes:

- Replace raw `SetStartupDependencyStatus(..., true/false)` calls with intent-named telemetry methods.
- Use telemetry-owned init failure reason constants through `telemetryInitFailureReason`.

Stable:

- Bootstrap remains the composition root.
- Startup dependency mode label strings remain bootstrap-owned.
- Telemetry remains optional fail-open.

## `internal/infra/http`

Changes:

- Add an inspectable uninitialized-server error and receiver guard for exported `Server` methods.
- Make manual root route metadata a single local source of truth.

Stable:

- Router behavior, middleware order, Problem responses, `/metrics` ownership, and route labels remain unchanged.

## `internal/infra/postgres`

Changes:

- Remove unused `ping_history` transaction workflow and tests.
- Simplify repository fields/interfaces after removing the transaction-only `db` dependency.

Stable:

- `Create`, `ListRecent`, SQLC query usage, generated code, migrations, and integration-test behavior remain unchanged.
