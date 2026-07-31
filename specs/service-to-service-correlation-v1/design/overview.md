# Trusted service-to-service correlation design

status: ready

## Selected architecture

Keep OpenTelemetry and gRPC-Go as the protocol owners and add one small,
transport-local enforcement adapter to each optional outbound client.

```text
generated HTTP client
  -> httpclient.Client
     -> retry
     -> propagation sanitizer
     -> otelhttp attempt span/metrics
        -> closed HTTP policy propagator
     -> existing authority/body/address bounds
     -> net/http

generated gRPC client
  -> grpcclient.ClientConn
     -> logical-call sanitizer / credential guard
     -> grpc-go attempt
     -> otelgrpc attempt span/metrics
        -> closed gRPC policy propagator
     -> native HTTP/2 transport
```

There is no shared propagation framework, global middleware switch, generated
placeholder client, or new dependency. `internal/reqctx` remains the single
request-ID value/validity authority. Each optional client owns the disclosure
policy and carrier mechanics for its own protocol.

## Drivers and current evidence

| Driver | Current evidence | Design consequence |
| --- | --- | --- |
| Correct remote causality | `otelhttp` creates a span per explicit HTTP attempt; grpc-go calls a client `StatsHandler` for each native attempt and `otelgrpc` starts the attempt span there. | Sanitization/injection runs after the attempt span exists and before the bounded/native transport sends. |
| Fail-closed disclosure | Current HTTP and gRPC clients implicitly inject W3C context; gRPC also accepts an arbitrary `Options.Propagators`. | Zero policy emits nothing remotely; the arbitrary outbound gRPC propagator option is removed. |
| No late gRPC metadata owner | grpc-go may accept resolver service configuration that selects configured retries and a balancer whose picker adds metadata after `StatsHandler.TagRPC`. | Use native `grpc.WithDisableServiceConfig`; transparent retry remains, while configured retry/load balancing needs a separate dependency design. |
| Preserve discovery without hidden headers | grpc-go also appends deprecated `resolver.Address.Metadata` after propagation. | Wrap the effective resolver locally, preserve its addresses/server names/attributes, and clear address metadata before grpc-go receives state. |
| Keep the resolver wrapper final | grpc-go normally inserts its proxy delegating resolver, which may construct an unwrapped DNS resolver behind the selected builder. | Use native `grpc.WithNoProxy`; proxy-dependent clients require their own reviewed dependency design. |
| Existing request-ID authority | `reqctx.RequestID` and `reqctx.ValidRequestID` already serve HTTP and gRPC inbound boundaries. | Reuse them unchanged; do not create another ID type or generator. |
| Optional capability purity | Initialization removes `internal/infra/httpclient` and `internal/infra/grpcclient` independently. | Policy types and mechanics stay inside those removable packages. |
| Generated contracts remain real-owner specific | The template has no downstream neighbor or downstream schema. | Add composition guidance and interface proof, not a fake client/schema or production bootstrap dependency. |
| Context/lifecycle remain operation and bootstrap owned | Both clients already accept caller contexts and expose existing cleanup. | No global timeout, detached context, connection-per-call, or new lifecycle registry. |

The selected implementation relies on repository-pinned, already operated
dependencies:

- OpenTelemetry Go `v1.44.0` and contrib `v0.69.0`, Apache-2.0, for W3C
  encoding and HTTP/gRPC client spans. `otelhttp.Transport` injects after
  starting its client span, and `otelgrpc` provides the native gRPC client
  `StatsHandler`. See the tagged
  [otelhttp transport](https://github.com/open-telemetry/opentelemetry-go-contrib/blob/v0.69.0/instrumentation/net/http/otelhttp/transport.go),
  [otelgrpc stats handler](https://github.com/open-telemetry/opentelemetry-go-contrib/blob/v0.69.0/instrumentation/google.golang.org/grpc/otelgrpc/stats_handler.go),
  and [OpenTelemetry license](https://github.com/open-telemetry/opentelemetry-go-contrib/blob/v0.69.0/LICENSE).
- gRPC-Go `v1.82.1`, Apache-2.0, for native attempts, metadata, interceptors,
  generated clients, and connection lifecycle. See the tagged
  [client stream implementation](https://github.com/grpc/grpc-go/blob/v1.82.1/stream.go),
  [metadata contract](https://github.com/grpc/grpc-go/blob/v1.82.1/Documentation/grpc-metadata.md),
  and [license](https://github.com/grpc/grpc-go/blob/v1.82.1/LICENSE).
- W3C Trace Context remains the wire authority. Its privacy guidance explicitly
  allows trust-boundary-specific propagation decisions and forbids sensitive
  values in `traceparent`/`tracestate`: [W3C Trace
  Context](https://www.w3.org/TR/trace-context/).
- Baggage remains excluded because OpenTelemetry documents that automatic
  propagation can disclose sensitive values to unintended third parties and
  carries no built-in integrity guarantee: [OpenTelemetry
  Baggage](https://opentelemetry.io/docs/concepts/signals/baggage/).

Versions and contracts above were checked on 2026-07-31 against `go.mod` and
tagged/official sources.

## HTTP mechanism

### API and construction

`internal/infra/httpclient` adds:

```go
type PropagationPolicy uint8

const (
    PropagationNone PropagationPolicy = iota
    PropagationTraceContext
    PropagationTrustedService
)
```

`Config` gains `Propagation PropagationPolicy`. `New` rejects values outside
that closed set before constructing or using a transport. The zero value is
valid and means `PropagationNone`.

`New` preserves the current fixed-authority and retry ownership, with this
outer-to-inner order:

1. `retryTransport`, when configured;
2. an unexported `propagationSanitizer`;
3. `otelhttp.Transport`, configured with the closed policy propagator;
4. existing `authorityTransport` and `responseLimitTransport`;
5. the owned `http.Transport`.

The HTTP client keeps its current tracer-provider ownership: `otelhttp` reads
the process-global OpenTelemetry tracer provider, while the existing
constructor continues to receive an explicit meter provider. This change does
not add a second telemetry-options API merely for test injection. Focused span
tests use the repository's existing cleanup-restored global OTel fixture and
must not run in parallel with other global-provider tests.

### Per-attempt enforcement

For every explicit attempt, `propagationSanitizer`:

1. clones the attempt request, header map, and trailer map;
2. finds header and trailer map keys case-insensitively and removes every
   occurrence of `traceparent`, `tracestate`, `baggage`, and `X-Request-ID`;
3. passes only the clone to `otelhttp`.

`otelhttp` then starts the client attempt span and calls the transport-local
policy propagator. That propagator:

- emits nothing for `PropagationNone`;
- delegates W3C representation to `propagation.TraceContext{}.Inject` for
  `PropagationTraceContext` and `PropagationTrustedService`;
- additionally sets one `X-Request-ID` for `PropagationTrustedService` only
  when the attempt context contains a value accepted by
  `reqctx.ValidRequestID`.

Its `Extract` is a no-op because it is client-injection-only, and `Fields`
returns only the fields the selected policy can emit. OpenTelemetry continues
to own when propagation runs relative to the client span; repository code owns
the closed field allowlist and request-ID rule.

Iteration with `strings.EqualFold` is required instead of relying only on
`http.Header.Del`: direct map assignment can introduce a non-canonical key that
`Del` does not remove. `http.Request.Clone` deep-copies both header and trailer
maps, so sanitizing the clone preserves caller ownership. Retrying re-enters
`otelhttp` and the policy transport, so each explicit retry gets its own client
span and a freshly derived wire context.

An internal retry performed entirely inside one standard-library
`http.Transport.RoundTrip` reuses already sanitized headers and therefore
cannot reopen disclosure. It remains one OTel HTTP client span. Reopen
Reliability only if a future accepted contract requires a span per hidden
connection resend.

## gRPC mechanism

### API and construction

`internal/infra/grpcclient.Options` replaces:

```go
Propagators propagation.TextMapPropagator
```

with:

```go
Propagation PropagationPolicy
```

The policy type and constants are transport-local and match the HTTP names.
`New` rejects unknown values before calling `grpc.NewClient`.

`New` installs:

- one unary and one stream interceptor for logical-call sanitization;
- `otelgrpc.NewClientHandler` with the closed policy propagator;
- `grpc.WithDisableServiceConfig`;
- `grpc.WithNoProxy`;
- connection-local sanitizing wrappers around every resolver builder that can
  win grpc-go's native target selection;
- the existing credentials and message/header limits.

`otelgrpc` remains the unwrapped owner of attempt spans, metrics, and
propagator invocation. The transport-local propagator has the same closed
injection behavior as the HTTP variant, using a gRPC metadata carrier.
Resolver-supplied service configuration and proxy delegation are disabled
before channel use. A resolver therefore cannot select configured retries,
load balancing, picker metadata, or a second unwrapped DNS resolver outside
that boundary.

### Resolver boundary

`New` leaves target parsing and the winning resolver choice to grpc-go. Before
calling it, the package obtains and wraps the finite set of builders that can
win that native selection for this connection:

1. the builder for an explicitly registered target scheme, when present;
2. the `dns` builder, which grpc-go uses as `NewClient`'s default when no
   process default was explicitly changed;
3. the builder named by `resolver.GetDefaultScheme`, covering an explicitly
   selected process default and the initial passthrough value.

Duplicate schemes collapse to one wrapper. Missing candidates are omitted;
grpc-go's own construction path remains responsible for rejecting a target
whose effective builder is unavailable. The wrappers are passed through
`grpc.WithResolvers`, which takes precedence over the process-global registry
for those schemes without replacing grpc-go's selection rule.

Each wrapper delegates `Build` to its underlying builder with a wrapped
`resolver.ClientConn`. When the underlying builder implements
`resolver.AuthorityOverrider`, a distinct wrapper type implements the same
interface and delegates `OverrideAuthority` unchanged. For every `UpdateState`
and deprecated `NewAddress` callback, the client-conn wrapper:

1. copies address and endpoint slices;
2. preserves `Addr`, `ServerName`, `Attributes`, and `BalancerAttributes`;
3. clears `Address.Metadata` in both top-level addresses and endpoint addresses;
4. forwards the copied state to grpc-go.

The resolver and its returned lifecycle remain native: resolution errors,
`ResolveNow`, and `Close` delegate unchanged. `WithDisableServiceConfig`
separately ignores resolver service config. `WithNoProxy` ensures grpc-go calls
the selected wrapped builder directly instead of inserting a proxy delegating
resolver whose internal DNS builder would escape the connection-local wrapper.
The wrapper owns no DNS, discovery, retry, balancing, TLS identity, authority,
or authentication behavior; it removes only a deprecated late
transport-header channel.

`resolver.Get`, `resolver.GetDefaultScheme`,
`resolver.AuthorityOverrider`, `resolver.Address.Metadata`, and
`grpc.WithResolvers` are experimental in the pinned gRPC-Go API. Their use is
bounded to `internal/infra/grpcclient/propagation.go`, and a future compile/API
change reopens only this resolver enforcement.

### Logical-call sanitization

Both client interceptors copy outgoing metadata, remove the four reserved keys
case-insensitively, and pass the copied context to the invoker/streamer. They do
not inject correlation fields. This occurs before grpc-go validates logical
outgoing metadata and prevents invalid/stale caller metadata from failing or
reaching the attempt path.

The interceptors also inspect the pinned exported
`grpc.PerRPCCredsCallOption`. When present, they replace its credential with a
thin delegating wrapper that:

- preserves `RequireTransportSecurity`;
- delegates `GetRequestMetadata`;
- copies the result;
- removes the four reserved correlation keys case-insensitively;
- preserves every other authentication field and the original error.

This closes the supported late credential path without owning token
acquisition or authentication policy. The wrapper exists only because per-RPC
credential metadata is added after `StatsHandler.TagRPC`; the propagator cannot
sanitize it. The call-option type is experimental in the pinned gRPC-Go API, so
a gRPC upgrade that changes it must fail compilation and reopen this narrow
enforcement mechanism.

### Per-attempt enforcement

For every native initial or transparent-retry attempt, `otelgrpc.TagRPC` starts
the concrete attempt span and invokes the closed propagator against a fresh
outgoing-metadata copy. The propagator emits the allowed W3C fields and valid
request ID from that attempt context. This gives the remote server the concrete
attempt span as parent and keeps one request ID across transparent retries and
streaming creation.

`grpc.WithDisableServiceConfig` is the enforcement boundary for
`balancer.PickResult.Metadata`: resolver state may still supply addresses, but
its service config cannot select a configured retry or metadata-producing
balancer. The native default balancer remains. The template does not install a
default service config.

Adding resolver-selected service configuration, `grpc.WithDefaultServiceConfig`,
custom balancer/dial options, proxy delegation, re-enabling resolver address
metadata, or another post-propagator metadata producer must reopen Security and
gRPC Client Design.
Same-process code can always bypass a template adapter by constructing its own
raw connection; this package closes the behavior of connections it constructs
rather than acting as a sandbox.

## Generated consumer composition

The template does not have a real downstream contract and therefore does not
add generated output or production dependency wiring.

For a real OpenAPI dependency, the dependency adapter:

1. generates a client from the provider-owned versioned schema with
   oapi-codegen client generation;
2. constructs one bounded `httpclient.Client` with the target and propagation
   policy selected by that dependency owner;
3. passes it through the generated `WithHTTPClient` / `HttpRequestDoer` seam and
   uses `httpclient.Client.BaseURL()` as the already validated server base;
4. passes each operation context unchanged;
5. maps generated transport types/errors into feature-owned types/errors.

For a real protobuf dependency, bootstrap constructs one long-lived
`grpcclient.ClientConn`, passes it to the generated
`New<Service>Client`, and closes it with the owning dependency lifecycle.

The provider adapter still owns authentication, operation deadlines,
retry/idempotency eligibility, remote error mapping, criticality/readiness, and
feature mapping. A feature-owned interface is introduced only when a present
feature needs inversion; the generated client is not wrapped in a speculative
generic port.

## Go ownership and changed surfaces

| Responsibility | Owner and exact surface | Dependency / cleanup / proof |
| --- | --- | --- |
| HTTP policy vocabulary and attempt enforcement | New `internal/infra/httpclient/propagation.go`; `Config`/`New` wiring in `client.go` | Imports existing `reqctx` and OTel propagation; removed automatically with `OUTBOUND_HTTP=none`. Policy, retry, generated-client, and cross-adapter integration tests live with the same removable capability. |
| gRPC policy vocabulary, interceptors, credential guard, policy propagator, and resolver-state sanitizer | New `internal/infra/grpcclient/propagation.go`; `Options`/`New` wiring in `client.go` | Imports existing `reqctx`, grpc credentials/metadata/resolver, and OTel propagation; native proxy delegation is disabled; removed automatically with `GRPC=none`. Tests in `propagation_test.go` plus focused resolver/transparent-retry/live tests. |
| Request-ID validity/context storage | Existing `internal/reqctx`; unchanged | Both optional clients depend inward on the current leaf contract. |
| Inbound extraction/response metadata | Existing `internal/infra/http` and `internal/infra/grpc`; unchanged production behavior | Existing tests remain regression proof; only integration coverage may reuse them. |
| Generated HTTP/protobuf authority | Provider-owned schema and generated dependency package; no template-generated change | `make openapi-check` / `make proto-check` remain drift owners for repository-owned contracts. |
| Concrete dependency policy and mapping | Future `internal/infra/<dependency>` adapter and service bootstrap | Not created without a real neighbor. Existing client cleanup methods remain the lifecycle seam. |
| Documentation | `docs/repo-architecture.md`, `docs/first-production-feature.md`, and profile-owned `docs/grpc.md` | Update current owner wording and executable composition examples; no new guide. |
| Template profiles | Existing `scripts/init-module.sh` removal paths and `scripts/ci/template-init-check.sh` fixtures | No marker or conditional shared package is added. Existing minimal/bounded/gRPC fixtures prove absence/retention. |

The resulting import direction remains acyclic:

```text
provider adapter / bootstrap
  -> optional httpclient or grpcclient
     -> reqctx
     -> standard library + existing OTel/grpc dependencies
```

Neither client imports inbound HTTP/gRPC adapters, generated service contracts,
feature packages, config loading, or bootstrap.

## Rejected alternatives

| Alternative | Why it loses |
| --- | --- |
| Manual generated-client request editors / gRPC metadata in each adapter | Repeats policy, can be forgotten, cannot reliably own HTTP/gRPC attempts, and does not sanitize stale caller fields. |
| Shared `internal/correlation` or `internal/infra/propagation` package | No consumer needs a cross-transport policy value; carrier/enforcement mechanics differ; the package would couple independently removable profiles and survive when neither client does. |
| Put disclosure policy in `internal/reqctx` | Confuses context value validity with remote transport trust and imports protocol/OTel concerns into a core leaf. |
| Enable a composite Trace Context + Baggage propagator | Expands disclosure without an allowlist/integrity/size policy and contradicts the accepted non-goal. |
| Rely on `TargetClass` or `.internal` to infer trust | Network placement does not establish workload identity or application authorization; it also cannot represent trusted HTTPS or untrusted private targets. |
| Interceptor-only gRPC injection | Runs at logical call/stream creation rather than each retry attempt, so it cannot use the concrete attempt span as the remote parent. |
| Custom delegating gRPC `stats.Handler` | A closed propagator passed directly to `otelgrpc` already runs after the attempt span is created. Delegating all stats callbacks adds machinery and still cannot sanitize later credential/picker metadata. |
| StatsHandler/propagator without logical guards | Runs after logical metadata validation and before per-RPC credentials add their metadata, leaving two bypass/failure paths. |
| Preserve resolver-supplied service config | Leaves configured retry, balancer, and late picker metadata without one enforceable owner. A real dependency may reopen this with explicit replay and metadata policy. |
| Restrict target schemes instead of wrapping resolvers | Breaks valid platform/custom discovery and still relies on process-global registration identity. A connection-local wrapper preserves discovery while removing the exact deprecated header channel. |
| Wrap only the apparently effective resolver | Reimplements grpc-go's process-default and unregistered-scheme selection and can lose optional authority override behavior. Wrapping each native candidate lets grpc-go keep the choice. |
| Keep grpc-go environment proxy delegation | The delegating resolver may create an unwrapped DNS resolver behind the selected builder, reopening late address metadata. The base service client connects directly; a required proxy is a separate dependency contract. |
| Allow resolver address metadata with a reserved-key convention | grpc-go appends it after every available policy hook; documentation cannot enforce the wire boundary. |
| Service mesh or full client framework | Provider/fleet-specific, not operated by every template user, and would move rather than remove policy ownership. |

## Failure, compatibility, and rollout boundary

- Unknown policies fail construction before network I/O.
- Missing/invalid context request IDs are omitted; they do not fail the call or
  create an unobservable second ID.
- Errors from the existing HTTP/gRPC transports and from per-RPC credentials
  retain their current identities/ownership.
- Existing derived HTTP clients that omit the new field stop remote W3C
  propagation because zero is `none`; they must opt into
  `PropagationTraceContext` or `PropagationTrustedService`.
- Existing derived gRPC clients using `Options.Propagators` fail compilation
  until migrated to `Propagation`. A custom non-W3C client propagator has no
  template-core migration path.
- Existing derived gRPC clients relying on resolver-supplied service config no
  longer receive configured retry, client-side load balancing, or resolver
  health policy through the base client. Those clients need an explicitly
  designed dependency adapter.
- Existing custom resolvers keep address selection, attributes, and TLS server
  names, but `resolver.Address.Metadata` is cleared. Dependencies that used it
  as an implicit transport-header channel must move to explicit credentials or
  another reviewed client.
- Existing clients that require grpc-go's environment-selected proxy route
  must remain on their current client until a dependency-specific proxy design
  is accepted. Removing `grpc.WithNoProxy` is a valid narrow rollback only
  after another mechanism closes the final resolver metadata boundary.
- The change requires no coordinated server rollout: current inbound servers
  already accept W3C trace context and validated request IDs, while `none`
  remains valid with older servers.
- Rollback restores the previous client construction and propagation behavior;
  it changes correlation disclosure only and has no persisted-data migration.

## Proof and reopen conditions

Implementation proof must cover:

- all three HTTP policies, mixed-case stale fields, valid/invalid/missing
  request IDs, caller immutability, explicit retries, local spans under `none`,
  exact traceparent-to-attempt-span parentage, and existing client bounds;
- all three gRPC policies for unary and streaming creation, logical metadata
  validation, per-RPC credential filtering with auth preservation, a
  representative transparent retry, disabled resolver service config,
  resolver-address-metadata removal, disabled proxy delegation, native
  explicit/default scheme selection and authority override, preserved
  address/server-name/attributes, caller immutability, local spans under
  `none`, and exact attempt parentage;
- a two-service HTTP path with shared trace/request ID and downstream
  cancellation;
- a live gRPC client/server path with request-ID and trace continuity;
- existing optional-profile removal/retention, OpenAPI/protobuf drift, focused
  race proof if tests introduce concurrent observers, and the aggregate gates
  triggered by the final diff.

Reopen only:

- Security when a supported client enables resolver service config or address
  metadata, custom balancer/dial options, baggage, arbitrary propagators, or
  another late metadata producer;
- Reliability when a real dependency requires a changed retry/deadline or
  hidden-attempt observability contract;
- Generated Contract Design when a real downstream schema/distribution model
  appears;
- Specification when request ID becomes identity, authorization, idempotency,
  or a durable cross-request correlation key.
