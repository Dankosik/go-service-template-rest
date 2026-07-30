# Optional production-ready gRPC client and server support

status: ready

## Scope and non-goals

### In scope

- Add an optional template capability selected during initialization as
  `GRPC=none|enabled`; the default is `none`.
- When enabled, let a derived service expose native gRPC alongside the existing
  REST API and private diagnostics endpoint without changing their contracts.
- Make protobuf source, generated Go client/server APIs, lint, compatibility
  checks, generation, and generated drift part of the repository's ordinary
  development and CI workflow.
- Support unary, server-streaming, client-streaming, and bidirectional-streaming
  RPCs with the current generic Go APIs.
- Supply reusable server and client transport foundations for lifecycle,
  health, correlation, telemetry, bounded messages, transport credentials,
  cancellation, and deliberate status mapping.
- Include an isolated, executable reference that demonstrates all four RPC
  cardinalities. It is upstream teaching material and follows the existing
  `REFERENCE_EXAMPLE=remove|keep` ownership rule.
- Document local use, service-to-service use, public exposure, contract
  evolution, and the first production RPC vertical slice.

### Non-goals

- Replacing the existing REST/OpenAPI transport or deriving REST behavior from
  protobuf annotations.
- Adding gRPC-Gateway, gRPC-Web, Connect, service discovery, a service mesh,
  the Buf Schema Registry, or a full service framework to the base capability.
- Inventing a derived service's authentication, authorization, idempotency,
  retry eligibility, keepalive, stream duration, payload size, rate limit, or
  business validation policy.
- Publishing generated SDKs or promising a non-Go client-language matrix.
- Publicly deploying or configuring a live Railway project.
- Defining placeholder business RPCs in a generated service.

## Behavior and contract delta

### CAP-1 — Initialization and capability ownership

`scripts/init-module.sh` accepts `GRPC=none|enabled`.

- Omitted `GRPC` is equivalent to `GRPC=none`.
- `GRPC=none` removes the gRPC runtime, client foundation, config, generator
  plugins, generated contracts, gRPC-specific tests, and gRPC reference
  implementation from the derived service. Ordinary REST behavior and its
  proof remain unchanged. Stable aggregate targets may remain as explicit
  no-ops so CI composition does not depend on the selected profile.
- `GRPC=enabled` retains those surfaces and records `grpc = "enabled"` in
  `template.lock`.
- Unsupported values fail before mutating the checkout.
- Repeating initialization with the same inputs is idempotent.
- The retained agent workflow and unrelated profile choices do not change.

Profile purity applies to capability-owned source, direct imports and module
requirements, runtime listeners, configuration, generated contracts,
generator plugins, tests, and commands with material behavior. It does not
require removing an unrelated transitive `grpc` or `protobuf` module already
needed by a retained dependency such as telemetry.

The template source may carry an upstream reference protobuf contract, but a
derived production service does not inherit it unless `GRPC=enabled` and
`REFERENCE_EXAMPLE=keep`. Removing the reference does not remove the enabled
gRPC foundation.

### CAP-2 — Contract authority and generated output

For a derived service's client-visible gRPC API:

- owned `.proto` files under `api/proto/` are authoritative;
- generated Go is derived, tracked, and never edited by hand;
- `api/proto/` remains absent until the first owned `.proto` file exists;
- generated code cannot override the schema or a ready behavior specification;
- schema packages are versioned (`...v1`, `...v2`, and so on) before they are
  published to consumers;
- each RPC owns distinct request and response message types, even when the
  first version has no fields;
- removed field numbers and names are reserved and never reused;
- enums have an `*_UNSPECIFIED = 0` value;
- persistence rows and protobuf messages remain separate contracts.

New Go protobuf contracts use Edition 2023 and declare the Opaque Go API in the
schema. Edition 2024 is allowed only after the owning service explicitly
accepts its cross-language defaults and proves every promised generator/client
language. Existing proto2/proto3 contracts may migrate separately; enabling the
capability does not rewrite them.

### CAP-3 — Developer workflow

An enabled derived service has documented commands with these observable
outcomes:

- format owned protobuf source;
- reject non-canonical formatting without mutating source;
- require documentation for public schema elements covered by Buf `COMMENTS`;
- lint it with the repository policy;
- generate Go messages and gRPC client/server bindings reproducibly;
- fail when tracked generated output differs from the authoritative source;
- compare the current schema with an explicit Git base and reject breaking
  changes under Buf's `FILE` compatibility rule set;
- run the complete protobuf contract gate through one stable aggregate command.

The stable contract gate succeeds as an explicit no-op before the first owned
`.proto` file. A generation command also reports that there is no owned source
rather than creating an empty `api/proto/` directory.

When the comparison base has no protobuf contract because this is the first
published API, the breaking check reports “not applicable.” An invalid or
unreadable base reference is an error; it must not be mistaken for an absent
base contract.

Tool versions are declared by repository-owned inputs. Ordinary generation
does not require a Buf account or registry access.

### SRV-1 — Activation, coexistence, and startup admission

`GRPC=enabled` retains both client and server foundations but does not expose a
network listener by itself. The initialized runtime default is
`grpc.server.enabled=false`, so a client-only service opens no gRPC port.

The operator activates the listener with `grpc.server.enabled=true`. Activation
also requires:

- a non-empty listen address;
- an explicit transport-security mode of `plaintext` or `tls`;
- for `plaintext`, an explicit acknowledgement that the deployment supplies
  the allowed loopback or encrypted-private-network boundary;
- for `tls`, readable certificate and private-key inputs that form a valid
  server credential.

There is no implicit transport-security mode. Missing, conflicting, or invalid
credential configuration fails startup before readiness and does not fall back
to plaintext.

When the runtime listener is explicitly enabled, one service process can serve:

- the existing REST API;
- the native gRPC API;
- the existing private diagnostics endpoint.

The three surfaces retain distinct reachability and protocol identities. A
gRPC setup failure is a process startup failure; the process must not continue
as an apparently healthy REST-only service after the operator asked it to
serve gRPC.

The gRPC listener may accept connections during process startup, but official
gRPC health reports `NOT_SERVING` until the same startup admission that makes
REST readiness healthy has completed. After admission it reports `SERVING`.
The existing HTTP liveness and readiness semantics remain unchanged.

### SRV-2 — Drain and bounded shutdown

On process drain:

1. shared readiness enters the existing draining state;
2. gRPC health becomes `NOT_SERVING`;
3. the existing readiness-propagation interval is honored;
4. new business RPCs are no longer admitted;
5. active HTTP requests and gRPC RPCs receive the remaining shared application
   shutdown budget;
6. active gRPC RPCs that outlive that budget are canceled;
7. diagnostics shuts down after both application transports;
8. telemetry flush remains the final process closeout stage.

Repeated drain or shutdown calls are safe and do not extend the original
budget. A long-lived stream has no right to keep the process alive after the
configured shutdown deadline.

### SRV-3 — RPC cardinalities and stream outcomes

The enabled transport supports the generated Go signatures for:

- unary: one request and one response;
- server-streaming: one request and zero or more responses;
- client-streaming: zero or more requests and one terminal response;
- bidirectional-streaming: independent ordered request and response streams.

For one stream, reads are serial and writes are serial. One read may run
concurrently with one write. The transport and reference code do not launch
multiple concurrent readers or multiple concurrent writers for the same
stream.

Every handler observes the RPC context. Cancellation or deadline expiry stops
owned work and releases request-owned resources. A method may define its own
stream duration or idle policy later; the base transport does not impose an
HTTP-style universal request timeout on streams.

Messages delivered before a terminal non-OK status are partial observations,
not an implicit transaction commit. A method that gives partial messages
durable business meaning must say so in its own contract.

### SRV-4 — Status and failure disclosure

Transport and validation failures use canonical gRPC status codes. Feature
errors are mapped deliberately at the gRPC adapter boundary.

- cancellation returns `CANCELLED`;
- an expired effective deadline returns `DEADLINE_EXCEEDED`;
- invalid request shape returns `INVALID_ARGUMENT`;
- `RESOURCE_EXHAUSTED` means an RPC that reached application admission was
  rejected before business handling because a local configured concurrency,
  message, quota, or capacity bound was exhausted and the transport can still
  carry an RPC status;
- `UNAVAILABLE` means an admitted RPC could not complete because this process
  is draining/not serving or a required service or dependency is transiently
  unavailable and no more specific feature-owned status applies;
- an unmapped error or recovered panic returns a sanitized `INTERNAL`.

No response status, detail, metadata, log, metric, or span may disclose an
internal error string, credential, SQL fragment, private hostname, request
payload, or protobuf payload by default.

A feature can add more precise status mappings only from its own documented
error identities. It must not pass a downstream service's gRPC status through
unchanged unless the feature contract explicitly owns that equivalence.

### SRV-5 — Correlation and observability

Every business RPC:

- accepts `x-request-id` metadata under the same validity contract as REST;
- generates a replacement when the value is absent or invalid;
- makes the accepted identifier available to feature code and returns it as
  response metadata;
- propagates W3C trace context;
- emits gRPC client/server telemetry for unary and streaming RPCs;
- records the bounded full gRPC method name, final status, duration, trace ID,
  span ID, and request ID in the owning observation surfaces;
- does not use peer-supplied metadata or message contents as metric labels.

Official health checks may suppress routine access logs just as HTTP health
checks do, while remaining observable through health state and aggregate
telemetry.

### SRV-6 — Bounds and overload

The enabled server has finite, documented defaults for:

- accepted transport connections;
- concurrent RPCs across the process;
- concurrent streams per connection;
- received metadata/header bytes;
- received and sent message bytes.

The reusable client has finite, documented defaults for received
metadata/header bytes and received and sent message bytes. Template-owned
outgoing metadata is itself bounded; feature-owned additional metadata remains
the feature's contract and must fit the peer's advertised receive bound.

A derived service may raise any bound only through validated configuration and
must then own the matching workload, memory, and concurrency proof. Zero,
negative, overflowed, or internally inconsistent values fail configuration;
they do not mean “unlimited.”

An RPC rejected by the process-wide application admission bound does not enter
the business handler and returns `RESOURCE_EXHAUSTED`. A connection cap or an
HTTP/2 metadata/stream safety bound may reject the connection or reset the
stream before an RPC reaches interceptors; that pre-RPC failure uses grpc-go's
native transport outcome and is not promised an application status code.

The application does not create an unbounded queue per stream or per RPC.
Native HTTP/2 flow control remains active, but it is not presented as a
substitute for application ownership of buffered work.

Method-specific quotas or load-shedding policies may be added by a derived
service. The base finite process-wide bound is a safety ceiling, not a claim
that one concurrency value provides fair scheduling between unary and
long-lived streaming work.

### CLI-1 — Reusable client behavior

The enabled capability supplies an idiomatic way to construct generated Go
clients with these guarantees:

- it uses `grpc.NewClient`;
- one returned `ClientConn` is safe to share across generated clients and
  concurrent RPCs;
- construction performs no readiness probe and does not claim the target is
  reachable;
- the caller owns closing the connection;
- transport credentials are explicit;
- client telemetry and message bounds match the server contract;
- individual operations own their deadline and cancellation;
- the template installs no retry service config and does not opt into
  `WaitForReady`; grpc-go's pre-commit transparent retry and any
  resolver-supplied service config remain native behavior;
- an application retry policy is enabled only by the method owner from
  documented idempotency and replay semantics.

The foundation does not wrap every generated method in a template-specific
interface. Feature code consumes generated clients or a narrow feature-owned
port when inversion is actually needed.

### SEC-1 — Reachability, credentials, and authentication boundary

The capability distinguishes three environments:

| Environment | Required behavior |
| --- | --- |
| In-process or loopback development/test | Explicit plaintext credentials are allowed. |
| Operator-owned encrypted private network | Plaintext gRPC is allowed only when network policy prevents public reachability and supplies transport encryption; application authentication remains a separate service decision. |
| Public or otherwise untrusted network | Application-owned TLS with hostname verification is required before business RPCs are exposed; the owning service must also define authentication and authorization. |

Plaintext is never silently selected for a server or client. Enabling the gRPC
capability does not claim that business RPCs are authenticated. The first
production RPC guide requires the service owner to disposition authentication
and authorization before public exposure.

Reflection is disabled by default. A derived service may enable it only after
accepting descriptor disclosure for that environment. Keepalive is not enabled
by default and requires a client/server peer policy.

### DEP-1 — Railway behavior and claim boundary

For Railway:

- private service-to-service gRPC uses the private DNS name and explicit gRPC
  port; the deployment must keep that port off public ingress;
- the documented WireGuard private network may satisfy transport encryption,
  but does not supply application identity or authorization;
- the template does not claim native gRPC support through a Railway public HTTP
  domain while current end-to-end trailer behavior is unresolved;
- public native gRPC requires either a freshly proven Railway HTTP path or a
  TCP Proxy path with application-owned TLS and correct certificate hostname
  validation;
- indefinite streaming is not promised through a public HTTP edge with a
  finite request-duration limit.

Railway project IDs, domains, TCP proxy ports, certificates, and secrets remain
operator-owned external configuration.

### REF-1 — Teaching and first-feature ergonomics

The upstream reference is executable and demonstrates:

- one service with unary, server-streaming, client-streaming, and
  bidirectional-streaming methods;
- Opaque message construction and access;
- server registration and forward-compatible embedding;
- a shared client connection;
- deadlines and cancellation;
- serial stream reads/writes and `CloseSend`;
- deliberate status mapping;
- health state and bounded shutdown;
- request ID and trace propagation;
- both plaintext loopback and TLS construction seams without bundled
  production credentials.

The first-production-RPC guide starts from an absent production `api/proto/`
directory and ends with one owned vertical slice: behavior, versioned schema,
generation, adapter implementation, bootstrap registration, client use,
focused proof, and compatibility check.

## Invariants and edge cases

- REST-only initialization must not retain capability-owned gRPC code, direct
  module requirements, runtime/config surfaces, generated contracts,
  generator plugins, or tests. Unrelated retained dependencies may still bring
  gRPC or Protobuf modules transitively, and the stable protobuf aggregate may
  remain as a named no-op.
- Enabling gRPC must not change existing REST routes, OpenAPI authority,
  diagnostics reachability, readiness meaning, or shutdown margin.
- gRPC health is derived from the same readiness/drain authority as HTTP; it is
  not a second independent readiness truth.
- A listener bind failure, server exit, background failure, or forced-shutdown
  timeout is visible to the process owner and cannot be converted to success.
- A missing protobuf source is an explicit no-op; malformed or incompatible
  source is an error.
- Adding a method to a published service must not break existing server
  implementations that follow the generated embedding contract.
- Generated drift, lint, breaking analysis, and Go compilation are distinct
  checks; one passing does not stand in for another.
- A canceled client may already have caused a server side effect. The base
  transport never represents cancellation as proof that no work happened.
- Retry never becomes safe merely because a failure code is retryable.
- Health and reflection are not business APIs and do not justify exposing the
  business listener publicly.

## Decisions, constraints, and authorities

- `api/proto/` is authoritative for owned protobuf contracts.
- Ready feature specifications own business meaning; protobuf encodes that
  meaning but does not invent it.
- gRPC status mapping is transport behavior; domain error identity remains
  feature-owned.
- The current Go Protobuf and gRPC generators own generated API shape.
- Edition 2023 plus schema-owned Opaque API is the new-contract baseline.
- Buf `STANDARD` lint and `FILE` breaking rules own protobuf style and initial
  compatibility detection; generated drift and Go proof cover generated API
  consequences Buf does not analyze.
- Existing `internal/health` readiness/drain state remains the process health
  authority.
- Existing process bootstrap owns listener lifecycle and shutdown ordering.
- Existing OpenTelemetry SDK ownership remains unchanged.
- Platform network policy owns reachability; transport credentials and
  application authentication remain explicit, separate controls.

## Success criteria and proof expectations

The change is acceptable only when current evidence proves:

1. `GRPC=none` produces a clean, compiling REST-only derived service with no
   capability-owned gRPC residue while tolerating unrelated transitive module
   dependencies and the stable no-op aggregate.
2. `GRPC=enabled` produces a clean, compiling client-only service with no gRPC
   listener by default; explicit valid server activation produces a service
   whose gRPC health transitions with shared startup and drain state, while
   incomplete or invalid transport credentials fail startup without a
   plaintext fallback.
3. The isolated reference compiles and passes an in-process or loopback
   client/server test for all four RPC cardinalities.
4. Focused proof covers deadline/cancellation, request-ID propagation,
   sanitized statuses, telemetry, connection/RPC/stream/metadata/message
   bounds, graceful stop, and forced stop.
5. HTTP, gRPC, and diagnostics can run together; a failure on either
   application transport terminates the process, and diagnostics closes last.
6. Protobuf format, lint, generate, drift, and breaking gates distinguish
   no-source, first-contract, compatible-change, breaking-change, and invalid
   base cases.
7. Generated Go uses Opaque-safe access and generic streams and compiles with
   the pinned module/tool versions.
8. Existing focused HTTP and bootstrap behavior remains green.
9. Documentation gives a complete first-production-RPC path and states the
   Railway private/public claim boundary without implying live deployment
   proof.
10. The repository's matching aggregate check passes at the scope required for
    publication.

## Risks, assumptions, and reopen conditions

- Assumption: the base template promises Go generated clients only. Reopen
  Protobuf edition and generator policy before promising another language.
- Assumption: a derived service may need both REST and gRPC in one process.
  Reopen process topology if a service requires independent scaling or
  deployment lifecycles.
- Assumption: finite symmetric message defaults are safer than gRPC's large
  effective send allowance. Reopen the numeric default only with representative
  payload and memory evidence.
- Assumption: no universal stream timeout is honest without a business SLA.
  Reopen per method when the owning feature defines duration or idle semantics.
- Railway public gRPC remains unproven. Reopen `DEP-1` only with current
  provider authority or an end-to-end unary and streaming probe through the
  exact intended ingress.
- Edition 2024 remains opt-in. Reopen the baseline when the template explicitly
  accepts all of its cross-language defaults and proves its declared client
  matrix.
- Reflection, keepalive, retry, gateway/transcoding, Protovalidate, BSR, and
  generated SDK publication remain future opt-ins; a concrete accepted
  requirement reopens only its owning decision.
