**Objective**  
Investigate and fix the HTTP router/handler `OPTIONS` behavior so existing paths return the normal `Allow` header with `204`, while CORS preflight still fails closed when CORS is not enabled.

**User Intent And Context**  
The user is pointing at a regression or confusing branch in the HTTP layer, specifically `OPTIONS`. The important split is:
- For an existing route, return the standard `Allow` handling and `204`.
- For a real CORS preflight request, keep the current fail-closed behavior when CORS is disabled.

They repeated that distinction, so preserve it exactly and do not collapse the two cases into one generic `OPTIONS` response.

**Confirmed Signals And Exact Identifiers**  
- `OPTIONS`
- `Allow` header
- `204`
- `CORS preflight`
- `CORS is not enabled`
- `problem json`
- `router_test.go`
- `openapi`

**Relevant Repository Context**  
This repo is an AI-native Go REST service template using `chi` for HTTP routing. The HTTP surface lives under `internal/infra/http/`, and the API contract source of truth is `api/openapi/service.yaml`. Generated bindings are under `internal/api/`.  
For this task, the most relevant validation path is focused HTTP package tests; OpenAPI drift checks should only be touched if the public contract actually changes.

**Inspect First**  
- `internal/infra/http/`
- `internal/infra/http/router_test.go` if present, or the nearest router/handler tests
- `api/openapi/service.yaml` only if the response contract truly changes
- `internal/api/README.md` if generated contract behavior needs to be confirmed

**Requested Change / Problem Statement**  
Find the router/handler path that distinguishes plain `OPTIONS` handling from CORS preflight. Adjust the logic so an existing route gets the normal HTTP `Allow` response with `204`, but an actual CORS preflight still fails closed unless CORS is enabled.

**Constraints / Preferences / Non-goals**  
- Do not churn OpenAPI unless the public contract changes.
- Keep `problem json` stable.
- Prefer the smallest behavioral fix that preserves the existing contract.
- Do not broaden this into a general router refactor.

**Acceptance Criteria / Expected Outcome**  
- `OPTIONS` on an existing path returns `204` and the expected `Allow` header.
- CORS preflight behavior remains fail-closed when CORS is disabled.
- Existing `problem json` behavior does not drift.
- Tests clearly cover the distinction between plain `OPTIONS` and CORS preflight.

**Validation / Verification**  
- Run the focused HTTP tests around the router/handler surface, likely `go test` in `./internal/infra/http`.
- Add or update the nearest router test, likely `router_test.go`.
- Only run `make openapi-check` if the change genuinely alters the contract.

**Assumptions / Open Questions**  
- `router_test.go` is only a likely starting point, not a confirmed file.
- The exact branch causing the bad `OPTIONS` response is not specified, so inspect the HTTP router/handler implementation first.