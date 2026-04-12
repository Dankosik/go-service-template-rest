Objective
Fix the HTTP `OPTIONS` handling so an existing route returns `204 No Content` with the correct `Allow` header, while CORS preflight stays explicitly fail-closed when CORS is not enabled.

User Intent And Context
- The user is not asking for broad router cleanup.
- The important split is:
  - normal `OPTIONS` on a known path should behave like an allowed route-level response
  - actual CORS preflight without CORS enabled should still fail closed
- They also want `problem+json` behavior to remain stable and do not want unnecessary OpenAPI churn if the public contract does not change.

Confirmed Signals And Exact Identifiers
- `OPTIONS`
- `Allow`
- `204`
- `CORS preflight`
- `cors preflight is not enabled`
- `problem json`
- `router_test.go`
- confirmed repo files:
  - `internal/infra/http/router.go`
  - `internal/infra/http/router_test.go`
  - `internal/infra/http/problem.go`
  - `internal/infra/http/openapi_contract_test.go`

Relevant Repository Context
- HTTP transport policy lives in `internal/infra/http/`.
- `router.go` currently handles `OPTIONS` inside `root.MethodNotAllowed`.
- `problem.go` owns the stable `application/problem+json; charset=utf-8` response shape.
- `router_test.go` already has targeted coverage for:
  - method-not-allowed responses
  - `OPTIONS` on known paths
  - CORS preflight fail-closed behavior
- OpenAPI is source-of-truth in `api/openapi/service.yaml`; only touch it if the contract actually changes.

Inspect First
- `internal/infra/http/router.go`
- `internal/infra/http/router_test.go`
- `internal/infra/http/problem.go`
- `internal/infra/http/openapi_contract_test.go` only if contract/runtime alignment looks affected

Requested Change / Problem Statement
- The router’s `OPTIONS` handling appears to be mixing normal route handling with CORS preflight handling.
- Make sure an existing path returns `204` plus the correct `Allow` header for ordinary `OPTIONS`.
- Keep the explicit fail-closed branch for real CORS preflight when CORS is not enabled.
- Preserve `problem+json` status/body behavior unless the fix truly requires a contract change.

Constraints / Preferences / Non-goals
- Do not churn OpenAPI unless the public contract changes.
- Keep `problem+json` stable.
- Prefer a focused router-policy fix with regression tests over broader refactoring.
- Preserve the current fail-closed CORS posture.

Acceptance Criteria / Expected Outcome
- `OPTIONS` on a known route returns `204 No Content`.
- The response includes the correct single `Allow` header.
- CORS preflight without enabled CORS still returns the intended fail-closed `problem+json` response.
- Unknown-path `OPTIONS` behavior remains `404` with the existing problem envelope.
- Tests clearly cover both the normal `OPTIONS` path and the CORS preflight branch.

Validation / Verification
- Run focused tests in `./internal/infra/http`, especially `router_test.go`.
- Only run `make openapi-check` if the change actually alters contract-visible behavior or generated API alignment.

Assumptions / Open Questions
- [assumption] The issue is localized to router policy and test coverage, not generated OpenAPI bindings.