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

Run it from the repository root:

```bash
go run ./examples/reference-service
curl http://localhost:8080/api/v1/articles/clear-owners
```

The important boundaries are visible in the directory layout:

- `api/openapi.yaml` owns the example's wire contract;
- `internal/openapi/` contains generated, drift-checked bindings;
- `internal/article/` owns business types, errors, the use case, and the
  repository port it consumes; the use case keeps unpublished articles hidden;
- `internal/article/memory/` adapts one concrete storage mechanism;
- `internal/httpapi/` maps feature results to contract responses;
- `main.go` only composes owners and manages the example process lifecycle.

The endpoint is public by design because it returns static, read-only example
content and has no actor, private data, or side effect. It is not registered in
`cmd/service` and does not change the production OpenAPI contract.

When adapting this slice to a real feature:

1. Make the security decision in the production OpenAPI contract first.
2. Put business behavior and consumer-owned ports in `internal/<feature>`.
3. Put the real adapter with its technical owner, such as
   `internal/infra/postgres`.
4. Wire concrete owners only in `cmd/service/internal/bootstrap`.
5. Add generated-contract, use-case, adapter, and real-path tests that falsify
   the feature's actual failure modes.

Do not copy the in-memory adapter when the feature requires durable state, and
do not copy the public security decision when the feature has identity,
authorization, private data, or side effects.
