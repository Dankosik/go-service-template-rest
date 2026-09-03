# Status And Error Mapping

## Load When

Load this when a gRPC handler, policy interceptor, or dependency has to publish
a failure — a new error path, a status that looks wrong to a caller, or a claim
about which code and detail the client observes.

## Decide

- Publish a non-`INTERNAL` feature failure by returning the feature error and
  registering its `failure.Mapper` in the local `domainErrors` slice in
  `cmd/service/internal/bootstrap/run.go`. That slice is passed to both HTTP and
  gRPC; the handler does not choose a transport status.
- `mapHandlerError` classifies through `failure.Classify`, then `mappedStatus`
  owns the gRPC projection. In particular, `failure.CodeAlreadyExists` maps to
  `codes.AlreadyExists`, not `codes.Aborted`. A mapper that leaves `Detail`
  empty publishes `failure.SanitizedDetail`. Read the current switch before
  changing another code; a failure it cannot project remains `INTERNAL`.
- Interceptors supplied through `Options.UnaryPolicy` / `Options.StreamPolicy`
  sit outside `mapHandlerError` and inside `policyErrorBoundary`, which preserves any
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

- Adding a "preserve any `GRPCStatus()`" branch to `mapHandlerError` so a handler status
  survives. That branch is the boundary stopping a dependency's or a generated
  client's status text from becoming this service's public detail; the panic
  recovery and raw-error cases rely on the same default.

## Prove

`go test -vet=off ./internal/infra/grpc` — a new mapping is proven in the
`mapHandlerError` table test, and an end-to-end code assertion belongs in
`server_test.go` against a composed server. Proving the mapper alone leaves its
append to the bootstrap `domainErrors` slice unproven, which is the half that
silently returns `INTERNAL` in production.
