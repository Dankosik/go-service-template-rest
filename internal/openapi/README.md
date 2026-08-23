# Generated OpenAPI Artifacts

This Codex-native repository keeps generated API guidance beside its owning package.

OpenAPI bindings are generated into this package via:

```bash
go generate ./internal/openapi
```

Generation config: `internal/openapi/oapi-codegen.yaml`.
Current server mode: `chi-server: true` + `strict-server: true`.

## Contract Lifecycle

The provider repository owns `api/openapi/service.yaml` and the matching runtime
behavior. Generated Go is committed and reviewed, but remains derived output.
Treat compatibility as three separate questions:

- wire compatibility for existing HTTP callers;
- generated-source compatibility for committed or published clients;
- semantic compatibility for auth, errors, ordering, retries, defaults, and
  business meaning.

`operationId` is a stable generated-source identifier. An intentional breaking
exception in `api/openapi/breaking-changes-approvals.txt` must be exact,
temporary, and recorded with an owner, migration rationale, deadline, and
consumer-removal evidence in the pull request. Remove the entry after the
approved change merges.

Use OpenAPI `deprecated: true`, RFC 9745 `Deprecation`, and RFC 8594 `Sunset`
only with migration guidance and a removal owner. Keep Git as the distribution
mechanism until a real consumer needs a bundled specification or generated
client outside this repository; then publish immutable versioned artifacts
rather than resolving mutable `main`.

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
2. Put use-case behavior in `internal/<feature>` before adding transport mapping; handlers should call feature behavior instead of owning business logic.
3. Run `make openapi-generate` or `go generate ./internal/openapi`.
4. Confirm the generated `openapi.StrictServerInterface` includes the new operation.
5. Implement the matching business operation in the service's own package as `<feature>_handlers.go`. `internal/infra/http` is shared template surface that adding a business operation does not edit; it owns only platform probes and profile operations that require raw transport access.
6. Inject that implementation as `httpx.Handlers.API` from `cmd/service/internal/bootstrap` instead of adding a manual `/api/...` route.
7. For protected operations, declare real OpenAPI `security`, provide 401/403 `application/problem+json` responses backed by `#/components/schemas/Problem`, and add scoped generated/strict middleware or an explicitly designed equivalent. Do not add broad root middleware that accidentally protects health or public sample routes. Metrics are not part of the application router.
8. Map domain-specific failures to Problem responses at the HTTP boundary; do not leak transport status codes into app use-case behavior. Framework-level failures with a `http.ResponseWriter` use the hand-written `writeProblem` catalog in `internal/infra/http/problem.go`. Generated strict handlers return their operation's generated typed Problem response, such as `*ApplicationProblemPlusJSONResponse`, because the strict interface does not expose a response writer.
9. Add contract/policy tests for status codes, Problem responses, generated-route ownership, security behavior, unauthenticated protected calls, and public-route non-regression.
10. Run `make openapi-check`.

For future parameterized endpoints, also prove that route labels in logs, metrics, and spans use OpenAPI route templates rather than concrete IDs.
