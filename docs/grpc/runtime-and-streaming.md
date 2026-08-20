# Runtime And Streaming

Load for registration, status mapping, client, or streaming behavior.

## Implement the generated server

Keep business behavior in `internal/<feature>`. The gRPC adapter maps generated
messages to that behavior and embeds the generated unimplemented server:

```go
type Server struct {
    ordersv1.UnimplementedOrdersServiceServer
    service *orders.Service
}

func (s *Server) GetOrder(
    ctx context.Context,
    request *ordersv1.GetOrderRequest,
) (*ordersv1.GetOrderResponse, error) {
    order, err := s.service.Get(ctx, request.GetId())
    if err != nil {
        return nil, err
    }
    return ordersv1.GetOrderResponse_builder{
        Id: new(order.ID),
    }.Build(), nil
}
```

The Opaque API intentionally avoids direct mutation of generated fields. Read
with getters and construct with generated builders or setters. Return
feature/domain errors rather than raw `status.Error`: the shared transport
maps known domain identities to sanitized gRPC codes and converts unknown
errors to `INTERNAL` without exposing their text.

A handler therefore does not choose its own `codes.Code`. Its feature returns a
domain error classified as a transport-neutral `failure.Code`, and the transport
renders it. That indirection is what keeps one
domain error answering consistently over both HTTP and gRPC, and it fixes the
reachable vocabulary:

| `failure.Code` | gRPC code |
| --- | --- |
| `CodeBadRequest`, `CodeUnprocessableContent` | `InvalidArgument` |
<!-- profile:http-idempotency-postgres:start -->
| `CodeIdempotencyKeyMismatch` | `InvalidArgument` |
| `CodeIdempotencyUnavailable`, `CodeIdempotencyOutcomeUnknown` | `Unavailable` |
<!-- profile:http-idempotency-postgres:end -->
| `CodeUnauthorized` | `Unauthenticated` |
| `CodeForbidden` | `PermissionDenied` |
| `CodeNotFound` | `NotFound` |
| `CodeMethodNotAllowed` | `Unimplemented` |
| `CodeAlreadyExists` | `AlreadyExists` |
| `CodeRequestEntityTooLarge`, `CodeTooManyRequests` | `ResourceExhausted` |
<!-- profile:authn-oidc-jwt:start -->
| `CodeRequestHeaderFieldsTooLarge` | `ResourceExhausted` |
<!-- profile:authn-oidc-jwt:end -->
| `CodeServiceUnavailable` | `Unavailable` |
| `CodeGatewayTimeout` | `DeadlineExceeded` |
| `CodeInternalError`, unclassified | `Internal` |

A classified error additionally carries structured details, so a caller reads its
class and its retry hint as data rather than parsing prose:

- `google.rpc.ErrorInfo` — `Reason` is the `failure.Code` upper-snake-cased
  (`CodeNotFound` reaches a caller as `NOT_FOUND`), and `Domain` is the service's
  own identity, taken from `APP__OBSERVABILITY__OTEL__SERVICE_NAME`. Renaming
  that key for telemetry reasons therefore changes a value remote callers match
  on. The reason matters because the gRPC code space is coarser than
  `failure.Code`: `InvalidArgument` answers for both `CodeBadRequest` and
  `CodeUnprocessableContent`, and `ResourceExhausted` for three more.
- `google.rpc.RetryInfo` — the mapper's own `RetryAfter`, exactly, when it is
  positive. HTTP renders the same value as a `Retry-After` header rounded up to
  whole seconds, which is that header's own granularity; neither transport ever
  advertises a delay shorter than the mapper's.

Both carry repository-owned values only. Cancellation, expiry, an unclassified
error, and a sanitized handler status carry no details at all, for the same
reason they carry no handler text.

HTTP retains its transport-only `conflict` fallback for a generic 409 response.
It is not a domain classification and therefore cannot enter the gRPC mapping;
a known creation collision is `already_exists` on both transports.

A canceled or expired RPC context answers `CANCELED` or `DEADLINE_EXCEEDED`
before classification runs, because that is the caller's own signal rather than a
service outcome. That precedence is total here, so a mapper cannot reclassify an
error that still wraps `context.Canceled` or `context.DeadlineExceeded` — a
dependency failure meant to reach the caller as `UNAVAILABLE` must not carry one
of those in its chain. Report the dependency's own identity instead of wrapping
the context error. HTTP applies the same precedence to an expired context, on
the reason its own owner records, so a mapper written against either transport
already has to obey it.

The gRPC codes this table cannot produce:

- `FailedPrecondition`
- `Aborted`
- `OutOfRange`
- `DataLoss`

Needing one is a contract decision that extends `internal/failure`, `problem`,
and `mappedStatus` together, not a local `status.Error` in a handler — which the
handler error boundary sanitizes anyway. This repository publishes no code
without a producer, so the identity arrives with the first feature that requires
a state change or a transaction retry rather than ahead of it.

Current generated streaming APIs are generic:

```go
func (s *Server) WatchOrders(
    request *ordersv1.WatchOrdersRequest,
    stream ordersv1.OrdersService_WatchOrdersServer,
) error {
    for update := range s.service.Watch(stream.Context(), request.GetCursor()) {
        if err := stream.Send(toProto(update)); err != nil {
            return err
        }
    }
    return stream.Context().Err()
}

func (s *Server) ImportOrders(
    stream ordersv1.OrdersService_ImportOrdersServer,
) error {
    for {
        request, err := stream.Recv()
        if errors.Is(err, io.EOF) {
            return stream.SendAndClose(importResult())
        }
        if err != nil {
            return err
        }
        if err := s.service.Import(stream.Context(), fromProto(request)); err != nil {
            return err
        }
    }
}
```

For one stream, one goroutine may read while another writes; multiple
concurrent reads or multiple concurrent writes are not supported. Do not put
an unbounded application queue behind HTTP/2 flow control. Every handler must
observe its context and stop feature work when the caller cancels or its
deadline expires.

The complete, isolated four-cardinality example is
`examples/grpc-reference-service`. Its client-streaming collector caps both
message count and aggregate value bytes before appending; transport flow
control and per-message limits do not bound total application memory for a
long-lived stream. It also demonstrates the error rule above: its limit
failures are a domain sentinel classified by an exported `DomainErrors()`, and
its tests compose the real `grpcx.NewServer` so the client-visible status they
assert is the one the shared transport actually produces.

## Register it in bootstrap

`cmd/service/internal/bootstrap/run.go` is the composition owner. Build the
feature and its transport adapter there, then fill `Services` on the existing
`bindings` value, without making the transport package import a feature:

```go
ordersService := orders.New(/* feature dependencies */)
ordersServer := ordersgrpc.NewServer(ordersService)

bindings.Services = []grpcx.RegisterService{
    func(registrar grpc.ServiceRegistrar) {
        ordersv1.RegisterOrdersServiceServer(registrar, ordersServer)
    },
}
```

Domain errors are already wired: `newGRPCRuntime` receives the same
`domainErrors` slice as the HTTP router, so a handler returns its feature error
and the shared transport classifies it once. Return feature/domain errors rather
than a raw `status.Error`; see step 3.

A new domain identity is classified once, for both transports, with a
`failure.Mapper` owned beside the feature's error identities and appended at
`runtimeDependencies.DomainErrors` in
`cmd/service/internal/bootstrap/startup_dependencies.go`. There is deliberately
no gRPC-only mapper seam: one error answering `404` over HTTP and `Internal` over
gRPC is the failure this shared slice exists to prevent.

`UnaryPolicy` and `StreamPolicy` carry the authentication and authorization
the contract requires; the template invents neither, so supply them through the
same binding.
<!-- profile:authn-oidc-jwt:start -->

The `AUTHN=oidc-jwt` profile has already filled both slices with the verifier's
interceptors, so append rather than assign. An assignment compiles, passes every
check, and silently removes gRPC authentication:

```go
bindings.UnaryPolicy = append(bindings.UnaryPolicy, authorizeUnary)
bindings.StreamPolicy = append(bindings.StreamPolicy, authorizeStream)
```
<!-- profile:authn-oidc-jwt:end -->

Policy interceptors run after correlation, sanitized logging, panic recovery,
and process admission, but before handler-error mapping and the generated
handler. A policy's direct gRPC status is treated as deliberate service-owned
output, so its detail must already be safe for the caller; raw internal errors,
and a status the policy merely wrapped rather than returned itself, are
converted to a generic `INTERNAL` status by the surrounding policy boundary.
Policies also receive standard health methods unless the service explicitly
exempts them.

## Build a client

Create one long-lived `ClientConn` per target and share it between generated
clients:

```go
tlsCredentials := credentials.NewTLS(&tls.Config{
    MinVersion: tls.VersionTLS13,
    ServerName: "orders.internal.example",
})
clientConfig := grpcclient.DefaultConfig("dns:///orders.internal.example:9091")
conn, err := grpcclient.New(
    clientConfig,
    grpcclient.Options{
        TransportCredentials: tlsCredentials,
        MeterProvider:        metrics.MeterProvider(),
        TracerProvider:       otel.GetTracerProvider(),
        Propagation:          grpcclient.PropagationTrustedService,
    },
)
if err != nil {
    return err
}
// The dependency owner closes conn during process shutdown.
client := ordersv1.NewOrdersServiceClient(conn)
```

`grpcclient.New` uses `grpc.NewClient`, so construction performs no network
I/O. The connection owns resolution, reconnect, and connection reuse. Do not
create one connection per call. Plaintext callers must still pass
`insecure.NewCredentials()` explicitly.

Propagation is selected once per dependency connection, not per call:

- `PropagationNone` is the fail-closed zero value: client spans and metrics
  remain local, while no trace or request ID crosses the target boundary.
- `PropagationTraceContext` sends only W3C `traceparent`/`tracestate`.
- `PropagationTrustedService` additionally sends the validated request ID from
  the request context.

The client removes caller-, credential-, and resolver-supplied
`traceparent`, `tracestate`, `baggage`, and `x-request-id` before
OpenTelemetry injects the selected allowlist. Baggage is never propagated.
Choose `TrustedService` only when the named neighbor is allowed to receive the
diagnostic request ID; TLS or private-network placement alone does not imply
that trust. Unknown policy values fail during construction.

When the dependency protects standard `Health/Watch`, pass its dynamic bearer
credential through `Options.PerRPCCredentials`. The connection credential
reaches both grpc-go's health stream and application RPCs, and the same reserved
metadata removal applies to it. A per-call credential reaches only that call and
cannot make an otherwise unauthenticated backend health-eligible. Disabling
health is valid only when the dependency does not publish the whole-process
health contract; it is not an authentication bypass.

Each operation owns its realistic deadline:

```go
ctx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
defer cancel()

response, err := client.GetOrder(
    ctx,
    ordersv1.GetOrderRequest_builder{Id: new(orderID)}.Build(),
)
```

The shared client rejects proxy delegation and resolver-supplied service
configuration so neither can bypass its final resolver-metadata guard or add a
retry policy the client did not choose. It does supply a default service config
of its own, carrying the address-selection policy below; grpc-go refuses the
resolver's and keeps the client's, so that is a decision this client makes
rather than one a peer introduces. It does not opt into `WaitForReady` and sets
no universal deadline.

Address selection is `round_robin` by default, so a target resolving to several
addresses reaches all of them. gRPC's own default sends every RPC to the first
address that connects, which behind a headless DNS name is one backend wearing
N addresses; against a single-address target the two are equivalent, so the
distributing default costs nothing there. Set `LoadBalancingPickFirst` on the
connection's `Config` when the fan-out of one subchannel per backend is not
wanted.

Round robin also watches the standard health service for the empty service name
by default. A backend receives new RPCs only while it reports `SERVING`;
`NOT_SERVING` removes it from new picks without canceling work already in flight.
An `UNIMPLEMENTED` Watch falls back to connectivity for that peer. Set
`clientConfig.HealthCheck = false` only for a dependency that does not publish
whole-process health under the empty service name. Direct `pick_first` remains
connectivity-owned in the pinned grpc-go version even when health is configured.

Idle keepalive is disabled by default. A dependency with a named intermediary
idle timeout may opt in by setting both `KeepalivePingInterval` and
`KeepalivePingTimeout` to positive values accepted by that peer; a partial or
negative pair is rejected before connection construction. Enabled pings run
without an active RPC, and grpc-go retains authority over its interval floor and
recovery after a peer rejects excessive pings.

grpc-go may still perform a transparent retry before an RPC is committed to the
server; the correlation allowlist is applied to every such attempt. Any
application retry policy remains a per-method business decision: enable it only
when replay is safe, bound attempts/backoff inside the caller's deadline, and
monitor attempts. A dependency that genuinely requires a proxy or resolver
service config needs a separate design that preserves the same final metadata
boundary. Long-lived streams need an explicit idle/duration policy owned by the
feature; the server ships no stream bound, and its unary bound is a separate key
from the HTTP one. See [RPC and connection lifetime](operations-and-proof.md#rpc-and-connection-lifetime).

Client defaults are 16 KiB received metadata and 4 MiB sent/received messages.
The server also has finite connection, process-RPC, per-connection-stream,
metadata, and message limits. Raising them requires a representative payload,
concurrency, and memory measurement.

## Upstream references

- [gRPC-Go generated code and streaming concurrency](https://grpc.io/docs/languages/go/generated-code/)
- [gRPC-Go client anti-patterns and `NewClient`](https://github.com/grpc/grpc-go/blob/master/Documentation/anti-patterns.md)
- [Deadlines](https://grpc.io/docs/guides/deadlines/) and
  [retry](https://grpc.io/docs/guides/retry/)
