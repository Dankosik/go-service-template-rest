# Sequence Design

## Startup / Public Ingress

1. `cmd/service/main.go` delegates to `bootstrap.Run`.
2. Bootstrap loads the config snapshot through `internal/config`.
3. Bootstrap loads network policy from environment.
4. Bootstrap checks whether `app.env` is non-local and `http.addr` is a wildcard bind.
5. If public ingress declaration is missing in that situation, startup fails with a sanitized configuration/policy error.
6. If public ingress is declared true, the existing exception metadata path applies.
7. If public ingress is declared false, startup may continue, with docs making clear that the operator is asserting private ingress.

## Readiness

1. Startup dependency admission initializes dependencies and registers readiness probes.
2. The internal startup readiness check can verify dependencies before external admission is marked ready.
3. External `/health/ready` checks startup admission first, then ingress policy and dependency readiness.
4. `internal/app/health.Service.Ready` keeps sequential probe execution unless implementation deliberately reopens design for parallel probing.
5. Config validation checks aggregate sequential readiness probe budgets against `http.readiness_timeout`.
6. `/health/live` remains process-only and does not call external dependencies.

## Request Path

1. Request enters root HTTP middleware stack.
2. Request correlation, security headers, request framing/body guards, access logging, recovery, route labeling, metrics, and tracing remain owned by `internal/infra/http`.
3. Normal API routes go through the generated chi server and strict handler.
4. Generated wrapper errors and strict request errors both use the same sanitized malformed-request Problem response path.
5. Handler methods call app services and return generated response objects.
6. `/metrics` remains the documented root-router exception for streaming behavior.

## Panic / Error Redaction

1. If a handler panics, `Recover` logs request/trace context and panic class/type only.
2. The raw recovered value is not logged.
3. Client response remains a generic internal-server-error Problem response.
4. OTLP malformed header parsing returns an error that does not contain the raw malformed value.

## OpenAPI Security Decision Check

1. OpenAPI contract tests load `api/openapi/service.yaml`.
2. Every operation must include an explicit security decision marker.
3. Public operations must be intentionally marked public or operational-public/private-required.
4. Protected operations must use a real security scheme and declare 401/403 Problem responses.
5. Missing or inconsistent security decisions fail the contract test.

## Config Key Drift Check

1. A package-local config test derives expected leaf keys from `Config` `koanf` tags.
2. The test compares derived keys with default/known config keys.
3. A new default key that is not represented in `Config`, or a new `Config` key that is not defaulted, fails fast.
4. Runtime parsing stays explicit and non-reflective.
