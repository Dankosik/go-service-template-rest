# Generated API Artifacts

OpenAPI bindings are generated into this package via:

```bash
go generate ./internal/api
```

Generation config: `internal/api/oapi-codegen.yaml`.
Current server mode: `chi-server: true` + `strict-server: true`.

## Adding A Strict-Server Endpoint

1. Change `api/openapi/service.yaml`; do not hand-edit generated Go.
2. Run `make openapi-generate` or `go generate ./internal/api`.
3. Confirm the generated `api.StrictServerInterface` includes the new operation.
4. Implement the matching `strictHandlers.<Operation>` method in `internal/infra/http`.
5. Wire the handler through the existing `Handlers` construction instead of adding a manual `/api/...` route.
6. Add contract/policy tests for status codes, Problem responses, generated-route ownership, and security behavior.
7. Run `make openapi-check`.

For future parameterized endpoints, also prove that route labels in logs, metrics, and spans use OpenAPI route templates rather than concrete IDs.
