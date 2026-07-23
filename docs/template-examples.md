# Template Examples

Search the repository for `TEMPLATE EXAMPLE`. Every match is intentionally
synthetic and must have exactly one outcome before production:

1. Delete the example and its generated/test artifacts when the service does
   not need that capability.
2. Replace it with feature-owned business behavior, keep the useful
   infrastructure, and remove the marker.

There is no permanent placeholder state.

## Included Examples

- `POST /api/v1/template-example/{slug}` proves OpenAPI path, query, JSON body,
  and unknown-field validation through `oapi-codegen/nethttp-middleware`.
  It is public by design; the template does not ship fake authentication.
- `template_example.sql` generates a real sqlc query. `Pool.InTx` binds the
  generated `Queries` to one pgx transaction through `pgx.BeginFunc`, and the
  container integration test proves both calls observe the same transaction.

## Removal Or Replacement

For the HTTP example, remove or replace the operation and schemas in
`api/openapi/service.yaml`, the marked strict handler and HTTP tests, then
run `make openapi-generate`. Remove `nethttp-middleware` only when no real
operation needs schema or security validation.

For the SQL example, remove or replace
`internal/infra/postgres/queries/template_example.sql` and its marked
integration test, then run `make sqlc-generate`. Remove `Pool.InTx` only when
the service has no transactional use case. Do not add a sample table or retain
an artificial migration just to keep the example alive.

For a protected operation, replace the public security decision with a real
OpenAPI security requirement, an actual `AuthenticationFunc`, 401/403 Problem
responses, and authorization tests. Never turn the example into an
accept-any-token authenticator.
