# Component Map

## `internal/config/load_koanf.go`

Changes expected:

- Update `collectNamespaceValues` so empty `APP__...` values are collected instead of skipped.
- Preserve namespace filtering and invalid env-key handling.
- Keep `lookupNonEmptyEnv` unchanged for environment intent and allowed-root control values.
- Replace `localEnvironment bool` through the file-loader call chain with a package-local named policy mode if touching that path:
  - `configFilePolicyLocal`
  - `configFilePolicyHardened`

Stable behavior:

- Config file ordering remains defaults, base file, overlays, then env.
- Non-local file hardening remains absolute path, allowed roots, no symlinks, not group/other writable, and size bounded.

## `internal/config/validate.go`

Changes expected:

- Harden `normalizeMongoProbeAddress` around empty hosts and bracket handling.
- Preserve current valid host behavior and default port policy.

Stable behavior:

- Redis and Mongo remain guard-only config validation helpers.
- Validation still wraps user-facing config failures with `ErrValidate` or `ErrSecretPolicy` as appropriate.

## `internal/config/defaults.go` and possible `internal/config/schema.go`

Changes expected:

- Stop deriving `knownConfigKeys()` from `defaultValues()`.
- Prefer a package-local schema helper derived from `Config` `koanf` tags.
- Keep `defaultValues()` focused on baseline default values.

Stable behavior:

- Existing default values remain unchanged.

## `internal/config/config_test.go`

Changes expected:

- Add Mongo malformed-host regression tests.
- Add empty env override tests.
- Replace `resetConfigEnv` empty-value cleanup with true unset/restore behavior so tests do not depend on empty env being skipped.
- Update known-key/default/tag tests so typed keys are the accepted-key registry and defaults are a subset.
- Add YAML baseline coverage for `env/config/default.yaml` versus `defaultValues()` where appropriate.
- Optionally update file-policy tests to use named policy constants instead of raw booleans.

## `env/config/default.yaml`

Changes expected:

- Add `observability.otel.exporter.otlp_traces_endpoint: ""` beside the other OTLP exporter keys.

## `docs/configuration-source-policy.md`

Changes expected:

- Clarify that `APP__...` entries are explicit overrides even when their value is empty, and invalid empty values are rejected by parsing or validation.

## Surfaces Not Expected To Change

- `cmd/service/internal/bootstrap/*`
- `internal/infra/*`
- `internal/app/*`
- OpenAPI, generated API, migrations, or telemetry metric definitions
