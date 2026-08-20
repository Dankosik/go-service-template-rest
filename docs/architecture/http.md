# HTTP Architecture

Load for an HTTP contract, route, middleware, exposure, or handler-composition
change.

1. Bootstrap builds config, telemetry, dependencies, feature services, router,
   and the HTTP server.
2. `internal/infra/http.NewRouter` owns request-ID admission, W3C trace-context
   extraction, security/body guards, recovery, access logging, route labels,
   and OpenTelemetry HTTP instrumentation.
3. The application router contains only the generated client API. `/metrics`
   stays on the separate bootstrap-owned diagnostics listener.
4. HTTP maps the request into the generated handler interface and calls the
   feature package.
5. The feature returns use-case results; HTTP maps them to contract responses or
   RFC 9457 problems from the closed transport catalog.
6. Edge observability uses bounded route templates, configured service identity,
   an explicit OTel cardinality cap, and the private Prometheus registry.

<!-- profile:http-idempotency-postgres:start -->
For `x-idempotent: true`, `internal/httpidempotency` owns the scoped-request and
generated-result contract. `internal/infra/postgresidempotency` binds replay
evidence and the business effect in one PostgreSQL transaction.
<!-- profile:http-idempotency-postgres:end -->

The shipped API is health-only. A new operation first decides whether it is
public, protected by OpenAPI security plus authentication and 401/403 Problem
responses, or blocked pending policy. Browser CORS remains fail-closed.
Diagnostics default to `127.0.0.1:9090`; non-loopback exposure needs a private
scrape network or authenticated design. The deployment edge owns public ingress.

For a new capability, update `api/openapi/service.yaml`, regenerate
`internal/openapi`, implement feature behavior, and satisfy the generated
interface behind `httpx.Handlers.API`. Do not edit shared HTTP infrastructure to
add a service operation. A cross-cutting shared request policy belongs in the
existing `Harden` chain only when it is truly shared; otherwise wrap the
feature-owned handler.
