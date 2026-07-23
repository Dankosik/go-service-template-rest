# Generated API Artifacts

This Codex-native repository keeps generated API guidance beside its owning package.

OpenAPI bindings are generated into this package via:

```bash
go generate ./internal/api
```

Generation config: `internal/api/oapi-codegen.yaml`.
Current server mode: `chi-server: true` + `strict-server: true`.

Strict-server generation provides typed request/response glue; it does not install
full runtime OpenAPI schema or security validation. The generated operations are
wrapped once by `oapi-codegen/nethttp-middleware`, which enforces path, query,
JSON body, and unknown-field validation from the embedded spec at runtime.
Protected routes still require a real `AuthenticationFunc`; the template
deliberately provides no placeholder auth, and no placeholder operations may be
left in the spec: replace the public security decision with a real OpenAPI
security requirement, an actual `AuthenticationFunc`, 401/403 Problem responses,
and authorization tests before shipping a protected route. The first operation
that accepts parameters or a body must add boundary contract tests (happy path
plus invalid path/query/body/unknown-field) — the health operations cannot
exercise request validation.

## Adding A Strict-Server Endpoint

Protected operations require a real security design before coding. Do not add placeholder auth, broad root middleware, or test-only identity plumbing as a shortcut; choose public-by-design, protected-by-real-policy, or blocked-pending-security-spec first.

1. Change `api/openapi/service.yaml`; do not hand-edit generated Go.
2. Put use-case behavior in `internal/app/<feature>` before adding transport mapping; handlers should call app behavior instead of owning business logic.
3. Run `make openapi-generate` or `go generate ./internal/api`.
4. Confirm the generated `api.StrictServerInterface` includes the new operation.
5. Implement the matching `strictHandlers.<Operation>` method in `internal/infra/http`; split handler files by feature when one file stops being readable.
6. Wire the handler through the existing `Handlers` construction instead of adding a manual `/api/...` route.
7. For protected operations, declare real OpenAPI `security`, provide 401/403 `application/problem+json` responses backed by `#/components/schemas/Problem`, and add scoped generated/strict middleware or an explicitly designed equivalent. Do not add broad root middleware that accidentally protects health, metrics, or public sample routes.
8. Map domain-specific failures to Problem responses at the HTTP boundary; do not leak transport status codes into app use-case behavior. All Problem bodies are written by the hand-written `writeProblem` catalog in `internal/infra/http/problem.go`; do not construct the generated per-operation `*ApplicationProblemPlusJSONResponse` types — they exist only as contract artifacts of the spec.
9. Add contract/policy tests for status codes, Problem responses, generated-route ownership, security behavior, unauthenticated protected calls, and public-route non-regression.
10. Run `make openapi-check`.

For future parameterized endpoints, also prove that route labels in logs, metrics, and spans use OpenAPI route templates rather than concrete IDs.
