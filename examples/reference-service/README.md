# Reference feature slice

This example shows one complete, small feature without adding fictional
behavior to the production service:

```text
OpenAPI contract and request validation
  -> strict HTTP handler
  -> article use case
  -> consumer-owned store port
  -> transactional in-memory adapter
```

It covers both halves of a REST contract: a **public read** and a **protected
write with a request body**.

`referenceservice` is a library package, not a `main` package — there is no
process here to run. The composition is proved by exercising `NewHandler` over
`httptest`, the same way `go build` would fail to prove it:

```bash
go test ./examples/reference-service/...
```

`TestReferenceServiceServesOverHTTP` and the other top-level tests in
`reference_test.go` start a real `httptest.Server` from `NewHandler` and drive
it with HTTP requests: a public `GET`, a bearer-protected `POST`, an oversized
body, a missing credential, and a panic in feature code — proving the example
inherits the shared hardened chain rather than merely compiling against it.
`internal/httpapi/router_test.go` covers the same contract one layer down,
against `NewAPIHandler` directly.

The important boundaries are visible in the directory layout:

- `api/openapi.yaml` owns the example's wire contract;
- `internal/openapi/` contains generated, drift-checked bindings;
- `internal/article/` owns business types, errors, the use case, and the
  store port it consumes; the use case keeps unpublished articles hidden;
- `internal/article/memory/` adapts one concrete storage mechanism;
- `internal/httpapi/` maps feature results to contract responses;
- `reference.go` is the composition root: it wires the feature onto its
  generated contract and the shared `httpx.Harden` middleware chain, and owns
  the demonstration bearer credential. It deliberately owns no process
  lifecycle — no `main`, listener, signal handling, or shutdown.

`reference.go` is the only non-test file in this example that may import
`internal/infra/*` (here, `httpx` and `telemetry`). That is not a convention;
it falls out of where the file lives. `.golangci.yml`'s
`feature_packages_no_adapters` rule matches `**/internal/**/*.go` and denies
those files from importing `internal/infra`, `internal/config`, or the
root-level `internal/openapi`. `reference.go` sits directly under
`examples/reference-service/`, outside any `internal/` directory, so the rule
never matches it. `internal/httpapi/`, `internal/article/`, and
`internal/article/memory/` all live under `internal/` and are matched, so
`internal/httpapi` cannot import an infra adapter directly — instead
`NewAPIHandler` takes `RejectFunc` values (`Authenticate`, `RejectRequest`,
`RejectResponse`) that the composition root builds from `httpx` and passes in.
This is the lesson the example exists to teach: adapters get wired at a
composition root outside `internal/`, and feature packages depend on the
functions and interfaces that root supplies, not on the adapters themselves.

Inside `internal/article/` the types, the ports, and both use cases share one
`article.go`, which is the one place this example deliberately stops short of
the layout [Project Structure And Module
Organization](../../docs/project-structure-and-module-organization.md#4-file-naming-and-granularity)
prescribes. That document's `model.go` / `repository.go` / `<verb>_<noun>.go`
split earns its place when a slice has enough use cases that a reader has to
search for one; at two, splitting costs more navigation than it saves. Copy this
file as a starting point, and split it the documented way as soon as a third use
case arrives — the boundaries above are the ones worth preserving from the
start, not this one.

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
