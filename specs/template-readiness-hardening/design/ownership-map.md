# Ownership Map

## OpenAPI Guard Ownership

- `Makefile` owns command composition and test selection.
- `internal/infra/http/openapi_contract_test.go` owns HTTP/OpenAPI runtime contract guardrails.
- Selected ownership decision: the security-decision guard should identify as an OpenAPI runtime-contract guard by test name.

## Endpoint/Auth Guidance Ownership

- `api/openapi/service.yaml` owns endpoint contract and real OpenAPI security declarations when protected endpoints exist.
- `internal/api/README.md` owns generated API usage guidance for endpoint authors.
- `internal/infra/http` owns transport mapping, scoped middleware, Problem response writing, route labels, and generated-route integration.
- `internal/app/<feature>` owns business/use-case behavior.

## Ping History Sample Ownership

- `internal/infra/postgres` owns sample repository validation and generated-row mapping.
- `env/migrations` owns schema; no schema change is needed for a sample limit.
- `internal/app` remains the owner for real future app-facing persistence ports and business records; the current infra-owned `PingHistoryRecord` stays sample-local only.

## Redis Policy Ownership

- `internal/config` owns Redis mode normalization and readiness policy derived from the immutable config snapshot.
- `cmd/service/internal/bootstrap` owns startup dependency probing, startup telemetry labels, criticality labels, and cleanup.
- Selected ownership decision: bootstrap must consume config-owned Redis policy rather than independently re-deriving store/cache mode.

## Avoided Ownership Moves

- Do not move auth decisions into generated code.
- Do not move SQLC generated types into app code.
- Do not create a cross-feature port package.
- Do not move Redis runtime behavior into config or bootstrap beyond guard/probe policy.
