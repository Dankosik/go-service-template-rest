# gRPC callers receive the same retry, identity, and liveness guarantees the HTTP transport already gives them

status: ready
Revision: repaired after the Go Ownership review panel reopened R1, R2, R4, and
R5. The panel's evidence is recorded with each repaired rule; every rule it did
not reopen is unchanged.

Problem: the native gRPC transport drops data the shared domain-error contract
promises, leaves handler occupancy and connection lifetime unbounded, and pins
every caller of a multi-backend dependency to one address. Each gap is invisible
from the gRPC side alone: the HTTP transport already answers all four, so the
divergence reads as a transport detail rather than a missing guarantee.

## Scope and non-goals

In scope: the server transport adapter's error rendering, RPC occupancy bound,
and connection lifetime; the client adapter's address selection and connection
liveness; the configuration that carries them; the service composition root that
supplies service identity; and `docs/grpc.md` as the operator-facing contract.

Non-goals, each with the disposition that keeps it out:

- **gRPC error vocabulary.** `problem.Code` stays the single classification
  vocabulary for both transports. `CodeConflict` continues to render `ABORTED`,
  which is `google.rpc.Code`'s own mapping for HTTP 409. `ALREADY_EXISTS`,
  `FAILED_PRECONDITION`, `OUT_OF_RANGE`, and `DATA_LOSS` remain unreachable, and
  `docs/grpc.md` already records the reopen condition — a service that needs one
  extends `problem` and the gRPC rendering together. No producer in this
  repository needs one.
- **Field-level error details.** `google.rpc.BadRequest` and its
  `FieldViolation` set are not attached. Nothing in this repository produces
  field-level violations; a mapper that did would extend the same rendering seam
  this spec establishes.
- **Proto message validation.** No validation policy is added. The existing
  supplied-policy seam already accepts one, and a constraint-free service would
  gain a policy that decides nothing.
- **Pre-decode admission shedding.** Admission continues to run after the unary
  request message is decoded. `grpc.InTapHandle` is experimental in grpc-go, and
  decode cost is already bounded by the receive-message limit.
- **Client-side health-checking load balancing.** Address removal stays with the
  orchestrator's readiness probe. Enabling it would open a persistent watch
  stream per backend for a health signal that is process-wide rather than
  per-service.
- **Server reflection.** Unchanged and still absent, on the disposition
  `docs/grpc.md` already records.

Deliberately unchanged: interceptor order and its two error boundaries; the
sanitize-by-default rule for an unowned handler error; correlation acceptance
and minting; the admission budget and its health exemption; every existing
transport bound; per-call retry and `WaitForReady`, which remain per-method
business decisions the client adapter does not make.

## Behavior and contract delta

### R1 — A classified retry hint reaches a gRPC caller

**Trigger.** A handler returns an error that a supplied domain mapper
classifies, and the resulting `problem.Mapped` carries a positive retry delay.

**Rule.** The status returned to the caller carries `google.rpc.RetryInfo` with
that delay exactly. The gRPC code and detail are unchanged by its presence.

**Cross-transport invariant.** Neither transport advertises a delay shorter than
the mapper's. HTTP's `Retry-After` is whole seconds, so it rounds up with a floor
of one second; gRPC's `RetryInfo` carries a `Duration` and is exact. This is the
invariant a client needs, and it is the strongest one both transports can hold:
integer seconds is `Retry-After`'s own granularity, not a defect to mirror. The
earlier wording — "the same duration" — was unachievable for any sub-second or
fractional delay and is replaced rather than weakened.

**Precedence and absence.** A non-positive delay attaches nothing. A cancelled
or expired RPC context, an unclassified error, and a sanitized handler status all
answer before classification and therefore carry no retry hint.

**Why this is a delta.** `problem.Mapped.RetryAfter` is already honored by the
HTTP transport as a `Retry-After` header. The same mapper value currently
reaches a gRPC caller as nothing at all, so one domain error tells an HTTP client
when to come back and tells a gRPC client to guess.

**Falsifier.** For a mapper delay of 200ms, 1s, and 1500ms, the gRPC `RetryInfo`
carries exactly that duration and the HTTP `Retry-After` carries 1, 1, and 2
seconds — each at least the mapper's delay. Removing the gRPC rendering fails the
gRPC half.

### R2 — A gRPC caller can identify the error class without parsing prose

**Trigger.** Any classified handler error, whether or not it carries a retry
delay.

**Rule.** The status carries `google.rpc.ErrorInfo` whose `Reason` is the
`problem.Code` value rendered in upper snake case, and whose `Domain` is the
service's own identity. `CodeNotFound` reaches a caller as `NOT_FOUND`.

**Why the rendering.** `google.rpc.ErrorInfo` documents `Reason` as at most 63
characters matching `[A-Z][A-Z0-9_]+[A-Z0-9]`. Every catalog code is lower snake
case ASCII and the longest is 31 characters, so uppercasing is total and
injective — distinct codes stay distinct. `problem.Code` remains the only stored
identity; this is a transport rendering, not a second spelling anyone maintains.

**Invariant.** Every code in the catalog renders to a conforming reason. A code
added later that does not is a defect in that code, caught over the whole
catalog rather than per call site.

**Absence and default.** When the composition supplies no service identity, the
`ErrorInfo` detail is omitted; every other part of the status is unchanged, and
`RetryInfo` is unaffected. The service composition root always supplies one, so
the production path always carries it.

**Why this is a delta.** The gRPC code space is coarser than `problem.Code`:
`InvalidArgument` is the answer for both `CodeBadRequest` and
`CodeUnprocessableContent`, and `ResourceExhausted` is the answer for
`CodeRequestEntityTooLarge`, `CodeTooManyRequests`, and — under the
authn profile — `CodeRequestHeaderFieldsTooLarge`. An HTTP client recovers the
distinction from the problem document's type URI. A gRPC client currently
cannot recover it at all.

**Side effect bound.** `Reason` and `Domain` are repository-owned values. No
handler error text, dependency name, or peer-supplied value becomes a detail.

**Falsifier.** Two domain errors mapping to distinct `problem.Code` values that
share one gRPC code arrive at the caller with distinct `ErrorInfo.Reason`
values. An unclassified error carries no `ErrorInfo` and no handler text.

### R3 — A unary RPC cannot occupy a handler indefinitely

**Trigger.** Any non-health unary RPC, with or without a caller deadline.

**Rule.** The RPC context carries a deadline no later than the configured unary
budget. A caller deadline earlier than the budget still wins; the budget is a
cap, never an extension.

**Observable outcome.** A handler that observes cancellation returns, and the
caller receives `DEADLINE_EXCEEDED`. Its admission slot is released when the
handler returns.

**Accepted limitation.** Cancellation is the protection, not the response. A
handler that ignores its context keeps its goroutine and its admission slot;
nothing in the transport can stop it. This is the identical limitation the HTTP
transport records for `RequestTimeout`, and it is accepted for the same reason:
running the handler on another goroutine returns early while leaking that
goroutine anyway.

**Default and bound.** The budget is its own gRPC configuration key with a
default of 8s — the value `http.request_timeout` already defaults to — so one
service answers on one budget over both transports out of the box while a
deployment can still move the two apart. It is a separate key rather than a read
of the HTTP value because the two transports have independent listeners and a
gRPC-only budget must be reachable without changing HTTP behavior. A non-positive
value disables the cap.

**Why this is a delta.** Admission sheds rather than queues. Combined with an
unbounded handler, a caller that opens RPCs without deadlines against a hung
dependency permanently consumes the process-wide budget: the server stays
`SERVING` and answers `RESOURCE_EXHAUSTED` to everything else. The HTTP
transport's own configuration already states this reasoning for its handler
budget; the gRPC transport has no counterpart.

**Falsifier.** A handler that blocks on its context, invoked without a caller
deadline, returns `DEADLINE_EXCEEDED` within the budget, and a second RPC
afterwards is admitted rather than shed.

### R4 — A streaming RPC is bounded by its own budget and, when enabled, by connection rotation

**Trigger.** Any non-health streaming RPC.

**Rule.** Exactly two *configured lifetime* bounds can end a stream from this
transport, and whichever expires first wins:

- its own configured budget, non-positive and therefore disabled by default;
- the connection's remaining lifetime, when R5's rotation bound is enabled.

Shutdown is a third terminator and is deliberately unchanged: the drain budget
still ends every remaining stream when it expires.

With the shipped defaults both are off, so no configured lifetime bound ends a
stream: it ends when its handler ends, when its caller goes away, or when
shutdown's drain budget expires.

**What cannot end it.** Neither the maximum-idle bound nor the keepalive ping
bound can cut a stream that is doing work. The idle clock runs only while no RPC
is outstanding, and the ping bound closes only when a ping goes unanswered, which
means the peer is gone. Both being on by default therefore costs a stream
nothing — which is what makes R5's split defaults safe.

**Relation that must hold.** When both the stream budget and the configured
maximum age are positive, the stream budget must be strictly smaller than that
configured age. The compared quantity is the configured value, not the effective
cut point. The drain lands within ±10% of the configured age and the force-close
follows it by exactly the grace, so the cut falls after the configured age on
every draw only when the grace exceeds a tenth of it. It varies per connection in
every case.

**What that relation does and does not buy.** It is a deterministic sanity bound
on configured values, and nothing stronger. It is neither necessary nor
sufficient for the stream budget to decide an outcome:

- not necessary — the effective cut is the jittered age *plus* the grace, so a
  budget above the configured age can still fire first. With age 30s, grace 10s,
  the cut lands between 37s and 43s, and a 35s budget decides on every jitter
  draw. The relation refuses that configuration anyway.
- not sufficient — jitter reaches 10% below the configured age, so with a large
  age and a small grace a budget just under the age can still lose. At age 1h and
  grace 10s the cut can land at 54m10s, ahead of a 59m budget.

What it does buy is a cheap, jitter-independent refusal of the configuration
that is most clearly pointless, decided from configured values alone.

The deeper reason no relation can promise more is that the two clocks have
different origins: the stream budget starts when the stream starts, rotation
started when the connection was accepted. A stream opened late in a connection's
life is cut by rotation regardless of any configured relation. Enabling rotation
therefore means accepting that some streams end with the transport's
`UNAVAILABLE` rather than the budget's `DEADLINE_EXCEEDED`.

**What this replaces.** The earlier wording said streams "behave exactly as they
do today unless a deployment sets it". That was false: grpc-go's maximum-age
timer sends GOAWAY and then force-closes the transport and every stream on it
once the grace period expires. Rotation is a stream bound whether or not it was
intended as one, so this contract names it rather than denying it. The
disposition `docs/grpc.md` already records — that a long-lived stream's duration
policy belongs to the feature owning the stream — survives, because rotation is
off unless a deployment turns it on.

**Falsifier.** With the stream budget and rotation off, a stream outliving the
unary budget completes normally. Under shortened idle and ping bounds — proving
the mechanism, since the shipped values are too long to exercise — the idle bound
does not close the connection while that stream is outstanding, and a server
keepalive ping issued during that stream is answered rather than closing it. With a positive stream budget and rotation
off, the stream is cancelled at that budget.
Configuration whose stream budget is at or above a configured maximum age is
refused at startup.

### R5 — A dead or idle peer stops holding capacity, and rotation is available

**Trigger.** Any accepted gRPC connection.

This rule has two parts because they differ in exactly one way that matters:
whether they can end work in progress.

**Liveness bounds — on by default, cannot cut live work.** A connection is closed
after a maximum idle period, and after a keepalive ping goes unanswered within
its timeout. Both clocks re-arm on activity, so only a connection that is
carrying nothing, or whose peer is gone, is closed.

*Defaults:* maximum idle 15m; server ping interval 1m; ping timeout 20s, which is
grpc-go's own default and therefore not a new number.

*Observable outcome:* a connection whose peer disappeared without a TCP close
releases its listener slot within roughly the ping interval plus its timeout,
rather than at grpc-go's 2h default.

**Rotation bound — off by default, ends everything on the connection.** When a
maximum age is configured, the connection is drained with GOAWAY at that age and
force-closed after an additive grace period, cutting every RPC still running.
grpc-go applies ±10% jitter to the age to spread connection storms.

*Defaults:* maximum age is unset, which grpc-go reads as infinity; age grace 10s,
used only when an age is set.

*Observable outcome when enabled:* a caller holding a rotated connection
reconnects and is re-resolved, so a new replica receives traffic without a client
restart.

*When to enable it:* behind an L4 load balancer or any hop that pins a caller to
one replica for the life of a connection. It is off by default because it is the
only bound here that ends work a caller cares about, and a template default that
silently cuts every RPC older than its age is a worse surprise than unbalanced
connections, which are visible in metrics.

**Why this is a delta.** grpc-go's defaults are infinite maximum idle, infinite
maximum age, and a 2h ping interval. The listener bounds accepted connections, so
without the liveness bounds a dead peer holds a slot for hours.

**Relations that must hold.** Every keepalive duration is non-negative. A
negative age is not a third meaning: grpc-go normalizes only zero to infinity, so
a negative one arms the timer immediately and rotates every connection at once —
the opposite of the off-by-default this rule chose. Zero disables rotation;
anything below it is refused.

When a maximum age is set, its grace must be **positive** and at least the gRPC
unary budget.

Positive, because grpc-go replaces a zero grace with infinity: the connection
would drain with GOAWAY and then never be force-closed, which contradicts this
rule's own text, makes its falsifier unreachable, and silently removes R4's
second stream bound while R4 still counts it. Zero is a disable switch for the
age, not for the grace.

At least the unary budget, mirroring the arithmetic `http.request_timeout`
already has against `http.write_timeout`: the budget must expire while the
connection can still carry the answer that reports it. When the unary budget is
disabled this term is vacuous and only positivity applies.

R4 owns the stream-budget relation.

**Falsifier.** With the shipped defaults, no configured lifetime bound closes a
connection carrying a long-running RPC, and a connection whose peer vanished is
closed within the ping bound. With a configured age, a connection held past it
receives GOAWAY and the client reconnects. Configuration that sets an age with a
non-positive grace, or with a grace below the unary budget, is refused at
startup.

### R6 — The server admits the keepalive behavior its own client half uses

**Trigger.** A client sends keepalive pings.

**Rule.** The server's minimum accepted client ping interval is configured, and
pings with no active stream are permitted.

**Invariant.** The client adapter's default ping interval is strictly greater
than the server adapter's default minimum accepted interval.

**Why this is a delta.** grpc-go's default enforcement rejects pings more
frequent than every 5 minutes and rejects any ping with no active stream, both
with `GOAWAY`. R8 makes this repository's own client ping on an idle connection,
so shipping both halves without agreeing on this is shipping a client this
server disconnects.

**Falsifier.** A parity check over the two adapters' defaults fails when the
client interval stops exceeding the server minimum.

### R7 — A client distributes RPCs across every resolved backend

**Trigger.** A client connection whose target resolves to more than one address.

**Rule.** The connection selects addresses by a configured load-balancing
policy. The default is round robin. The alternative is first-address selection.
An unrecognized value is refused at construction.

**Precedence, unchanged.** Resolver-supplied service config remains rejected,
proxies remain rejected, and the resolver metadata strip seam is unchanged. The
policy is supplied as this client's own default service config, which grpc-go
applies without reopening the resolver route.

**Why this is a delta.** With no default service config, grpc-go selects first
address. Against a headless DNS target, every RPC from every replica goes to one
backend. The current documentation describes this as adding no hidden balancer
policy, but first-address selection is itself a distribution decision.

**Cost at one address.** Round robin over a single address is equivalent to
first-address selection, so a mesh or sidecar target pays nothing for the safer
default.

**Falsifier.** A client against a target resolving to two servers reaches both.
Setting first-address selection reaches one.

### R8 — A client connection survives an idle intermediary

**Trigger.** A shared client connection with no active RPC.

**Rule.** The client sends keepalive pings on a configured interval, including
when no stream is active, and closes the connection when a ping goes unanswered
within its timeout.

**Defaults.** Ping interval 30s, ping timeout 10s. The interval satisfies R6's
invariant against the server's 10s minimum and exceeds grpc-go's own 10s floor.

**Why this is a delta.** The documented shape is one long-lived connection per
dependency built at startup. Without pings, a NAT or load balancer idle timeout
silently discards it, and the failure surfaces as the next RPC's error rather
than as a reconnect.

**Falsifier.** An idle connection remains usable past an interval at which the
peer-side idle bound would otherwise have discarded it.

### R9 — A missing service registration is refused, not skipped

**Trigger.** Server construction receives a nil entry among its service
registrations.

**Rule.** Construction fails. It does not skip the entry.

**Why this is a delta.** The entry is currently skipped, producing a server that
starts and serves without a method its composition meant to publish — the exact
failure the adapter's own documented stance on programming errors rejects
elsewhere.

**Falsifier.** Construction with a nil registration returns an error, and no
listener is opened.

## Invariants and edge cases

- Health RPCs stay exempt from the unary and stream budgets, as they already are
  from admission, routine access logs, and protocol telemetry. A probe must not
  consume or be cut by a business budget.
- Error details never carry handler error text, dependency identity, or any
  peer-supplied value. R1 and R2 add repository-owned values only.
- A status that carries details is still the status the existing boundaries
  decided. Details are attached where a classified `problem.Mapped` is rendered,
  which is unreachable for a cancelled RPC, an expired RPC, an unclassified
  error, and a sanitized handler status.
- Configuration accepted by the runtime config owner must remain constructible
  by the transport adapter, and the adapter's own bounds must remain a superset
  of what that owner accepts. New bounds join the existing parity obligation.
- Every crossing from runtime configuration into the transport adapter's bounds
  must carry the new values. **Four** such crossings exist and none can reach the
  others: the service composition root, the adapter's own configuration oracle,
  the benchmark server under `examples/`, and the reference service's own test
  harness at `examples/grpc-reference-service/service_test.go`. Three in-repo
  comments still claim three; correcting them is part of this outcome.
- The reference benchmark's measured path must stay comparable: its connection
  and transport bounds are the production ones, so any new bound it does not
  carry silently changes what the benchmark measures.

## Decisions, constraints, and authorities

| Decision | Chosen | Rejected, and why |
| --- | --- | --- |
| Retry hint representation | `google.rpc.RetryInfo` | A bespoke metadata key: the standard detail already exists and clients already read it. |
| Error identity representation | `google.rpc.ErrorInfo` with `problem.Code` upper-snake-cased as `Reason` | The problem type URI as `Reason`: it names an RFC 9110 section, not this service's error class. The code verbatim: it violates `ErrorInfo`'s documented shape, giving up the interoperability that was the reason to choose the standard detail. |
| Retry parity claim | Neither transport advertises a delay shorter than the mapper's | "The same duration": unachievable, because `Retry-After` is whole seconds by its own definition while `RetryInfo` is exact. Rounding gRPC up to match: discards precision to imitate a coarser transport. |
| Rotation default | Off | On at 30m: it is the only bound here that ends work in progress, and a template that silently cuts every RPC and stream older than its age trades a visible metric problem for an invisible correctness surprise. |
| Error domain source | The service identity already carried by `observability.otel.service_name` | A new service-name key: a second name for one identity. Sharing it also makes an `ErrorInfo` domain and a trace's service attribute agree. |
| Unary budget value | Its own gRPC key defaulting to 8s, matching `http.request_timeout`'s default | An independent default number: one service answering on two budgets out of the box is an operator trap. Reading `http.request_timeout` directly: it makes a gRPC-only budget unreachable and crosses two configuration sections in one validation rule. |
| Stream budget default | Disabled | A shared budget with unary: cuts legitimate streams or fails to protect unary traffic. |
| Budget mechanism | One policy, configured per RPC kind | A unary-only policy: reintroduces the drift the adapter's single-policy design exists to prevent. |
| Age grace value | 10s, applied only when an age is set, and required positive | A value below the unary budget: the force-close at age plus grace would cut an in-flight RPC still inside its own budget. Zero: grpc-go reads it as infinity, so the connection would drain and never close. |
| Liveness bounds default | On | Off with rotation: it would leave the dead-peer slot leak that motivated this rule, and neither bound can cut live work, so they cost a stream nothing. |
| Load-balancing default | Round robin | First-address selection: correct only when the target is single-address, and silently wrong otherwise. |
| Load-balancing delivery | This client's own default service config | Re-enabling resolver service config: reopens a route the client's trust boundary deliberately closed. |

Authorities this spec rests on, each read at the pinned version in `go.mod`:

- `google.golang.org/grpc/keepalive` for the parameter contract, the current
  defaults this spec replaces, the ±10% age jitter, the 1s server ping floor,
  the 10s client ping floor, and the explicit warning that client and server
  keepalive settings must be set in coordination.
- `grpc.WithDisableServiceConfig`'s own documentation, which states that it
  disables resolver-supplied service config only and that a supplied default
  service config is still used. This is what makes R7 compatible with the
  client's existing trust boundary.
- `internal/infra/http/middleware_timeout.go` and `internal/config` for the
  already-accepted handler-budget semantics and value that R3 mirrors.
- `docs/grpc.md` for the already-recorded dispositions this spec keeps: the
  error vocabulary's reopen condition, feature ownership of stream duration
  policy, and reflection's absence.

Not derivable from this repository, and marked as such: the 15m maximum idle
period and the 1m server ping interval are conventional operational values, not
measurements taken here. They are chosen to be well above any reconnect cost and
well below grpc-go's 2h default, and R5's falsifier proves the mechanism rather
than the value. No maximum connection age is shipped, so no such value needs
defending; a deployment that enables rotation owns its own.

Official gRPC guidance recommends that a server monitor cancellation rather than
impose its own deadline. R3 is therefore this repository's decision, not an
external recommendation: it is justified by the specific interaction between a
shedding admission limiter and unbounded handler occupancy, and by the HTTP
transport having already accepted the same trade-off.

## Success criteria and proof expectations

Each criterion names its evidence boundary; mechanism and test placement remain
design and implementation decisions.

1. One domain error carrying a retry delay, driven through both transports,
   produces a gRPC `RetryInfo` equal to the mapper's delay and an HTTP
   `Retry-After` no shorter than it, for a sub-second, whole-second, and
   fractional delay.
2. Two domain errors that share one gRPC code arrive with distinct `ErrorInfo`
   reasons; an unclassified error arrives with neither detail nor handler text;
   every code in the catalog renders to a conforming `Reason`.
3. A context-respecting handler invoked without a caller deadline returns
   `DEADLINE_EXCEEDED` within the unary budget, and the admission budget is
   available afterwards.
4. With the stream budget and rotation off, a stream outlives the unary budget.
   Under shortened idle and ping bounds — the mechanism, not the shipped values,
   which are too long to exercise — the idle bound does not close the connection
   while that stream is outstanding and a ping issued during it is answered
   rather than closing it. With a configured stream budget and rotation off, the
   stream is cancelled at that budget.
5. Under a shortened ping bound, a connection whose peer vanished is closed
   within it, while a connection carrying a long-running RPC is not. Under a
   shortened age, a connection past it is closed and the client recovers on a new
   connection.
6. A parity check fails when the shipped client's default ping interval stops
   exceeding the shipped server's default minimum accepted interval; and a client
   connection with no active RPC is observed to ping, against a peer whose
   enforcement policy rejects the ping.
7. A client against a two-address target reaches both addresses under the
   default policy and one under first-address selection.
8. Server construction with a nil service registration returns an error.
9. Configuration parity holds in both directions for every new bound; all four
   configuration-to-adapter crossings carry them; and every configuration
   relation R4 and R5 state is refused at startup — a negative keepalive
   duration, a stream budget at or above a configured maximum age, and an age set
   with a non-positive grace or with a grace below the unary budget.
10. Existing gRPC proof continues to pass unchanged: interceptor order, both
    error boundaries, correlation acceptance, admission sharing, telemetry
    filtering and sanitization, transport limits, shutdown behavior, the client
    trust boundary and its three strip seams, and the four-cardinality reference
    path.

Concurrency and lifecycle behavior is triggered by R3, R4, and R5, so the proof
for those rules carries race and liveness evidence rather than functional
evidence alone.

No performance claim is made. R3 and R5 add no work to the success path beyond
one timer per RPC and grpc-go's own connection bookkeeping; if a benchmark
comparison is wanted, it is a separate accepted claim with its own workload.

## Risks, assumptions, and reopen conditions

- **Assumption:** no current consumer depends on gRPC statuses arriving without
  details. Safe boundary: details are additive and change neither code nor
  message. Invalidated by a consumer that rejects unknown details. Reopen owner:
  this spec, on evidence of such a consumer.
- **Assumption:** the 8s unary budget is adequate for every operation this
  template's services expose, because the HTTP transport already imposes it.
  Invalidated by an operation whose accepted latency exceeds it. Reopen: that
  operation's owner raises the configured value or moves the operation to a
  stream, which R4 leaves unbounded by default.
- **Risk:** a deployment that enables rotation cuts every RPC and stream older
  than the configured age. grpc-go does not transparently resume a stream that
  has already delivered a message, so the feature owning a long-lived stream must
  handle the failure. Observable: `UNAVAILABLE` on streams at roughly the age
  interval. Owner: the deployment enabling rotation, which R4's configuration
  relation forces it to reconcile against any stream budget it also set.
- **Risk:** a deployment whose load balancer or client library pings more often
  than the configured minimum accepted interval is disconnected. Observable:
  `GOAWAY` with enhanced-calm on connections that previously survived. Owner:
  the deployment, by lowering the minimum. R6's parity check covers only this
  repository's own client half.
- **Risk:** round robin opens a connection per resolved backend where
  first-address selection opened one. Observable: backend connection count and
  the server's own connection bound. Owner: the deployment, by selecting
  first-address behavior for a target where the fan-out is not wanted.
- **Reopen condition on the error vocabulary:** a service that must express
  `ALREADY_EXISTS` or `FAILED_PRECONDITION` reopens the non-goal above and
  extends `problem` and the gRPC rendering together, as `docs/grpc.md` already
  requires.

## Review disposition

The shared review trigger applies: this spec fixes protected-domain,
hard-to-reverse contract and transport decisions.

**First pass:** root self-review only, because the read-only lane carrier was
unavailable at the time. It missed three defects that the downstream Go Ownership
panel then found — R4 and R5 contradicting each other on what bounds a stream,
R1 asserting a parity no implementation could reach, and R2 disagreeing with this
file's own decisions table on the caller-visible reason spelling. That is direct
evidence that self-review is not sufficient for this artifact, not a general
claim about self-review.

**This revision** repairs those rules plus the crossing count. Three independent
read-only rounds then ran over the repaired surface:

1. Whole-artifact review of the repaired spec — `FAIL`. Three blockers: the
   crossing count was fixed in the invariants but not in its own success
   criterion; R4's relation compared against an undefined quantity and claimed a
   purpose no reading delivered; and the stated relations accepted a zero age
   grace, which grpc-go replaces with infinity so the connection would drain and
   never close. Plus concerns on an unobservable falsifier, an over-broad "exactly
   two bounds" claim, and an unfixed identity for the unary budget key.
2. Focused re-review of the repairs — `FAIL` on one: the replacement purpose
   claim for R4's relation was itself false, because the effective cut is the
   jittered age plus a mandatory positive grace.
3. Narrow check of that repair and its two follow-on edits — `PASS`, with three
   non-blocking wording items, all applied.

Every round's findings were re-checked at the anchor by the root before being
applied; none was accepted on the reviewer's word alone.

R1, R2, R6, R7, R8, R9, every non-goal, and every deliberately-unchanged
disposition carry forward unaffected.

Still resting on judgment rather than proof: the two operational values marked as
not derivable above, and the R3 trade-off, which is recorded with its reasoning
and reopen owner.
