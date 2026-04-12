# Component Map

## `cmd/service/internal/bootstrap`

Expected changes:
- `startup_bootstrap.go`: pass a network-policy-aware telemetry admission step into tracing setup without broad startup rewiring.
- `network_policy_*`: likely no policy behavior changes; tests may add OTLP-specific admission cases if helpful.
- `startup_common.go`: introduce one helper for config load stage durations and use it from metrics/span recording.
- `startup_probe_addresses.go`: preserve redacted Postgres DSN parse diagnostics or add secret-safe proof/commenting.
- `startup_dependency_labels.go` and `startup_dependencies.go`: clarify probe stage field roles or collapse stage/span naming.
- Related tests in the same package.

Stable:
- `Run` remains the service composition root.
- Dependency criticality and readiness behavior stay unchanged.
- Existing network-policy semantics stay unchanged except that telemetry now consumes them.

## `internal/infra/telemetry`

Expected changes:
- `tracing.go`: extract one target-description helper that shares endpoint selection with `buildTraceExporterOptions`.
- `tracing.go`: align exporter-configured behavior so headers alone cannot create an uninspected SDK-default/env endpoint.
- `tracing_test.go`: cover explicit `OTLPTracesEndpoint` precedence, scheme-less HTTP behavior, headers-without-endpoint behavior, and invalid endpoint parsing.

Stable:
- Telemetry owns OTel SDK option construction.
- Telemetry must not import bootstrap or know `NETWORK_*`.
- Existing sampler and resource identity behavior remains unchanged.

## `internal/config`

No expected production changes. Config validation may remain as-is unless implementation chooses to reject headers without endpoint at config-load time; the preferred plan is to keep this in telemetry fail-open setup to avoid turning optional tracing mistakes into startup-critical config validation.

## Docs

No broad docs update is required. A small code comment may be useful where the design preserves redacted DSN parse diagnostics or disables header-only exporter setup.
