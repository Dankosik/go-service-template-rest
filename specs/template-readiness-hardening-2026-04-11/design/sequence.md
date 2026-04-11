# Sequence Design

## Future Endpoint Addition After Hardening

1. Update `api/openapi/service.yaml`.
2. Decide endpoint security explicitly:
   - public by design,
   - protected by a real auth middleware/contract,
   - or blocked pending security spec.
3. Run OpenAPI generation.
4. Add app behavior in `internal/app/<feature>`.
5. Add an app-owned port/interface beside the consuming app package when the app needs an adapter.
6. Add infra adapter under `internal/infra/<integration>` or `internal/infra/postgres`.
7. Wire concrete dependencies in `cmd/service/internal/bootstrap`.
8. Implement generated strict handler mapping in `internal/infra/http`.
9. Add layer-appropriate tests.
10. Run targeted Make validation.

## Startup And Readiness After Hardening

1. Bootstrap loads and validates config.
2. Bootstrap initializes telemetry and dependency admission.
3. Dependency admission returns runtime handles, readiness probes, statuses, and cleanup callbacks together.
4. Bootstrap builds app services and HTTP handlers with explicit dependencies.
5. HTTP server may begin serving, but external readiness remains false until startup admission succeeds.
6. Startup admission runs dependency/app readiness without depending on `admission.Ready()`.
7. `/health/ready` checks the external readiness gate and then app health probes.
8. On shutdown, bootstrap flips draining/readiness off, waits configured propagation delay, shuts down HTTP, cleans up runtime handles, and flushes telemetry within documented process budget.

## Generated Route Ownership

1. Normal routes are registered by generated OpenAPI chi server.
2. Root-router exceptions are explicitly listed.
3. Route-owner tests reject accidental manual/generated overlap.
4. `/metrics`, if retained as a root exception, is documented as operational/private-scrape behavior.

## Persistence Flow

1. Schema changes start in deterministic migrations.
2. Query files live under `internal/infra/postgres/queries`.
3. Generated sqlc code lives under `internal/infra/postgres/sqlcgen`.
4. Hand-written repositories wrap generated rows/types before exposing adapter-facing records or app-owned ports.
5. Business app packages consume ports/contracts, not `sqlcgen`.
6. Integration tests prove migration-backed behavior only when the change touches schema/runtime DB behavior.

