# Trusted service-to-service correlation

status: ready

## Scope and non-goals

### In scope

- Make the optional bounded HTTP client and optional native gRPC client carry
  correlation metadata according to an explicit outbound trust policy.
- Preserve one W3C trace across trusted HTTP and gRPC service hops and carry the
  already accepted `X-Request-ID` / `x-request-id` alongside it.
- Keep outbound calls locally observable even when the remote target is not
  allowed to receive correlation metadata.
- Supply executable two-service proof for correlation and cancellation rather
  than relying only on isolated middleware tests.
- Document the shortest supported path from a downstream service's
  authoritative OpenAPI or protobuf contract to its generated client and the
  shared transport.

### Non-goals

- Workload identity, authentication, authorization, token acquisition, mTLS
  policy, service discovery, endpoint failover, or a service mesh.
- A universal timeout, retry, circuit-breaker, rate-limit, idempotency, or
  downstream-error policy. Those remain per dependency or per operation.
- Resolver-supplied gRPC service configuration, load-balancing policy, health
  checking, or configured retry. The base client keeps those disabled because
  they can change retry and late-metadata behavior outside the selected
  propagation boundary.
- Environment-selected HTTP/HTTPS proxying for the base gRPC client. grpc-go's
  proxy delegating resolver can replace the selected target resolver after the
  package's enforcement boundary, so the base client connects directly.
- Propagating OpenTelemetry baggage or arbitrary application metadata.
- Defining an organization-wide request-ID format beyond the existing bounded
  template contract.
- Copying a downstream OpenAPI or protobuf source into this template, publishing
  generated SDKs, or adding a framework around generated clients.
- Changing inbound HTTP or gRPC request-ID acceptance, response correlation,
  logging, readiness, or shutdown behavior.

## Behavior and contract delta

### COR-1 — Explicit outbound propagation policy

Each reusable outbound HTTP client and gRPC client connection selects exactly
one policy:

| Policy | Remote W3C `traceparent` / `tracestate` | Remote request ID | Local client telemetry |
| --- | --- | --- | --- |
| none | stripped | stripped | retained |
| trace context | injected from the outbound call context | stripped | retained |
| trusted service | injected from the outbound call context | inject the valid request ID already present in that context | retained |

The zero/default policy is `none`. Network placement, plaintext/private
transport, TLS, or a hostname suffix does not by itself establish application
trust. A dependency owner must deliberately select a stronger policy.

Before every network attempt, the reusable client removes caller-supplied
`traceparent`, `tracestate`, `baggage`, and request-ID values, including HTTP
request headers and trailers. It then adds only the metadata permitted by the
selected policy. A retry repeats the same policy from the same call context; it
cannot preserve stale metadata from a previous attempt.

The policy is immutable for the lifetime of the HTTP client or gRPC connection.
Sanitization and injection operate on per-call/per-attempt request or metadata
copies and do not mutate the caller-owned HTTP headers, HTTP trailers, or
outgoing gRPC metadata.

An unknown policy fails client construction before network I/O.

The reusable gRPC client disables resolver-supplied service configuration.
grpc-go transparent retry remains native behavior, but a resolver cannot select
a configured retry or balancer that injects metadata after template policy
enforcement. A dependency that needs configured retry, client-side
load-balancing, or resolver health checking requires a separate dependency
design that owns replay safety, metadata trust, observability, and recovery; it
does not bypass the base client through a target scheme.

The reusable gRPC client also disables grpc-go's environment-selected proxy
delegation. Without that restriction, grpc-go may construct a second DNS
resolver behind the connection-local wrapper and reintroduce address metadata
after policy enforcement. A dependency that must traverse a proxy requires a
separate reviewed client whose proxy, resolver, authentication, metadata, and
trust behavior are explicit.

The client preserves resolver-owned target/address selection, address
attributes, and TLS server name, but clears deprecated
`resolver.Address.Metadata` before grpc-go can attach it to transport headers.
Resolver address metadata is not a supported authentication or propagation
channel. grpc-go still selects the target scheme and default scheme according
to its native rules. Every resolver builder that can win that selection for
the connection is wrapped, including its optional authority override. Failure
to resolve the effective builder fails client construction rather than
silently falling back to an unwrapped resolver.

### COR-2 — Request-ID authority

The request context is the only authority for automatic outbound request-ID
propagation.

- Under `trusted service`, a request ID is sent only when
  `reqctx.RequestID(ctx)` satisfies the existing `reqctx.ValidRequestID`
  contract.
- An absent or invalid context value causes the outbound request-ID field to be
  absent. The client does not silently invent a second identifier that the
  caller cannot also observe in its own logs.
- A caller-supplied HTTP header or gRPC outgoing metadata value never overrides
  the context value and is removed when the context has no valid value.
- The downstream service applies its existing inbound validation. For a valid
  propagated ID, caller and callee expose the same value in response metadata
  and context-aware logs.

### COR-3 — Trace authority and privacy boundary

W3C Trace Context is the only remote trace propagation format supplied by the
template. Under `trace context` or `trusted service`, the emitted parent is the
client span created for the concrete HTTP attempt or gRPC attempt, so the
downstream server span joins the same trace with the correct parentage.

Under `none`, the client still creates local spans and metrics, but the remote
request contains no template-managed trace context. Existing remote headers
cannot bypass that policy.

`baggage` is removed for all three policies and is not included in the global,
outbound HTTP, or outbound gRPC propagator. A future baggage field requires its
own allowlist, sensitivity, size, cardinality, and trust-boundary decision.

The current outbound `grpcclient.Options.Propagators` extension point is
removed. It cannot coexist honestly with a closed `none` / W3C-only / trusted
policy because an arbitrary propagator can inject fields outside that policy.
Existing derived repositories that set it receive a compile-time migration:
they must select the explicit propagation policy, and a non-W3C custom
propagator has no compatibility path in template core. The inbound gRPC server
propagator option and its current bootstrap selection remain unchanged.

### COR-4 — Context, cancellation, and bounded signals

Generated or handwritten HTTP and gRPC clients receive the operation context
from the feature adapter. The reusable transport does not replace that context,
detach from it, or create a universal per-operation deadline.

Cancellation and deadline expiry continue through the shared client to the
remote handler and release request-owned work. Retries, when explicitly enabled
for HTTP, remain inside the original call budget.

Request IDs, trace IDs, span IDs, remote method/route identities, and
dependency names may appear in logs and spans under their existing bounded
contracts. Request IDs, trace IDs, span IDs, raw URLs, headers, metadata, and
payload values do not become metric labels.

### CON-1 — Generated consumer ownership

For a service dependency:

- the provider's versioned OpenAPI or protobuf schema remains authoritative;
- generated client code is derived and is not edited by hand;
- the dependency adapter owns authentication, operation deadlines, retry and
  idempotency eligibility, provider error mapping, and conversion between
  generated transport types and feature-owned types;
- the generated HTTP client uses the bounded HTTP client's `Do` method, and a
  generated gRPC client uses the shared `grpcclient` connection;
- bootstrap owns construction, long-lived reuse, cleanup, and concrete
  propagation policy selection.

The template documents and proves this composition but does not generate a
client without a real downstream contract. Handwritten request/response models
that duplicate an available authoritative generated contract are not the
supported path.

## Invariants and edge cases

- Incoming trace context and request ID acceptance remain unchanged.
- The bounded HTTP client's fixed-authority, address, timeout, body, header,
  connection, redirect, and retry constraints apply on every attempt.
- The gRPC client's explicit transport credentials and message/header limits
  remain unchanged.
- Correlation policy does not confer caller identity or authorization.
- A private target may still select `none`; an HTTPS target may select
  `trusted service` when the dependency owner has established that trust.
- Empty contexts, background jobs, invalid request IDs, caller-supplied stale
  propagation headers, retries, unary RPCs, and streaming RPC creation all
  follow the same policy table.
- The default `none` policy is a deliberate compatibility change from the
  current clients' implicit W3C propagation. Existing derived repositories
  that require cross-service tracing must select `trace context` or
  `trusted service` explicitly during adoption.
- Existing derived repositories that initialize outbound
  `grpcclient.Options.Propagators` must replace that field with the explicit
  propagation policy. This intentional compile-time break prevents a custom
  propagator from silently bypassing the new disclosure boundary.
- Existing derived repositories that depend on resolver-supplied gRPC service
  configuration must adopt an explicitly designed dependency client instead of
  the base `grpcclient`. Resolver-selected configured retries, load balancing,
  and picker metadata no longer affect this client.
- Existing custom gRPC resolvers continue to own addresses, endpoint
  attributes, and server names, but their deprecated address metadata is
  removed. A dependency that used it for transport headers must move those
  values to explicit credentials or another reviewed dependency contract.
- Existing derived repositories that rely on `HTTP_PROXY`, `HTTPS_PROXY`, or
  `NO_PROXY` for this gRPC connection must use an explicitly designed
  dependency client. The base client deliberately ignores those variables.
- Services initialized without `OUTBOUND_HTTP=bounded` or without
  `GRPC=enabled` retain no client surface for that unselected capability.

## Decisions, constraints, and authorities

- W3C Trace Context owns trace header representation; the existing
  `reqctx.ValidRequestID` contract owns request-ID validity.
- OpenTelemetry instrumentation continues to own HTTP and gRPC client spans and
  protocol metrics. Repository code owns only the trust decision, header
  sanitization, request-ID policy, and composition.
- This contract supersedes the earlier gRPC capability statement that a
  resolver may supply retry policy. The base client now disables resolver
  service configuration; grpc-go transparent retry remains unchanged.
- The provider repository or its published versioned schema owns generated
  client contracts. A local adapter owns service-specific policy.
- No new runtime dependency is justified: the implementation uses the existing
  OpenTelemetry, gRPC-Go, generated-client, and standard-library surfaces.
- The work must not consume or modify the concurrent Goose migration changes
  in the primary checkout.

## Success criteria and proof expectations

- Focused HTTP proof falsifies each policy using pre-populated stale
  `traceparent`, `tracestate`, `baggage`, and `X-Request-ID` headers, including
  retry attempts and invalid/missing context request IDs.
- Focused gRPC proof falsifies each policy for unary and streaming calls using
  pre-populated outgoing metadata.
- A deterministic gRPC transparent-retry proof observes every remote attempt
  and falsifies stale metadata leakage or wrong attempt-span parentage under
  each applicable policy; a single successful logical RPC is not sufficient
  evidence. A separate construction/behavior proof establishes that
  resolver-supplied configured retry and load-balancing policy are ignored and
  resolver address metadata is absent while native explicit/default scheme
  selection, authority override, address/server-name/attribute resolution, and
  direct connection despite proxy environment variables remain intact.
- A two-service HTTP proof observes one trace and one accepted request ID in
  both service contexts/logs, then proves cancellation reaches downstream
  request-owned work.
- A live client/server gRPC proof observes one trace and one accepted request
  ID across unary and streaming boundaries.
- Existing inbound correlation, HTTP client safety/retry, gRPC limits and
  telemetry, profile-removal, generated-drift, and aggregate repository checks
  remain green at the proof level triggered by the final diff.
- Documentation gives one concrete generated HTTP consumer composition and one
  generated gRPC consumer composition without claiming provider policy is
  generic.

## Risks, assumptions, and reopen conditions

- The default becomes deliberately non-propagating. Reopen compatibility only
  if a currently supported derived-repository contract requires implicit W3C
  propagation; the safer fallback is an explicit compatibility mode during
  adoption, not implicit trust by hostname.
- Resolver service configuration is deliberately disabled to close a
  post-policy metadata/retry owner. Reopen gRPC Client Design only for a real
  dependency with a current resolver/load-balancing requirement and an
  enforceable late-metadata boundary.
- Resolver address metadata is deliberately removed because grpc-go appends it
  after propagation. Reopen Security only if a real dependency proves a
  required address-metadata contract that can exclude reserved correlation
  fields at the final wire boundary.
- grpc-go proxy delegation is deliberately disabled because it inserts a
  resolver behind the enforceable wrapper. Reopen System / Integration Design
  only for a real dependency that requires a proxy and can enforce the same
  final metadata and trust boundary.
- The proof uses representative in-process services. gRPC uses loopback; the
  HTTP proof uses a test-owned private-address listener and in-process DNS so
  it exercises the production `PrivateHTTP` dial guard instead of weakening
  that guard for loopback. A deployed platform, proxy, or mesh may still
  rewrite propagation fields; reopen Delivery when production topology
  requires that claim.
- Request ID remains a diagnostic handle, not a security principal. Reopen
  Security if a service attempts to authorize, rate-limit, or deduplicate work
  from it.
- Generated-client documentation assumes the provider exposes an authoritative
  versioned schema. If it does not, the dependency owner must establish contract
  authority before generation; the template must not guess it.
