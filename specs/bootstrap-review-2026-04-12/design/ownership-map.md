# Ownership Map

## Source Of Truth

- OTLP endpoint precedence and endpoint parsing: `internal/infra/telemetry`.
- Network egress admission policy: `cmd/service/internal/bootstrap`.
- Config value source and secret policy: `internal/config` plus `docs/configuration-source-policy.md`.
- Startup dependency labels and stage vocabulary: `cmd/service/internal/bootstrap`.
- Config load stage duration mapping for bootstrap metrics/spans: `cmd/service/internal/bootstrap`, derived from `config.LoadReport`.

## Dependency Direction

Allowed:
- `cmd/service/internal/bootstrap` imports `internal/infra/telemetry` and asks it for a pure target description.
- `cmd/service/internal/bootstrap` applies `networkPolicy.EnforceEgressTarget`.
- `internal/infra/telemetry` continues importing `internal/observability/otelconfig`.

Not allowed:
- `internal/infra/telemetry` must not import bootstrap or parse `NETWORK_*`.
- `internal/config` should not become responsible for bootstrap network-policy decisions.
- Bootstrap should not reimplement OTLP URL parsing rules that telemetry already owns.

## Reopen Conditions

Reopen specification/design if:
- the telemetry helper cannot describe the exact target that `otlptracehttp.New` will use;
- OTel SDK environment fallback still changes target after bootstrap admission;
- security/reliability policy requires telemetry egress denial to reject startup rather than degrade tracing;
- preserving Postgres parser cause safely requires a wider redaction utility shared beyond bootstrap.
