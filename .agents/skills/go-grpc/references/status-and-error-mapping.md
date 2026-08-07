# Status And Error Mapping

## When To Load

Load this when a gRPC handler, policy interceptor, or dependency has to publish
a failure — a new error path, a status that looks wrong to a caller, or a claim
about which code and detail the client observes.

## Behavior Change Thesis

Without this file, a handler publishes its failure the way every gRPC codebase
does: `return nil, status.Error(codes.NotFound, "widget not found")`. The client
observes `INTERNAL` with detail `request failed`. `errorMappingAround` is the
innermost policy in `internal/infra/grpc/interceptors.go`, and only
four inputs escape its collapse: `context.Canceled`, `context.DeadlineExceeded`,
an `*ownedStatusError` (unexported, so no handler can construct one), and an
error some registered `problem.Mapper` classifies. Everything else becomes
`INTERNAL`. The repository proves this deliberately: the
`unmarked downstream` case in `interceptors_test.go` asserts that
`status.Error(codes.PermissionDenied, "dependency secret")` reaches the client
as `Internal` / `request failed`. A handler test asserting only that an error
came back cannot see the collapse.

## Decision Rubric

- Publish a non-`INTERNAL` failure by returning a domain error and registering a
  `problem.Mapper` — `runtimeDependencies.DomainErrors()` in
  `cmd/service/internal/bootstrap/startup_dependencies.go` is the seam, and
  `classifyPostgresDomainError` beside it is the shape. The status is chosen by
  the `problem.Code`, not by the handler.
- `mappedStatus` fixes the projection: `BadRequest`/`UnprocessableContent` →
  `InvalidArgument`, `Unauthorized` → `Unauthenticated`, `Forbidden` →
  `PermissionDenied`, `NotFound` → `NotFound`, `MethodNotAllowed` →
  `Unimplemented`, `Conflict` → **`Aborted`**, `RequestEntityTooLarge` and
  `TooManyRequests` → `ResourceExhausted`, `ServiceUnavailable` → `Unavailable`,
  `GatewayTimeout` → `DeadlineExceeded`. A mapper that leaves `Detail` empty
  publishes the catalog `Title`. A failure needing a code this table cannot
  produce is a problem-catalog decision, not a handler decision.
- Interceptors supplied through `Options.UnaryPolicy` / `Options.StreamPolicy`
  sit outside `mapError` and inside `policyErrorBoundary`, which preserves any
  error implementing `GRPCStatus()`. An authentication or authorization
  interceptor's `status.Error(codes.PermissionDenied, …)` therefore does reach
  the client verbatim — its detail is already public. The same call inside a
  handler is erased. State which side of that boundary a new rejection lives on.
- Methods under `/grpc.health.v1.Health/` bypass both admission and error
  mapping, so standard health keeps its own semantics — an unknown service stays
  `NOT_FOUND` rather than being rewritten by application mapping.
- `examples/grpc-reference-service` returns raw `status.Error(codes.ResourceExhausted, …)`
  and its test registers the service on a bare `grpc.NewServer`. Composed through
  `grpcx.NewServer` — as `cmd/benchmark-server` does, with no `DomainErrors` — the
  same handler answers `INTERNAL`. Copy its ownership shape, not its status
  handling.

## Reject

- Adding a "preserve any `GRPCStatus()`" branch to `mapError` so a handler status
  survives. That branch is the boundary stopping a dependency's or a generated
  client's status text from becoming this service's public detail; the panic
  recovery and raw-error cases rely on the same default.

## Validation Shape

`go test -vet=off ./internal/infra/grpc` — a new mapping is proven in the
`mapError` table test, and an end-to-end code assertion belongs in
`server_test.go` against a composed server. Proving the mapper alone leaves the
registration in `DomainErrors()` unproven, which is the half that silently
returns `INTERNAL` in production.
