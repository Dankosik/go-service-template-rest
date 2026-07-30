# gRPC transport support research

status: ready
valid_as_of: 2026-07-29
refresh_triggers:
- before changing pinned gRPC, Protobuf, generator, or Buf versions;
- before claiming that native gRPC works through Railway public HTTP domains;
- when a derived service adds a non-Go generated client language;
- when Protobuf publishes a later edition or changes Edition 2024 support guidance.

## Accepted outcome and evidence boundary

The template needs an easy, production-safe way to add and consume unary,
server-streaming, client-streaming, and bidirectional-streaming gRPC methods
without moving business behavior into transport code or weakening the existing
startup, readiness, drain, telemetry, security, generation, and compatibility
contracts.

This research closes decision-changing facts and candidate coverage. It does
not select the target architecture; System / Integration Design owns that
selection.

Evidence covers:

- the current template repository, Go `1.26.5`, and its existing HTTP,
  diagnostics, config, health, bootstrap, generation, and profile contracts;
- gRPC-Go `v1.82.1`, Go Protobuf `v1.36.11`,
  `protoc-gen-go-grpc v1.6.2`, Buf CLI `v1.72.0`, and the current upstream
  documentation available on the validity date;
- current Railway public, private, and TCP networking contracts plus current
  operator evidence about public-edge trailer handling;
- representative substitutes at the same decision level: direct gRPC-Go,
  Connect, gRPC-Gateway, and full service/RPC frameworks.

It does not establish a service-specific authentication policy, public
internet exposure requirement, latency budget, retry policy, message-size
budget, or generated client language matrix. Those values belong to each
derived service unless the design can define a safe transport default without
inventing business policy.

## Open-item map

| Item | Method | Downstream owner | Disposition |
| --- | --- | --- | --- |
| Where gRPC fits in this template | Current-state baseline | System design | Repository owners already separate transport-free features, concrete adapters, and bootstrap lifecycle; gRPC must preserve those owners. |
| Canonical Go runtime and API shape | External contract + solution discovery | System design | Direct gRPC-Go, Connect, and full frameworks are characterized below. No unsearched viable family is likely to change the decision level. |
| Protobuf syntax and Go API | External contract + conflict challenge | Specification / design | Edition 2023 with schema-owned `API_OPAQUE` exposes fewer cross-language defaults than Edition 2024; proto3 remains compatible but hides the Go API choice in generator flags. |
| Generation, lint, and compatibility | External contract + solution discovery | Delivery design | Buf v2 supplies compiler, lint, breaking, and generation orchestration. Its binary must not be placed in the repository Go tools module. |
| Unary and streaming ergonomics | External contract | Specification / test design | Current generated Go APIs use generics for every streaming cardinality and enforce serial reads and serial writes per stream. |
| Client construction and resilience | External contract | Go ownership / reliability design | `grpc.NewClient` plus one long-lived shared `ClientConn` per target is current guidance. Deadlines and retry eligibility remain operation policy. |
| Server lifecycle and health | Current-state + external contract | Reliability / Go ownership design | Native `GracefulStop` is unbounded and needs a bounded `Stop` fallback under the existing process shutdown budget. Official gRPC health supports serving and drain state. |
| Telemetry and middleware | External contract | Observability / security design | OpenTelemetry stats handlers cover unary and streaming RPCs. Native interceptor chains are sufficient; a middleware framework is not a prerequisite. |
| Listener topology | External contract + solution discovery | System design | Native gRPC on a separate HTTP/2 listener preserves the full gRPC-Go surface. `ServeHTTP` and Connect are shared-HTTP alternatives with different protocol and feature consequences. |
| Railway reachability | External contract + freshness challenge | Delivery / security design | Private networking is encrypted and works as raw TCP. Current public HTTP documentation claims HTTP/2, but current operator evidence shows HTTP/2 demux and stripped trailers. Public native gRPC needs revalidation or raw TCP plus application TLS. |
| Reflection and browser clients | External contract | Specification / security design | Reflection is optional and discloses descriptors; browser support requires Connect/gRPC-Web or a proxy and is not inherent in native gRPC-Go. |
| Validation and REST transcoding | Solution discovery | Later feature owner | Protovalidate and gRPC-Gateway are complements, not prerequisites for transport support. |

## Current template baseline

### Established facts

- `api/proto/` is already reserved as the source-of-truth location for a real
  non-HTTP protobuf contract. The project-structure check rejects an empty
  placeholder directory.
- Business behavior belongs in `internal/<feature>/`; generated contract code
  and gRPC mapping must not enter that package. Concrete transports belong
  under `internal/infra/*`; construction and lifecycle belong in
  `cmd/service/internal/bootstrap/`.
- The process currently owns an application HTTP listener and an optional
  private diagnostics listener. Startup admission happens before traffic is
  accepted as ready. Drain marks shared health state first, waits for readiness
  propagation, shuts the application server down inside one budget, and shuts
  diagnostics down last.
- The bootstrap runtime-server interface matches `http.Server`:
  `Serve(net.Listener) error`, `Shutdown(context.Context) error`, and
  `Close() error`. `grpc.Server` does not implement that contract because it
  exposes `GracefulStop()` without a context and `Stop()` for forced
  termination.
- Runtime configuration is typed, defaulted, environment-loadable, and
  validated in `internal/config`. Optional capabilities are removed during
  initialization through explicit profile markers, and their retained choice
  is recorded in `template.lock`.
- The template initializes PostgreSQL and bounded outbound HTTP independently.
  The default derived service retains neither optional capability. Generated
  outputs are checked for drift from their canonical source.
- The current client-visible REST authority is OpenAPI 3. gRPC-Gateway would
  introduce a second way to derive HTTP behavior and therefore cannot be
  treated as a neutral generator addition.

### Decision implications

- A gRPC runtime cannot bypass shared startup admission, drain state, shutdown
  budgeting, telemetry flush, or config validation.
- A placeholder RPC in the production service would become a false business
  contract. Teaching all four cardinalities therefore needs an isolated
  reference surface or a generated-service exercise rather than speculative
  production APIs.
- If gRPC is optional, initialization purity must cover runtime dependencies,
  config, proto source, generated code, tests, tools, docs, and CI targets; a
  half-removed capability would make the minimal service dishonest.

### Stop rationale

The authoritative repository owners and affected lifecycle path are known.
Additional repository scanning is unlikely to change placement or lifecycle
ownership; design still must choose the exact adapter and profile shape.

## gRPC-Go execution contract

### Established facts

- A protobuf service can define four RPC cardinalities: unary,
  server-streaming, client-streaming, and bidirectional-streaming. Current
  `protoc-gen-go-grpc` output uses generic stream interfaces by default.
- Generated client calls and server handlers are concurrency-safe across RPCs.
  For one stream, one goroutine may read while another writes, but multiple
  concurrent reads or multiple concurrent writes are not safe.
- Each server RPC handler runs in its own goroutine. A handler must therefore
  honor context cancellation and must not retain request-owned mutable data
  past its lifetime without explicit ownership transfer.
- `grpc.NewClient` is the current constructor. It creates a virtual channel and
  performs no network I/O. `ClientConn` manages resolution, connection
  establishment, reconnect, and load balancing and is intended to be shared.
  `Dial` and `DialContext` are deprecated anti-patterns for new code.
- gRPC sets no useful universal application deadline. Clients should set
  realistic operation deadlines; Go propagates incoming deadlines to outgoing
  gRPC calls by default. Server work must observe cancellation.
- Transparent retry exists, and explicit retry is configured per method
  through service config. Retrying a method is safe only when its idempotency
  and side-effect semantics allow replay. Global retry or `WaitForReady` would
  hide failures and is not a safe default.
- Native HTTP/2 flow control provides transport backpressure, but it does not
  make an application-created unbounded queue safe. Streaming is beneficial
  for a logical long-lived flow; it should not replace independent unary calls
  merely to save connection setup because a shared `ClientConn` already
  reuses connections.
- `GracefulStop` stops accepting new RPCs and waits for pending RPCs without a
  context bound. `Stop` cancels active RPCs immediately. Production graceful
  shutdown therefore needs a timer or context bounded fallback.
- The official health implementation can publish per-service and overall
  serving state and has `Shutdown()` to change all statuses to `NOT_SERVING`.
- `grpc.Server.ServeHTTP` is experimental relative to the native server path,
  uses the Go HTTP/2 stack, and lacks some native gRPC features. Sharing one
  `net/http` listener is therefore not behaviorally identical to a native
  gRPC listener.

### Inference

One process-wide HTTP timeout cannot safely own both ordinary HTTP requests and
long-lived gRPC streams. Stream duration, idle behavior, and operation
deadlines must remain explicit method or deployment policy.

### Counter-evidence considered

- Creating a connection per call looks simpler locally, but contradicts the
  virtual-channel contract and adds connection churn without improving
  isolation.
- Enabling retry globally looks resilient, but cannot distinguish replay-safe
  methods from non-idempotent mutations.
- One shared HTTP listener looks operationally smaller, but changes the gRPC
  implementation path and loses native features.

### Downstream implications

Design needs:

- one reusable client construction seam that accepts explicit credentials,
  telemetry, target, and message bounds without owning operation deadlines or
  retry policy;
- server adapters for both unary and streaming interceptors;
- a bounded graceful-stop adapter plus health-state transition during drain;
- tests that exercise all four cardinalities, cancellation, serial stream
  access, health transition, and forced shutdown.

### Sources and limits

- [gRPC-Go package documentation](https://pkg.go.dev/google.golang.org/grpc@v1.82.1)
- [gRPC-Go generated-code reference](https://grpc.io/docs/languages/go/generated-code/)
- [gRPC-Go concurrency contract](https://github.com/grpc/grpc-go/blob/v1.82.1/Documentation/concurrency.md)
- [gRPC-Go client anti-patterns](https://github.com/grpc/grpc-go/blob/v1.82.1/Documentation/anti-patterns.md)
- [gRPC deadlines](https://grpc.io/docs/guides/deadlines/)
- [gRPC cancellation](https://grpc.io/docs/guides/cancellation/)
- [gRPC retry](https://grpc.io/docs/guides/retry/)
- [gRPC performance guidance](https://grpc.io/docs/guides/performance/)
- [gRPC flow control](https://grpc.io/docs/guides/flow-control/)
- [gRPC graceful shutdown](https://grpc.io/docs/guides/server-graceful-stop/)
- [gRPC-Go health package](https://pkg.go.dev/google.golang.org/grpc/health@v1.82.1)

These sources define library behavior, not a derived service's retry-safe
methods, stream-duration policy, authentication, or latency budget.

### Stop rationale

The current public API, maintainer source, anti-pattern, concurrency, and
failure-path documentation agree. Further library tutorials are unlikely to
change the lifecycle or client-ownership implications.

## Protobuf schema and generated Go API

### Established facts

- Protobuf Editions are the successor to proto2 and proto3 syntax. The latest
  released edition is 2024.
- The Protobuf team recommends the Go Opaque API for new development. Edition
  2024 enables it by default. Edition 2023 can select the same API explicitly
  in the schema with:

  ```proto
  edition = "2023";

  import "google/protobuf/go_features.proto";
  option features.(pb.go).api_level = API_OPAQUE;
  ```

- Opaque API choice affects generated Go message access, not wire format or
  gRPC method cardinality. Hand-written Go uses getters, setters, or immediate
  builders rather than depending on exported generated fields.
- Edition 2024 also changes defaults outside Go, including C++ string and enum
  APIs, Java class placement and outer naming, symbol visibility, and naming
  enforcement. Adopting it as a Go-only convenience silently accepts a wider
  cross-language policy.
- Proto3 can generate Opaque Go messages through a generator flag, but the
  schema no longer carries that policy and every generation consumer must
  reproduce it.
- Current local versions support Edition 2023 and Edition 2024:
  Go Protobuf `v1.36.11`, `protoc-gen-go-grpc v1.6.2`, and installed
  `protoc 34.1`. Buf fixed Edition 2024 support before its current `v1.72.0`.
- Standard Buf breaking analysis does not own changes to custom Go
  `api_level`. Generated drift and compilation must therefore prove that
  policy.
- Durable schema evolution rules remain independent of syntax:
  never reuse field numbers, reserve removed numbers and names, start enums
  with an `UNSPECIFIED` zero value, avoid required fields, model presence
  explicitly, and keep RPC messages independent from storage schemas.

### Conflict challenge and disposition

The leading hypothesis was “latest Edition 2024 is the natural template
baseline.” Independent semantic challenge falsified the Go-only justification:
Edition 2024 couples the desired Opaque API to unrelated cross-language
defaults. Edition 2023 with explicit `API_OPAQUE` preserves the desired Go API
while keeping those future client-language choices visible.

This does not forbid Edition 2024. Its decision-flip condition is an explicit
template commitment to all Edition 2024 cross-language defaults plus a
generator/client compatibility matrix for every promised language.

### Downstream implications

Specification can require schema-owned Opaque Go generation without deciding
future non-Go client defaults. Generated code and examples must themselves use
Opaque-safe access so drift catches an accidental API-level change.

### Sources and limits

- [Protobuf Editions overview](https://protobuf.dev/editions/overview/)
- [Go Opaque API FAQ](https://protobuf.dev/reference/go/opaque-faq/)
- [Edition 2024 announcement](https://protobuf.dev/news/2025-06-27/)
- [Protobuf best practices](https://protobuf.dev/best-practices/dos-donts/)
- [Protobuf style guide](https://protobuf.dev/programming-guides/style/)
- [Protobuf field presence](https://protobuf.dev/programming-guides/field_presence/)
- [Protobuf version support](https://protobuf.dev/support/version-support/)
- [Buf breaking rules](https://buf.build/docs/breaking/rules/)

No non-Go generated client language is currently promised by the repository,
so the wider Edition 2024 compatibility matrix is deliberately unknown rather
than assumed.

### Stop rationale

Current Protobuf authority and current generator support resolve the Go API
choice and expose the cross-language consequence. More searches cannot select
future client languages for a derived service.

## Schema toolchain and compatibility gates

### Established facts

- Buf v2 can build schemas with its internal compiler, apply `STANDARD` lint,
  apply conservative `FILE` breaking rules, format files, and orchestrate
  local or remote code-generation plugins.
- Local plugins can be invoked as a command plus arguments. This permits
  repository-pinned `protoc-gen-go` and `protoc-gen-go-grpc` tools without
  requiring plugin binaries on ambient `PATH`.
- Remote plugins can pin plugin version and revision but require the Buf
  Schema Registry during generation. They trade local setup for an external
  availability and custody dependency.
- Buf explicitly advises against installing the Buf CLI through a project's
  `tools.go` or `go tool` module because its dependencies may be resolved
  against incompatible project versions. For CI it recommends a release
  binary or image; `go install ...@version` is also isolated from the local
  module.
- `protoc-gen-go-grpc` defaults
  `require_unimplemented_servers=true`, preserving forward-compatible server
  interface evolution when implementations embed the generated
  `Unimplemented...Server`.
- Breaking analysis compares schemas, not generated Go. A complete gate needs
  lint, breaking against the chosen Git base, generation, clean drift, Go
  compilation, and runtime contract tests.

### Counter-evidence considered

- Raw `protoc` plus shell flags is viable but reintroduces compiler include
  paths, plugin discovery, lint, and breaking tooling the template would need
  to build separately.
- Remote plugins are reproducible by declared revision, but make ordinary
  generation network-dependent.
- Putting Buf in `tools/go.mod` would match existing Go tools visually but
  directly contradicts Buf's current installation contract.

### Downstream implications

Delivery design needs to choose:

- an externally pinned Buf binary path for local/CI use;
- repository-pinned local Go generator plugins or explicitly accepted remote
  plugins;
- a no-source behavior that keeps `proto-check` safe when no owned `.proto`
  contract exists;
- a base-reference contract for breaking checks that fails honestly when the
  base schema is unavailable.

### Sources and limits

- [Buf CLI installation](https://buf.build/docs/cli/installation/)
- [Buf v2 module, lint, and breaking configuration](https://buf.build/docs/configuration/v2/buf-yaml/)
- [Buf v2 generation configuration](https://buf.build/docs/configuration/v2/buf-gen-yaml/)
- [protoc-gen-go-grpc documentation](https://pkg.go.dev/google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2)

The local checkout does not currently have Buf installed. A temporary pinned
binary or CI installation is required for implementation proof.

### Stop rationale

The current Buf contract explicitly resolves the only surprising ownership
question—Buf cannot share the tools module. Remaining choices are design
trade-offs, not missing tool facts.

## Observability, errors, health, and security

### Established facts

- `otelgrpc.NewServerHandler` and `otelgrpc.NewClientHandler` are gRPC stats
  handlers that instrument both unary and streaming RPCs. Interceptors remain
  appropriate for application policy such as authentication, request
  correlation, sanitized logging, panic recovery, and domain-error mapping.
- gRPC-Go natively chains unary and stream interceptors; a general middleware
  framework is not required for composition.
- gRPC status codes are the transport error contract. A server should map
  stable domain errors to deliberate codes and optional structured details,
  rather than leak raw internal or downstream errors.
- Metadata carries authentication, tracing, and request context. It is not a
  payload channel; servers commonly enforce a bounded header list, and
  sensitive metadata or protobuf payloads must not be logged.
- TLS transport credentials are the canonical public/untrusted-network
  mechanism. `insecure.NewCredentials` disables transport security and is
  only honest when an owned private encrypted network supplies the trust
  boundary.
- gRPC reflection lets generic tools discover descriptors, but also publishes
  the service schema. It is opt-in policy, not a production prerequisite.
- Keepalive settings are a client/server contract. Aggressive client pings can
  cause disconnects; upstream guidance discourages enabling them without
  coordination or at very short intervals.
- Default receive limits are bounded but send limits are much larger. A
  template should not imply that arbitrarily large messages are safe merely
  because the library can encode them.

### Downstream implications

- Telemetry should use stats handlers, with low-cardinality full method names
  and no payload or credential attributes.
- Authentication must expose an interceptor seam but cannot be invented by the
  generic template.
- Server and client message bounds should be explicit and symmetric by
  default, with per-service changes justified by workload evidence.
- Reflection and keepalive should default off unless a derived service owns
  the exposure and peer contract.

### Sources and limits

- [OpenTelemetry gRPC instrumentation](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc@v0.69.0)
- [gRPC interceptors](https://grpc.io/docs/guides/interceptors/)
- [gRPC status codes](https://grpc.io/docs/guides/status-codes/)
- [gRPC metadata](https://grpc.io/docs/guides/metadata/)
- [gRPC authentication](https://grpc.io/docs/guides/auth/)
- [gRPC-Go credentials](https://pkg.go.dev/google.golang.org/grpc/credentials@v1.82.1)
- [gRPC reflection](https://grpc.io/docs/guides/reflection/)
- [gRPC keepalive](https://grpc.io/docs/guides/keepalive/)

The sources establish transport mechanisms, not a derived service's identity
provider, authorization rules, certificate authority, or acceptable payload.

### Stop rationale

Mechanism and unsafe defaults are established. Service-specific policy cannot
be learned from generic gRPC sources and is correctly left to the owning
feature or deployment.

## Deployment and Railway boundary

### Established facts

- Railway private networking uses internal DNS over encrypted WireGuard
  tunnels, stays within the environment, and does not expose the port
  publicly. Raw TCP protocols can use it.
- Railway public networking documentation says the edge accepts HTTP/1.1 and
  HTTP/2, requires TLS, caps combined headers at 32 KiB, and limits ordinary
  HTTP requests to 15 minutes while data continues to transfer.
- Current Railway employee/operator evidence reports that the public edge
  demultiplexes HTTP/2 to HTTP/1.1 and that gRPC response trailers are stripped
  on an affected path. The same report confirms private internal routing
  works. The thread later auto-closed without evidence of a platform fix.
- Railway TCP Proxy exposes a raw internal TCP port through a generated host
  and external port, can coexist with HTTP exposure, and supports custom DNS.
  The assigned external port remains required, and custom hostnames can affect
  client hostname validation.

### Conflict and disposition

The official public-networking page's broad HTTP/2 support statement is not
sufficient proof of end-to-end native gRPC semantics. gRPC requires HTTP/2
trailers, and current operator evidence demonstrates a contrary path. Research
therefore cannot support a claim that Railway public HTTP domains are a valid
native gRPC ingress.

This is freshness-sensitive rather than a permanent platform rejection.
Refresh at the earliest of:

- a derived service requesting public native gRPC;
- Railway documenting end-to-end gRPC/trailer support;
- a safe unary plus streaming probe through the exact intended public domain.

### Downstream implications

- Private service-to-service gRPC can rely on Railway's encrypted private
  network only when deployment policy ensures the listener is not publicly
  exposed and authentication requirements are independently satisfied.
- Public native gRPC needs a raw TCP path with application-owned TLS and
  certificate name validation, or a newly verified Railway HTTP edge.
- Browser/public HTTP compatibility points toward Connect, gRPC-Web, or a
  gateway/proxy, not an assumption that native grpc-go is browser-compatible.
- A public edge's 15-minute request limit is incompatible with a promise of
  indefinite streams.

### Sources and limits

- [Railway private networking](https://docs.railway.com/networking/private-networking)
- [Railway public networking limits](https://docs.railway.com/networking/public-networking/specs-and-limits)
- [Railway TCP Proxy](https://docs.railway.com/networking/tcp-proxy)
- [Current Railway gRPC trailer incident discussion](https://station.railway.com/questions/g-rpc-response-trailers-being-stripped-on-71b96615)

The incident discussion is operator evidence, not a durable provider contract.
No live Railway project or public domain is in scope, so no representative
deployment probe is currently possible.

### Stop rationale

The conflict is explicit and has an objective refresh checkpoint. More generic
provider pages cannot prove the exact missing end-to-end trailer behavior.

## Candidate map

### Neutral frame

The live decision is not “which gRPC framework is popular.” It is which
mechanism gives derived services a source-owned protobuf contract, idiomatic
Go client/server APIs for all cardinalities, bounded lifecycle, transport
security, telemetry, compatibility gates, and low template cost while
preserving the repository's existing owners.

### Inbound runtime substitutes

| Candidate | Relationship | Evidence-backed implication | Flip condition |
| --- | --- | --- | --- |
| Direct `grpc-go` native server | Substitute | Canonical Go implementation, full native feature surface, generics-based generated APIs, separate HTTP/2 listener, explicit lifecycle adapter required. | Loses if browser/shared-HTTP compatibility is an accepted primary behavior rather than an optional complement. |
| Connect-Go `net/http` handler | Substitute | Serves Connect, gRPC, and gRPC-Web through standard HTTP handlers and supports all streaming shapes; easier browser and shared-ingress fit, but adds protocol semantics and is not the full native grpc-go server path. | Wins only if multi-protocol/browser/shared listener is an accepted requirement that dominates native feature parity. |
| Full framework such as Kratos, go-kit, or go-zero | Substitute | Adds lifecycle, transport, discovery, endpoint, or scaffolding abstractions, but overlaps the template's existing bootstrap and feature boundaries and enlarges the mandatory programming model. | Reopens if the template intentionally adopts the framework as its overall service architecture, not merely gRPC support. |
| Custom HTTP/2/protobuf RPC stack | Substitute | No missing capability justifies owning framing, status, flow control, health, interoperability, and generator compatibility. | No credible flip condition under the accepted outcome. |

### Complementary mechanisms

| Candidate | Relationship | Disposition |
| --- | --- | --- |
| gRPC-Gateway | Optional complement | Generates REST transcoding from annotations. It would reopen REST contract authority because this template currently owns REST through OpenAPI 3. |
| Protovalidate | Optional complement | Declarative schema validation with CEL. Useful when a service owns validation constraints; not required for transport startup or streaming. |
| `go-grpc-middleware/v2` | Optional complement | Supplies common interceptors, but native chains plus local narrow policy are enough until a concrete middleware behavior is required. |
| Buf Schema Registry | Optional managed complement | Can own modules, dependencies, remote plugins, and SDKs. It adds account, availability, custody, and publication policy not required for local source ownership. |
| Reflection | Optional operational complement | Helps `grpcurl` and tooling discover descriptors, but increases schema disclosure and is not required when clients have source/descriptors. |

### Toolchain substitutes

| Candidate | Relationship | Evidence-backed implication |
| --- | --- | --- |
| Buf v2 + local pinned Go plugins | Substitute | One compiler/lint/breaking/generate front end; generation works without BSR, plugins stay repository-pinned, Buf binary remains separately installed. |
| Buf v2 + remote pinned plugins | Substitute | Minimal plugin installation and declared revisions; generation becomes registry/network dependent. |
| Raw `protoc` + local plugins + separate lint/breaking tools | Substitute | Standards-compatible but recreates orchestration and compatibility gates with more ambient setup. |

### Candidate-space saturation

The search covered existing repository reuse, Go's canonical gRPC and
Protobuf implementations, native platform networking, maintained OSS
alternatives, managed schema tooling, and custom implementation. Further
searches by “Go RPC framework,” “gRPC-Web,” “protobuf generation,”
“transcoding,” “streaming RPC,” and “Railway HTTP/2” converged on the same
families or complements. No uncharacterized candidate occupies a materially
different live decision slot.

## Downstream proof obligations

Test design must assign exact proof for:

- reproducible schema lint, generation, clean drift, and base-to-head breaking
  detection;
- Opaque generated API compilation and generated-server forward compatibility;
- unary, server-streaming, client-streaming, and bidirectional-streaming
  client/server behavior;
- cancellation and deadlines on unary and streaming work;
- explicit status-code and metadata mapping without internal error or secret
  disclosure;
- official gRPC health before admission, while serving, and during drain;
- bounded graceful stop with forced cancellation after the shutdown budget;
- simultaneous HTTP, gRPC, and diagnostics lifecycle, with diagnostics last;
- OpenTelemetry propagation and spans/metrics for unary and streaming RPCs;
- configured message and connection bounds;
- a reusable client connection that closes cleanly;
- initialization purity for both gRPC-disabled and gRPC-enabled derived
  services;
- current Railway private reachability guidance and a public-ingress
  revalidation gate rather than an unsupported production claim.

## Research closeout

Every triggered current-state, external-contract, solution-discovery, and
freshness surface has either been inspected or recorded as an explicit
downstream policy/proof gap. The leading Edition 2024 and Railway public-edge
hypotheses were challenged and narrowed. The next owner can derive design
drivers and select the architecture without repeating the search.
