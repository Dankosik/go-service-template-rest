# Native gRPC

The `GRPC=enabled` initialization profile adds an optional native gRPC-Go
client/server capability beside the existing REST API. It does not replace
OpenAPI, add a gateway, or create a second business layer.

```text
REST :8080 ─────┐
                ├─> internal/<feature> ─> dependencies
gRPC :9091 ─────┘

metrics :9090 (private diagnostics)
```

REST and gRPC use separate listeners in one process. They share startup
admission, dependency readiness, request IDs, telemetry, shutdown budgeting,
and feature services. The native gRPC path keeps unary, server-streaming,
client-streaming, and bidirectional-streaming APIs available without routing
them through `net/http`.

## Enable the capability

Choose the profile when initializing a repository:

```bash
make template-init \
  MODULE=github.com/acme/orders \
  CODEOWNER=@acme/platform \
  DATABASE=none \
  GRPC=enabled
```

Omitting `GRPC` is equivalent to `GRPC=none`: protobuf tooling, gRPC runtime
packages, examples, tests, dependencies, config, and CI profile checks are
removed. The selected value is recorded in `template.lock`, and repeating the
same initialization is byte-stable.

The retained capability is still disabled at runtime. To serve locally over
explicit plaintext:

```dotenv
APP__GRPC__SERVER__ENABLED=true
APP__GRPC__SERVER__ADDR=:9091
APP__GRPC__SERVER__TRANSPORT_SECURITY=plaintext
APP__GRPC__SERVER__ALLOW_PLAINTEXT=true
```

For application TLS:

```dotenv
APP__GRPC__SERVER__ENABLED=true
APP__GRPC__SERVER__ADDR=:9091
APP__GRPC__SERVER__TRANSPORT_SECURITY=tls
APP__GRPC__SERVER__TLS__CERT_FILE=/run/secrets/service.crt
APP__GRPC__SERVER__TLS__KEY_FILE=/run/secrets/service.key
```

There is no implicit plaintext mode. TLS files are loaded and the
certificate/key pair is checked before any listener is opened or readiness is
published.

## Add a service

### 1. Define the contract

Create the canonical schema under `api/proto/<organization>/<service>/v1`.
New schemas use Edition 2023 and select the Go Opaque API in the schema:

```proto
edition = "2023";

package acme.orders.v1;

import "google/protobuf/go_features.proto";

option features.(pb.go).api_level = API_OPAQUE;
option go_package = "github.com/acme/orders/internal/gen/proto/acme/orders/v1;ordersv1";

service OrdersService {
  // GetOrder returns one order by identifier.
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
  // WatchOrders streams matching order updates.
  rpc WatchOrders(WatchOrdersRequest) returns (stream WatchOrdersResponse);
  // ImportOrders imports a client-provided order sequence.
  rpc ImportOrders(stream ImportOrdersRequest) returns (ImportOrdersResponse);
  // SyncOrders synchronizes orders bidirectionally.
  rpc SyncOrders(stream SyncOrdersRequest) returns (stream SyncOrdersResponse);
}
```

The `.proto` file is the public authority. Generated Go under
`internal/gen/proto` is derived and must not be edited by hand. Keep messages
independent from database rows, never reuse field numbers, reserve deleted
numbers and names, and start enums with an `UNSPECIFIED` zero value.
The generator configuration does not override the Go API level. The current
policy accepts new Edition 2023 schemas only when they select `API_OPAQUE` and
rejects Edition 2024 until the cross-language contract decision is reopened.
Retained proto2/proto3 contracts keep their existing generated API until their
owner migrates them deliberately, but lint accepts one only when a readable
`BASE_REF` contains legacy syntax at the same path. A renamed or newly added
proto2/proto3 file is therefore a new contract and is rejected.

### 2. Generate and verify

```bash
make proto-format
make proto-format-check
make proto-generate
BASE_REF=origin/main make proto-check
BASE_REF=origin/main make proto-breaking
```

The pinned Buf v2 runner compiles without a local `protoc` installation or Buf
account. Official `protoc-gen-go` and `protoc-gen-go-grpc` plugins come from
the pinned tools module. `proto-check` performs a non-mutating format check,
runs `STANDARD` plus public-contract `COMMENTS` lint, and fails when generated
Go has drifted. `proto-breaking` requires a readable Git base, reports a first
publication as not applicable, and applies conservative `FILE` compatibility
rules. `BASE_REF` is optional for an Edition-only repository and required to
prove that any proto2/proto3 path is retained rather than newly introduced.

Commit schema and generated Go together. Add `buf.lock` only after
`buf.yaml` gains an external dependency; with no dependencies there is nothing
to lock.

### Editor feedback and manual calls

Buf includes an LSP server, so editor diagnostics, navigation, completion, and
formatting use the same pinned binary and repository policy as CI. Configure
the editor's LSP client to start this command from the repository root:

```bash
bash ./scripts/run-buf.sh lsp serve
```

Reflection stays disabled by default. For a local plaintext smoke call, pass
the checked-in schema to `buf curl` instead:

```bash
bash ./scripts/run-buf.sh curl \
  --schema . \
  --protocol grpc \
  --http2-prior-knowledge \
  --data '{"id":"order-123"}' \
  http://127.0.0.1:9091/acme.orders.v1.OrdersService/GetOrder
```

Use the deployment's TLS endpoint and trust configuration instead of
`--http2-prior-knowledge` outside an explicitly allowed plaintext boundary.
Generated clients and the real TCP integration suite remain the CI oracle;
`buf curl` is for development and operational smoke checks.

### 3. Implement the generated server

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

A handler therefore does not choose its own `codes.Code`. It chooses a
`problem.Code`, and the transport renders it. That indirection is what keeps one
domain error answering consistently over both HTTP and gRPC, and it fixes the
reachable vocabulary:

| `problem.Code` | gRPC code |
| --- | --- |
| `CodeBadRequest`, `CodeUnprocessableContent` | `InvalidArgument` |
| `CodeUnauthorized` | `Unauthenticated` |
| `CodeForbidden` | `PermissionDenied` |
| `CodeNotFound` | `NotFound` |
| `CodeMethodNotAllowed` | `Unimplemented` |
| `CodeConflict` | `Aborted` |
| `CodeRequestEntityTooLarge`, `CodeTooManyRequests` | `ResourceExhausted` |
<!-- profile:authn-oidc-jwt:start -->
| `CodeRequestHeaderFieldsTooLarge` | `ResourceExhausted` |
<!-- profile:authn-oidc-jwt:end -->
| `CodeServiceUnavailable` | `Unavailable` |
| `CodeGatewayTimeout` | `DeadlineExceeded` |
| `CodeInternalError`, unclassified | `Internal` |

A canceled or expired RPC context answers `CANCELED` or `DEADLINE_EXCEEDED`
before classification runs, because that is the caller's own signal rather than a
service outcome. `FailedPrecondition`, `AlreadyExists`, `OutOfRange`, and
`DataLoss` are not reachable through this table; needing one is a contract
decision that extends `problem` and `mappedStatus` together, not a local
`status.Error` in a handler.

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

### 4. Register it in bootstrap

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

A new domain identity is classified once, for both transports, by appending its
`problem.Mapper` at `runtimeDependencies.DomainErrors` in
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
conn, err := grpcclient.New(
    grpcclient.DefaultConfig("dns:///orders.internal.example:9091"),
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
hidden retry/balancer policy. It does not opt into `WaitForReady` and sets no
universal deadline. grpc-go may still perform a transparent retry before an
RPC is committed to the server; the correlation allowlist is applied to every
such attempt. Any application retry policy remains a per-method business
decision: enable it only when replay is safe, bound attempts/backoff inside
the caller's deadline, and monitor attempts. A dependency that genuinely
requires a proxy or resolver service config needs a separate design that
preserves the same final metadata boundary. Long-lived streams need an
explicit idle/duration policy owned by the feature; the server does not apply
the HTTP request timeout to them.

Client defaults are 16 KiB received metadata and 4 MiB sent/received messages.
The server also has finite connection, process-RPC, per-connection-stream,
metadata, and message limits. Raising them requires a representative payload,
concurrency, and memory measurement.

## Runtime behavior

Every RPC passes through the same interceptor chain, identical for unary and
streaming. The package doc on `internal/infra/grpc` is the single owner of the
order and of what each position buys a policy author — read it with
`go doc ./internal/infra/grpc`, and `builtinPolicies` in
`internal/infra/grpc/chain.go` is the executable copy. What the chain guarantees
a caller:

- `x-request-id` is validated or created and returned in response metadata. A
  single valid value is accepted; zero values, an invalid value, or two or more
  values all mint a fresh identifier, which is stricter than the HTTP listener's
  first-of-several header read.
- Panics never reach the caller and never disclose the panic value.
- Admission is one non-blocking process-wide RPC semaphore; an RPC over the
  limit is shed as `RESOURCE_EXHAUSTED` rather than queued.
- Service-supplied policy interceptors and generated handlers each answer
  through a sanitizing error boundary. A policy that means to choose its own
  status returns a plain `status.Error`; a handler classifies its domain error
  through `DomainErrors`. Anything else reaches the caller as `INTERNAL`, text
  included.
- Completion is logged once, outside error mapping, so the record carries the
  status the caller actually received.

Health RPCs bypass application admission and are excluded from routine access
logs by default. The standard `grpc.health.v1.Health` service starts
`NOT_SERVING`, becomes `SERVING` only after the same dependency admission as
HTTP readiness, and returns to `NOT_SERVING` before drain. Its standard status
semantics bypass business-handler error mapping, so an unknown service remains
`NOT_FOUND`.

Successful business access logs are compatible and complete by default:
`APP__GRPC__SERVER__ACCESS_LOG_SUCCESS_SAMPLE_RATE=1`. For a measured
high-throughput workload, the rate may be lowered to a value in `[0,1]`.
Non-OK terminal statuses are still logged, and a positive
`APP__GRPC__SERVER__ACCESS_LOG_SLOW_THRESHOLD` retains successful calls at or
above that duration before sampling is considered. The default `0s` disables
the slow override because the template cannot invent a service latency SLO.
Sampling is deterministic for the validated request ID, so it adds no shared
random-number lock. When INFO logging is disabled, the interceptor bypasses
timing and attribute construction entirely.

On shutdown, HTTP readiness and gRPC health enter drain together. After the
configured propagation delay, HTTP and gRPC drain concurrently under the same
remaining application shutdown budget. gRPC first uses `GracefulStop`; expiry
starts transport `Stop` and returns at the caller's deadline without joining
either library stop call. grpc-go cannot kill a Go handler: a handler that
ignores its canceled RPC context may continue until the process exits, so every
handler must release feature work and dependencies on cancellation.

OpenTelemetry client/server `StatsHandler`s cover unary and streaming protocol
spans and metrics. Repository interceptors add only request identity, access
status, and low-cardinality active/shed RPC instruments. Request messages,
metadata values, peer-controlled names, and raw error text are not metric
labels. Server protocol telemetry is limited to methods present in the
registered service descriptors; unknown peer-supplied method paths are omitted
instead of becoming unbounded span names or `rpc.method` series. Routine server
health spans and duration samples are also omitted by default because probe
frequency is not business traffic. Set
`APP__GRPC__SERVER__TELEMETRY_HEALTH_CHECKS=true` for focused diagnosis; health
handling and client-side telemetry are unchanged either way.

The process-wide RPC limit and per-connection stream limit constrain different
owners. Keep one long-lived shared client connection first. Add another
connection only when representative evidence shows client-side queueing at the
per-connection stream ceiling while process admission, CPU, memory, network,
and dependencies still have headroom. Raising either limit without the
matching concurrent payload-memory measurement is not a performance
optimization.

Reflection, keepalive tuning, a registry, grpc-gateway, Connect, and gRPC-Web
are absent by design. Add one only for a concrete contract, security, or
measured reliability requirement.

## Profile-guided optimization

Go PGO is available as an explicit build input, not a template default:

```bash
make bench-profile \
  BENCH_PACKAGE=./internal/infra/grpc \
  BENCH_PATTERN='^BenchmarkGRPCUnary/full_json/64B$' \
  BENCH_PROFILE=cpu \
  BENCH_WORKLOAD_ID=grpc-reference-unary

make build-pgo PGO_PROFILE=.artifacts/bench/profiles/cpu.pprof
make docker-build PGO_PROFILE=path/inside/build-context/service.pprof
```

The benchmark profile above proves only the build path. A release profile must
come from the same service binary under a representative production-shaped
workload. Retain its source revision, workload identity, Go version,
collection interval, and SHA-256 in the private delivery evidence. Refresh it
after a material compiler, workload, schema, or hot-path change. Do not commit
a synthetic template `default.pgo`: automatic discovery can silently apply a
stale profile to a later release. Every activated local or image build rejects
an absent or unreadable profile before compilation. Rebuild with
`PGO_PROFILE=off` for immediate rollback.

PGO does not replace correctness, race, contract, image, or benchmark proof.
Only equivalent repeated baseline/candidate evidence may claim an improvement.

<!-- profile:grpc-reference-benchmark:start -->
## Local smoke and synthetic benchmarks

For four-cardinality correctness and local synthetic sensitivity, use the
production-composed loopback server and digest-pinned Grafana k6 harness:

```bash
make bench-grpc-inspect
make bench-grpc-smoke
make bench-grpc
```

The harness builds
`examples/grpc-reference-service/cmd/benchmark-server` before measurement. The
command composes the generated reference service through the production
`grpcx.NewServer` adapter, applies the canonical production connection and
transport bounds, and listens only on an allocated `127.0.0.1` port. It does
not enable reflection or change the production service configuration.

The single canonical k6 scenario loads the Edition 2023 reference schema
directly and exercises unary, server-streaming, client-streaming, and
bidirectional-streaming RPCs. Inspect the schema and scenario on the exact
digest-pinned image before relying on a new host.

`bench-grpc-smoke` uses one VU and completes every cardinality once. Exact
payload, order, message count, one-based server-stream sequence, clean stream
termination, and a non-zero success denominator for each cardinality are
thresholds. Smoke timings are diagnostic wiring evidence only.

`bench-grpc` runs an explicit warmup followed by a completion window equal to
the configured RPC timeout, then a fixed measured interval. The separation
prevents an in-flight warmup stream from overlapping measured load. Measured
iterations rotate across all four methods. Custom metrics retain operation and
stream successes, sent/received/processed messages, correctness and terminal
failures, unary duration, stream lifetime, and correlated bidi message lag.
The summary retains p50, p95, and p99 trend values, k6 byte and dropped-work
metrics, and independently tagged success denominators. Warmup does not enter
those custom measured-phase metrics, and every streaming operation uses the
recorded RPC timeout.

The bounded defaults are sensitivity assumptions, not service budgets. Override
them only as part of a named workload:

```bash
make bench-grpc \
  GRPC_BENCH_WORKLOAD_ID=reference-1k-8msg-4vu \
  GRPC_BENCH_PAYLOAD_BYTES=1024 \
  GRPC_BENCH_STREAM_MESSAGES=8 \
  GRPC_BENCH_VUS=4 \
  GRPC_BENCH_WARMUP_DURATION=5s \
  GRPC_BENCH_DURATION=30s \
  GRPC_BENCH_RPC_TIMEOUT=10s
```

`GRPC_BENCH_PAYLOAD_BYTES` is bounded at 1 MiB and
`GRPC_BENCH_STREAM_MESSAGES` at 1024 so the harness cannot silently raise the
reference or production safety limits. One client connection is shared per k6
VU. On macOS the container reaches the loopback listener through
`host.docker.internal`; on Linux the runner uses host networking. Any other
platform fails closed rather than widening the listener.

Artifacts are mode-scoped below `.artifacts/bench/grpc/`:

- `summary.json`: evidence level, workload inputs, unavailable resource
  observables, and the complete k6 metric summary;
- `k6.log` and `server.log`: client and benchmark-server diagnostics;
- `run.meta`: source/schema/scenario digests, pinned image, platform route,
  server Go toolchain/OS/architecture, payload/concurrency/duration settings,
  and Docker identity;
- `samples.json.gz`: optional per-sample output when
  `GRPC_BENCH_RAW_SAMPLES=1`.

The runner validates artifact ownership before deletion, starts and records one
server PID, and always terminates and joins only that process. Compilation and
startup are outside the measured k6 phase.

A successful local synthetic run proves harness sensitivity under its recorded
inputs. It does not prove production throughput, sustainable capacity, or an
optimization. Those claims additionally require an equivalent representative
service/testbed, repeated comparable baseline and candidate runs, server CPU
and memory/GC, admission utilization and rejection, connection/stream counts,
network utilization, and load-generator CPU/network headroom. The local summary
records those decision-grade observables as unavailable. The general
measurement and comparison contract remains in
[`docs/benchmarking.md`](benchmarking.md).
<!-- profile:grpc-reference-benchmark:end -->

## Railway boundary

The repository does not claim that a native gRPC service is reachable through
a Railway public HTTP domain. Use Railway private networking with the service's
internal DNS and explicit port for service-to-service traffic. For public
native gRPC, revalidate end-to-end HTTP/2 trailers on the current platform or
use Railway TCP Proxy with application TLS and hostname verification.

The existing `railway.toml` exposes and health-checks the REST listener only.
Enabling the gRPC runtime does not publish its port or prove public reachability.

## Focused proof

```bash
go test -vet=off \
  ./internal/reqctx \
  ./internal/infra/http \
  ./internal/infra/grpc \
  ./internal/infra/grpcclient \
  ./internal/infra/telemetry \
  ./cmd/service/internal/bootstrap

go test -vet=off -race \
  ./internal/infra/grpc \
  ./internal/infra/grpcclient \
  ./cmd/service/internal/bootstrap

go test -vet=off ./examples/grpc-reference-service/...
# Upstream/PostgreSQL profile process proof:
go test -vet=off -count=1 -tags=integration ./test/... -run GRPC
make proto-check
```

The upstream/PostgreSQL-profile integration test starts the production binary,
waits for standard gRPC health and existing HTTP readiness, then proves
coordinated SIGTERM shutdown. `DATABASE=none` removes container-backed process
tests; its bootstrap lifecycle test remains in the ordinary Go suite.

## Upstream references

- [gRPC-Go generated code and streaming concurrency](https://grpc.io/docs/languages/go/generated-code/)
- [gRPC-Go client anti-patterns and `NewClient`](https://github.com/grpc/grpc-go/blob/master/Documentation/anti-patterns.md)
- [Deadlines](https://grpc.io/docs/guides/deadlines/),
  [retry](https://grpc.io/docs/guides/retry/),
  [health](https://grpc.io/docs/guides/health-checking/), and
  [graceful shutdown](https://grpc.io/docs/guides/server-graceful-stop/)
- [Go Opaque API](https://protobuf.dev/reference/go/opaque-faq/) and
  [Protobuf compatibility practices](https://protobuf.dev/best-practices/dos-donts/)
- [Buf generation](https://buf.build/docs/generate/),
  [formatting](https://buf.build/docs/format/),
  [lint rules](https://buf.build/docs/lint/rules/),
  [editor/LSP integration](https://buf.build/docs/cli/editors-lsp/),
  [`buf curl`](https://buf.build/docs/curl/),
  [breaking rules](https://buf.build/docs/breaking/rules/), and
  [CLI installation guidance](https://buf.build/docs/cli/installation/)
- [Railway private networking](https://docs.railway.com/private-networking) and
  [TCP Proxy](https://docs.railway.com/networking/tcp-proxy)
