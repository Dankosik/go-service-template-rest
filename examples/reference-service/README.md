# Reference feature slice

This runnable example shows one complete, small feature without adding fictional
behavior to the production service:

```text
OpenAPI contract and request validation
  -> strict HTTP handler
  -> article use case
  -> consumer-owned repository port
  -> immutable in-memory adapter
```

It covers both halves of a REST contract: a **public read** and a **protected
write with a request body**.

Run it from the repository root. Choose a throwaway local value; the example
refuses to start without one, so no credential is ever baked into source.

```bash
export REFERENCE_WRITE_TOKEN="$(openssl rand -hex 16)"
```

```bash
go run ./examples/reference-service
```

```bash
curl http://localhost:8080/api/v1/articles/clear-owners
```

```bash
curl -i -X POST http://localhost:8080/api/v1/articles \
  -H "Authorization: Bearer ${REFERENCE_WRITE_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"slug":"owned-writes","title":"Owned writes","summary":"A write needs an identity."}'
```

Without the header the same request returns `401` with a `WWW-Authenticate`
header and a Problem body, because `createArticle` declares `bearerAuth` in the
contract and the validator enforces the declaration.

The important boundaries are visible in the directory layout:

- `api/openapi.yaml` owns the example's wire contract;
- `internal/openapi/` contains generated, drift-checked bindings;
- `internal/article/` owns business types, errors, the use case, and the
  repository port it consumes; the use case keeps unpublished articles hidden;
- `internal/article/memory/` adapts one concrete storage mechanism;
- `internal/httpapi/` maps feature results to contract responses;
- `main.go` only composes owners and manages the example process lifecycle.

`getArticle` is public by design: it returns read-only example content with no
actor or private data. `createArticle` is protected because it changes stored
content, so it declares a security scheme and documents `401` and `403`.
Neither is registered in `cmd/service` and neither changes the production
OpenAPI contract.

**The bearer check here is not an authentication design.** It compares one
process-wide token in constant time so the example can show how a
spec-declared `securityScheme` becomes a runtime check through
`openapi3filter.AuthenticationFunc`. The wiring is worth copying; the
credential model is not. A real service owns identity, rotation,
authorization, and audit.

Two other details are load-bearing rather than decorative:

- validation is declared in the contract *and* re-checked in the use case, so
  the invariant holds for any future caller that does not arrive over HTTP;
- uniqueness is enforced by the adapter, because only the storage layer can
  make the check and the write one atomic step. A real datastore maps its
  unique-constraint violation onto `article.ErrAlreadyExists` instead of
  reading before writing.

When adapting this slice to a real feature:

1. Make the security decision in the production OpenAPI contract first.
2. Put business behavior and consumer-owned ports in `internal/<feature>`.
3. Put the real adapter with its technical owner, such as
   `internal/infra/postgres`.
4. Wire concrete owners only in `cmd/service/internal/bootstrap`.
5. Add generated-contract, use-case, adapter, and real-path tests that falsify
   the feature's actual failure modes.

Do not copy the in-memory adapter when the feature requires durable state, do
not copy the public security decision when the feature has identity,
authorization, private data, or side effects, and do not copy the demonstration
bearer token into anything real.

This directory is removed during `make template-init`. Pass
`REFERENCE_EXAMPLE=keep` to retain it in a generated service.
