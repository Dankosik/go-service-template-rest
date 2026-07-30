# gRPC capability design

status: ready

## Selected architecture

The template will add gRPC as an optional native transport capability, not as a
replacement for REST and not as a framework around feature code.

```text
                       one process
  ┌──────────────┐  ┌──────────────────┐  ┌───────────────────┐
  │ REST listener│  │ native gRPC      │  │ private diagnostics│
  │ chi/OpenAPI  │  │ grpc-go + health │  │ metrics/pprof      │
  └──────┬───────┘  └────────┬─────────┘  └─────────┬─────────┘
         │                   │                      │
         └──────────┬────────┘                      │
                    │                               │
       transport-free internal/<feature>            │
                    │                               │
       shared startup admission and drain ──────────┘
```

`GRPC=none|enabled` is resolved by `scripts/init-module.sh`. The upstream
template carries the enabled implementation so it can prove it. An initialized
service defaults to `GRPC=none`; an enabled service still defaults at runtime
to `grpc.server.enabled=false`.

The production stack is:

- `google.golang.org/grpc` for client and server transport;
- `google.golang.org/protobuf` and the official Go generators;
- Buf v2 for format, `STANDARD` plus `COMMENTS` lint, generation orchestration,
  and `FILE` breaking checks;
- `otelgrpc` `StatsHandler`s for standard client/server spans and metrics;
- the official gRPC health service;
- small repository-owned interceptors and lifecycle adapters for policies the
  native libraries deliberately do not own.

No gRPC framework, grpc-gateway, middleware bundle, service registry, reflection
service, global retry policy, or keepalive policy is added.

## Decision drivers and alternatives

| Decision | Selection | Consequence and reopen condition |
| --- | --- | --- |
| Wire stack | Native `grpc-go` | Gives the generated generic streaming APIs and the smallest dependency/policy surface. Reopen only if a required browser/REST compatibility contract appears. |
| Listener topology | Separate HTTP and gRPC listeners in the existing process | Keeps protocol identity, TLS, limits, and Railway exposure explicit while sharing feature services and lifecycle. Reopen if the platform proves one-port protocol multiplexing is required. |
| Schema language | Edition 2023 plus schema-owned `API_OPAQUE` | Gets the recommended Go Opaque API without silently adopting Edition 2024 cross-language defaults. Reopen when every promised client language accepts and proves Edition 2024. |
| Retained schema syntax | proto2/proto3 only at the same path in a readable comparison base | Keeps existing contracts operable without letting a new or renamed legacy-syntax file bypass the Edition 2023 baseline. |
| Contract tooling | Buf v2 with local official Go plugins | One reproducible workflow without a registry/account dependency. `buf.lock` is committed only after external Buf dependencies exist. |
| Generated ownership | `.proto` under `api/proto`; generated Go under `internal/gen/proto` | Schemas are public authority; Go output is internal derived code. The directory remains absent before the first owned proto. |
| Server observability | `otelgrpc` stats handler plus repository access/admission interceptors | Stats handlers own protocol telemetry and propagation; interceptors add request ID, sanitized access records, domain status mapping, panic recovery, and process admission. |
| Client abstraction | A `grpcclient.New` `ClientConn` constructor | Generated clients remain the normal API. Feature-owned ports are introduced only when the feature needs inversion. |
| Public Railway ingress | No native gRPC-over-public-HTTP claim | Private networking is supported. Public use needs a fresh end-to-end trailer probe or TCP Proxy plus application TLS. |

Connect, gRPC-Gateway, and a full Go service framework were rejected as base
choices because they add a second public protocol or lifecycle/config model
without an accepted requirement. `go-grpc-middleware` was rejected because
native interceptor chaining covers the selected policies and avoids a
dependency whose value would only be composition syntax.

## Contract and tool ownership

Root `buf.yaml` describes the future production module at `api/proto`.
`buf.gen.yaml` invokes repository-pinned `protoc-gen-go` and
`protoc-gen-go-grpc` binaries through `scripts/run-go-tool.sh`. Buf itself is
run by `scripts/run-buf.sh` at one pinned version in an isolated cache; it is
not added to the repository tools module.

`scripts/proto.sh` is the single command owner. It:

- reports a stable no-op while `api/proto` has no owned source;
- also processes the isolated upstream gRPC reference module when present;
- formats, checks canonical formatting without mutation, lints public contract
  documentation and schema rules, generates, checks generated drift, and
  compares the production module with an explicit readable Git base;
- distinguishes an absent base contract from an invalid base reference.

The reference contract lives under `examples/grpc-reference-service` with its
own Buf configs and generated package. It is removed by either `GRPC=none` or
`REFERENCE_EXAMPLE=remove`, so it never becomes a production API accidentally.

## Runtime ownership

### Configuration

`internal/config` owns a `GRPCConfig` section. Its server subsection has:

- `enabled`, `addr`, and explicit `transport_security`;
- `allow_plaintext`, `tls.cert_file`, and `tls.key_file`;
- finite connection, process-RPC, per-connection-stream, metadata, receive
  message, and send message limits;
- health access-log suppression.

The default bounds are constraint defaults, not a throughput claim:

| Setting | Default |
| --- | ---: |
| `grpc.server.max_connections` | 4096 |
| `grpc.server.max_concurrent_rpcs` | 256 |
| `grpc.server.max_concurrent_streams` | 100 |
| `grpc.server.max_header_list_bytes` | 16384 |
| `grpc.server.max_receive_message_bytes` | 4194304 |
| `grpc.server.max_send_message_bytes` | 4194304 |

All bounds must be positive and fit the native API types. The process-RPC
ceiling must not exceed `max_connections * max_concurrent_streams`. Enabling
the server requires a non-empty address and exactly one explicit security
mode. Plaintext additionally requires `allow_plaintext=true` and rejects TLS
inputs. TLS rejects the plaintext acknowledgement and requires both files.
File readability and certificate/key validity are checked while the server is
built, before any listener or readiness transition.

### Server adapter

`internal/infra/grpc` (package `grpcx`) owns:

- construction and validation of native server options;
- standard health registration and its `NOT_SERVING → SERVING →
  NOT_SERVING` controller;
- unary and streaming interceptor chains;
- the process-wide non-blocking RPC semaphore;
- domain-error-to-status mapping;
- the `runtimeServer` adapter for `Serve`, context-bounded graceful stop, and
  forced stop.

The constructor receives explicit registration callbacks rather than importing
feature or generated business packages. Bootstrap is the only composition
owner and registers each feature adapter there.

For both unary and streaming handlers, outer-to-inner interceptor order is:

1. request-ID correlation and response metadata;
2. sanitized access completion log;
3. panic recovery;
4. non-blocking process admission;
5. caller-supplied authentication/authorization or other trusted feature
   policy inside a raw-error sanitization boundary;
6. generated-handler domain-error mapping and sanitization;
7. generated handler.

Official health RPCs bypass process admission and are omitted from routine
access logs by default. They also bypass generated-handler error mapping so the
standard health service retains its specified status semantics. The HTTP/2
per-connection stream limit still applies.
Unmapped errors and panics are logged internally but returned only as sanitized
`INTERNAL`. Statuses produced inside this adapter use an unexported
`ownedStatusError` that implements `GRPCStatus`; the error mapper preserves only
that marker. An ordinary `status.Error` returned by a handler or dependency has
no trusted provenance and is sanitized. Feature-specific codes therefore enter
through service-owned `problem.Mapper` identities, not by passing through a
downstream status. Supplied policy interceptors sit outside that handler
boundary; a direct status they return is service-owned caller output and must
already contain safe detail. Their context failures are normalized and any raw
non-status error is sanitized to the adapter-owned generic `INTERNAL`.

`internal/reqctx` becomes the single owner of request-ID validation and
generation. HTTP and gRPC both call it, preserving the existing 128-character
safe alphabet contract.

`internal/infra/telemetry` exposes a transport-parameterized admission-load
instrument constructor. HTTP retains its current metric names; gRPC records
active and shed RPCs under `service.grpc.server`. Standard protocol duration
and status dimensions remain owned by `otelgrpc`. Full method names come only
from registered descriptors: after health and feature registration, the server
derives the accepted method set from `grpc.Server.GetServiceInfo()` and filters
unknown paths out of `otelgrpc` server signals. No peer metadata or message
field is a metric label.

### Client adapter

`internal/infra/grpcclient` owns a constructor around `grpc.NewClient`.
Configuration is passed per dependency, not placed in the process-global
runtime config. It requires:

- a non-empty target;
- non-nil explicit transport credentials, including
  `insecure.NewCredentials()` for an accepted plaintext boundary;
- positive metadata and send/receive message limits;
- an explicit meter provider, with the global tracer provider and W3C trace
  propagator installed during bootstrap.

`grpcclient.DefaultConfig(target)` supplies documented defaults of 16 KiB
received metadata and 4 MiB for both received and sent messages. `New` rejects
zero or negative values, so a caller starts from `DefaultConfig` and makes an
explicit validated override rather than relying on zero-value native limits.

It installs the `otelgrpc` client stats handler and default call message
bounds. It does not connect, install a retry service config, block, set
`WaitForReady`, set a global deadline, or hide `ClientConn.Close`. grpc-go's
pre-commit transparent retry and resolver-supplied service config remain native
transport behavior. One returned connection is shared by the feature's
generated clients.

## Status mapping

The transport maps context errors first, preserves only its private
`ownedStatusError`, and then consults the existing ordered `problem.Mapper`
list:

| Problem code | gRPC code |
| --- | --- |
| bad request, unprocessable content | `INVALID_ARGUMENT` |
| unauthorized | `UNAUTHENTICATED` |
| forbidden | `PERMISSION_DENIED` |
| not found | `NOT_FOUND` |
| method not allowed | `UNIMPLEMENTED` |
| conflict | `ABORTED` |
| request too large, too many requests | `RESOURCE_EXHAUSTED` |
| service unavailable | `UNAVAILABLE` |
| gateway timeout | `DEADLINE_EXCEEDED` |
| internal or unmapped | `INTERNAL` |

The mapper may return only service-owned safe detail. A raw error string or
unmarked `status.Error` is never copied to the status. Process admission
independently returns `RESOURCE_EXHAUSTED` before a business handler runs.
Connection, metadata, and HTTP/2 stream caps execute below interceptors and may
close/reset transport state instead of returning an application status.

## Composition and lifecycle

`serveHTTPRuntime` becomes protocol-neutral `serveRuntime`. It binds all
requested application listeners before startup admission, closing already
opened listeners on any later bind failure. A requested gRPC setup or bind
failure rejects the whole process.

The same readiness refresh admits both transports. On success, bootstrap marks
the gRPC health service `SERVING`; HTTP readiness and gRPC health therefore
publish the same admission result.

The existing drainer becomes a composite of `health.Service` and the gRPC
health controller. On signal it marks both not serving, waits the existing
propagation interval, then concurrently drains HTTP and gRPC under the same
remaining application shutdown context. The gRPC adapter runs
`GracefulStop`; when the context expires it calls `Stop` and returns the context
error without joining either library stop call past the budget for a handler
that ignores cancellation. Those bounded stop goroutines are process-lifetime
after forced stop and exit when all handlers eventually return. Diagnostics and telemetry retain their
existing later positions.

The server does not impose a universal per-stream timeout. Generated handlers
must observe their context; a feature that permits long-lived streams owns its
idle/duration policy.

## Security, rollout, and compatibility boundaries

- Reflection and keepalive remain absent.
- Plaintext is an explicit deployment acknowledgement, never a default.
- TLS uses application-owned certificate/key material and fails closed.
- Authentication/authorization is a supplied interceptor seam, not a capability
  claim.
- Private Railway service-to-service traffic uses private DNS and an unexposed
  port. Public native gRPC remains unverified until the current Railway edge is
  proven to preserve HTTP/2 trailers; TCP Proxy requires application TLS and
  hostname validation.
- Schema compatibility is proved against an explicit Git base with Buf
  `FILE`; first publication is reported as not applicable.

Reopen design if the accepted outcome requires browser clients, public
Railway HTTP ingress, multi-language Edition 2024, service discovery, global
retry semantics, authenticated base RPCs, reflection, keepalive, or one-port
protocol multiplexing.
