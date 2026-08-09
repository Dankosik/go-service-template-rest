# gRPC clients use safe idle behavior, health-aware routing, and transport-neutral failures

status: ready

Problem: the shared gRPC client actively pings every resolved backend while
idle, does not consume the standard health state the server already publishes,
and derives gRPC status semantics from an HTTP-owned problem catalog. The first
behavior is unsafe as a generic dependency default, the second can route new
RPCs to a draining direct-discovery backend, and the third renders an existing
resource collision as the transaction-retry status `ABORTED`.

This specification supersedes these parts of
`specs/grpc-transport-hardening/spec.md`:

- the error-vocabulary non-goal, its reopen condition, and the proof that every
  HTTP problem-catalog code must have a gRPC projection;
- the client health-checking non-goal;
- R6's default-client parity invariant, its R8-dependent rationale and
  falsifier, and the corresponding part of success criterion 6 and the risk
  statement. R6's server-side rule remains in force for explicitly enabled
  clients;
- R8's mandatory idle-keepalive behavior.

That specification's remaining server, load-balancing, lifecycle, telemetry,
and transport-bound decisions stay in force.

Research is scoped down: the preceding read-only audit already compared the
current tree with the current official gRPC keepalive, health-checking, status,
grpc-go client-construction, generated-streaming, and Protobuf Opaque-API
contracts. No unresolved evidence question can change the behavior below.

## Scope and non-goals

In scope:

- the default and explicitly enabled outbound keepalive behavior;
- standard client-side health checking for the existing round-robin policy;
- the domain-error classification vocabulary consumed by HTTP and gRPC;
- compatibility of existing non-conflict error identities and both transports'
  existing sanitization boundaries;
- operator documentation and executable boundary proof.

Non-goals:

- application retries, `WaitForReady`, per-call deadlines, resolver-supplied
  service configuration, proxies, reflection, or xDS;
- service-specific health states: the repository continues to publish and
  consume the empty service name as whole-process health;
- changing server readiness, admission, drain order, keepalive, connection
  bounds, or forced-shutdown behavior;
- changing token validation or business authorization policy; under the OIDC
  profile `Check` remains the only public health method, while `Watch` follows
  the same authenticated stream boundary and process admission budget as other
  protected streams;
- inventing a generic error-code registration framework. The closed repository
  vocabulary remains ordinary typed constants and explicit transport mappings.

## Behavior and contract delta

### R1 — Idle client keepalive is opt-in

**Trigger.** An outbound connection is built from the shared client's default
configuration.

**Rule.** The connection sends no HTTP/2 keepalive ping while it has no active
RPC. A dependency owner enables idle keepalive only by supplying both a positive
ping interval and a positive ping timeout. Supplying one without the other, or
supplying a negative value, is rejected during construction.

**Enabled behavior.** An explicitly enabled connection may ping without an
active stream; that is the behavior the opt-in names. Its configured interval
and timeout become grpc-go's initial client parameters. grpc-go retains
authority over its documented floors and over adaptive effective intervals,
including increasing the interval for later connections after a peer rejects
excessive pings; this layer neither mutates the configured values nor overrides
that recovery. The dependency owner is responsible for choosing values accepted
by the named server and any intermediary. A server rejection remains observable
through standard grpc-go connection behavior; this layer adds no application
retry.

**Compatibility.** Per-call deadlines, transparent retries before commit,
reconnection, connection sharing, and the server's own liveness policy are
unchanged. Removing the active default may let an intermediary discard a truly
idle connection; the next RPC then follows grpc-go's normal reconnect behavior
inside that RPC's deadline.

**Falsifier.** With health checking disabled to exclude its persistent `Watch`
stream, a connection using the default keepalive fields emits no ping past the
interval at which the previous 30-second policy did. The same peer observes a
ping from a deliberately enabled connection. Partial keepalive configuration
fails before any connection attempt. grpc-go's own focused proof remains the
authority for adaptive interval behavior after `too_many_pings`; this repository
does not duplicate that library state machine.

### R2 — Round-robin routing consumes standard health

**Trigger.** The shared client uses round-robin address selection with health
checking enabled; the default client configuration selects both.

**Rule.** The client watches the standard gRPC health service for the empty
service name on each resolved backend. A backend whose `Watch` method is
supported becomes eligible for new RPCs only while it reports `SERVING`;
`NOT_SERVING` removes it from selection without canceling RPCs already in
flight. A later `SERVING` response makes it eligible again.

**Unsupported peer.** If the peer returns `UNIMPLEMENTED` for standard health,
grpc-go disables subchannel health checking for that peer and continues using
connectivity state. This is the protocol's compatibility behavior, not a local
failure or retry policy. A peer that implements standard health but does not
publish the empty service name is different: it remains unhealthy under this
contract, so its dependency owner must explicitly disable health checking. A
dependency-specific health service name reopens this specification rather than
being guessed from the RPC package.

**Trust boundary.** Resolver-supplied service configuration remains disabled.
The load-balancing and health-checking policy is the client's own fixed service
configuration, so a resolver still cannot add retries, replace address
selection, or weaken metadata sanitization. A dependency may supply one
connection-scoped per-RPC credential source so grpc-go's internal health
`Watch` and ordinary application RPCs authenticate through the same connection.
Credential-supplied correlation metadata remains untrusted and is removed by
the same reserved-key boundary as call-scoped credentials. Credential errors
and transport-security requirements retain grpc-go's existing behavior.

**Authenticated profile.** With OIDC/JWT enabled, `Check` remains a public
probe, while `Watch` requires a valid bearer token, consumes the process-wide
RPC admission budget, and cannot outlive the verified token's `exp + ClockSkew`.
An enabled round-robin client of that profile supplies a connection-scoped
credential; without one, the protected health stream fails closed and the
backend does not become health-eligible. Disabling health remains valid only
for a dependency that does not publish the accepted empty-service health
contract, not as an authentication bypass.

**Drain consequence.** The server continues to publish `NOT_SERVING` before the
readiness-propagation delay. A default template client watching that state stops
selecting the draining backend during that delay, while orchestrator endpoint
removal remains a second compatible signal.

**Falsifier.** With two resolved backends, calls reach both while both report
`SERVING`; after one reports `NOT_SERVING`, new calls reach only the other; after
it reports `SERVING` again, calls reach both. A peer without the health service
remains callable. Over real TLS with the OIDC/JWT server boundary installed,
grpc-go's automatic empty-service `Watch` reaches the protected handler with a
verified principal from the connection credential, reserved correlation values
returned by that credential are absent, and an application RPC on the same
connection succeeds without call-scoped authentication metadata.

### R3 — Domain failure identity is transport-neutral

**Trigger.** A handler returns an error classified by a registered domain
mapper.

**Rule.** The mapper produces a stable service failure code, safe detail, and
optional retry delay without selecting an HTTP status, RFC problem definition,
or gRPC status. HTTP and gRPC project that one classification independently.
Unknown errors remain sanitized, and cancellation or deadline expiry retains
precedence over classification.

**Shared identity.** HTTP publishes the lower-snake-case failure code in its
problem body. gRPC publishes the same identity upper-snake-cased in
`google.rpc.ErrorInfo.Reason`. Existing non-conflict codes and their current
HTTP and gRPC outcomes remain unchanged.

**Conflict semantics.** The generic domain code `conflict` is replaced for the
current producer by the smallest caller-actionable identity:

| Failure code | HTTP | gRPC | Caller meaning |
| --- | --- | --- | --- |
| `already_exists` | `409 Conflict` | `ALREADY_EXISTS` | The create target already exists; repeating the same create is not a concurrency recovery. |

The reference article slug collision is `already_exists`. A future producer
that requires a state change or a transaction retry adds its exact standard
identity then; this specification does not publish unused codes for it.

**HTTP compatibility.** Existing HTTP-only fallback responses may continue to
use the HTTP problem code `conflict`. Domain mappers do not. The reference
example's client-visible code changes from `conflict` to `already_exists`; this
is an accepted contract correction in a non-production example with no shipped
consumer. Status, title, type URI, and safe detail remain compatible.

**gRPC compatibility.** Existing classified statuses other than the ambiguous
conflict projection are unchanged. `RetryInfo`, sanitization, policy-interceptor
status trust, standard health status pass-through, and handler-owned-status
rejection remain unchanged.

**Falsifier.** The same slug-collision domain error reaches HTTP as status 409
with code `already_exists` and reaches a production-composed gRPC caller as
`ALREADY_EXISTS` with `ErrorInfo.Reason=ALREADY_EXISTS`. An unclassified internal
error exposes neither its text nor a machine-readable failure identity.

## Invariants and edge cases

- A default client still creates no network connection during construction and
  one `ClientConn` remains shareable per dependency.
- Health checking and keepalive are independent: health says whether a service
  should receive work; keepalive says whether a connection is alive.
- Health `Check` remains public and outside business admission. `Watch` uses the
  protected stream's authentication, process admission, and token-expiry bound;
  it remains excluded from routine server access logs and default server
  protocol telemetry.
- No health, keepalive, or error policy adds an application retry.
- No resolver, credential, or caller metadata regains authority over reserved
  correlation keys.
- Generated protobuf authority, all four stream cardinalities, bounded stream
  aggregation, TLS/plaintext decisions, and public-ingress limitations are
  unchanged.

## Decisions, constraints, and authorities

- Default idle keepalive is disabled because the official gRPC keepalive guide
  recommends avoiding keepalive without calls and intervals materially below one
  minute unless coordinated with the service owner. Rejected: changing 30s to a
  different universal number, because no number is safe for every dependency.
- Standard health is enabled for the existing round-robin default because both
  halves already implement the protocol and `NOT_SERVING` otherwise has no
  direct-client consumer. Rejected: a custom probe or polling loop, because the
  standard streaming protocol already carries transitions and grpc-go's
  balancer already owns eligibility.
- OIDC/JWT protects `Watch` because it is long-lived caller-driven work, while
  `Check` stays public as the bounded platform probe. A connection-scoped
  credential is selected over publishing `Watch` or disabling health because
  grpc-go already applies that native credential seam to internal control
  streams and application RPCs.
- Domain classification becomes transport-neutral because the current resource
  collision means `ALREADY_EXISTS`, not the transaction-retry status selected by
  an HTTP-to-gRPC lookup table. Rejected: preserving generic `conflict` plus
  inspecting error text or handler type in the gRPC adapter.
- `grpc.NewClient`, standard health, grpc-go generated interfaces, Buf v2,
  Edition 2023 with schema-owned `API_OPAQUE`, and the existing separate listener
  remain protocol and repository authorities.

## Success criteria and proof expectations

1. A default client emits no idle keepalive; explicit complete configuration
   enables it; invalid partial configuration is refused before I/O.
2. Default round-robin routing follows `SERVING → NOT_SERVING → SERVING` per
   backend, degrades compatibly when health is unimplemented, and composes over
   TLS with the protected OIDC/JWT `Watch` without leaking credential-supplied
   reserved correlation metadata.
3. Explicitly health-disabled round-robin and `pick_first` clients remain
   callable.
4. Domain classification can be consumed without importing the HTTP problem
   catalog or `net/http`.
5. The slug collision produces the exact HTTP and gRPC outcomes in R3 on
   composed transport paths.
6. Existing non-conflict status, metadata, deadline, streaming, admission,
   health, shutdown, TLS, telemetry, and protobuf checks remain green.
7. Client keepalive/health and server lifecycle concurrency proof passes under
   the race detector.

No throughput, latency, public-ingress, or production-capacity claim is part of
this outcome.

## Risks, assumptions, and reopen conditions

- A deployment that needs idle keepalive must now opt in. Reopen R1 only with a
  named dependency and measured or documented intermediary timeout; that owner
  supplies a compatible interval and timeout rather than changing the global
  default.
- Default round-robin health adds one standard health watch per participating
  backend. Reopen R2 if a measured fleet shows material watch cost or a concrete
  dependency requires a different service name or an authentication mechanism
  that cannot supply grpc-go `PerRPCCredentials`; preserve health-aware drain or
  explicitly select orchestrator-only routing for that dependency.
- The new conflict code changes one reference-example machine identity. Reopen
  R3 only for a real compatibility owner with a shipped consumer; retain the
  distinct caller action even if a migration alias is required.
- A new failure category reopens the closed classification vocabulary only when
  a real producer cannot state its caller action using the existing codes.

## Review disposition

The shared review trigger applies because this specification changes
client-visible failure identity and connection-lifecycle policy.

1. Independent whole-artifact review — `FAIL`: grpc-go's adaptive keepalive
   interval was assigned to the adapter, the keepalive falsifier did not isolate
   the health `Watch` stream, and two failure identities had no producer.
2. Root repair and evidence check — the contract now leaves runtime adaptation
   with grpc-go, disables health in the keepalive falsifier, retains only
   `already_exists`, and explicitly supersedes the old R6/R8 proof clauses.
3. Independent focused re-review of those repairs — `PASS`; no material finding
   remained at that revision.
4. Later grpc-go v1.82.1 evidence reopened only the claim that `pick_first`
   ignores subchannel health and the resulting construction rejection.
5. Root repair — R2, its falsifier, and success criterion 3 no longer reject
   health-enabled `pick_first`; explicit health disabling remains supported.
6. Independent focused re-review of the repaired boundary — `PASS`; no finding
   remains, and the earlier review dispositions outside that boundary still
   apply.
7. Implementation evidence reopened R2: the hardened OIDC/JWT profile protects
   `Health/Watch`, while grpc-go's automatic health stream can consume only a
   connection-scoped credential and the shared client exposed no such seam.
8. Root repair keeps `Check` public, makes protected `Watch` explicit, and
   requires one sanitized connection credential plus composed TLS/OIDC proof.
   Earlier R1 and R3 dispositions remain unchanged.
9. Independent focused review of the fixed repaired R2 boundary — `PASS`; its
   eligibility, fail-closed credential behavior, protected server policy, and
   composed proof leave no Specification-owned divergence.

The root re-checked each accepted finding against grpc-go v1.82.1 and the
superseded specification before applying it.
