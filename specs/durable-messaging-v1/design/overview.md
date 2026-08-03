# Durable Messaging V1 Technical Design

status: complete

## Selected architecture

The capability is one removable, concrete NATS JetStream pack. It uses
`github.com/nats-io/nats.go` v1.52.0 and its `jetstream` package directly. The
real-broker proof pins
`nats:2.14.3-alpine@sha256:c11af972c99ae542de8925e6a7d9c533aa1eb039660420d2074beed6089b3bf0`.
There is no broker interface, framework, outbox, inbox, schema registry,
topology reconciler, or feature event catalog.

The two compositions are deliberately different:

```text
cmd/service -> internal/infra/natsjs.Client -> nats.go/JetStream
              producer + readiness only

cmd/worker  -> internal/infra/natsjs.Client -> nats.go/JetStream
              one durable pull Consumer + one binary-local Handler adapter
```

`cmd/service/internal/bootstrap` owns the HTTP process connection and exposes
the concrete producer at the composition root for feature handlers. The source
template has no feature event to publish, so it does not add a fake endpoint or
an ACK-and-drop handler. `cmd/worker/internal/bootstrap` owns a separate process
lifecycle. Its `main` passes no handler and fails before connecting until the
initialized service registers one binary-local handler adapter that invokes
feature-owned behavior. Focused bootstrap tests inject a real handler without
adding a production example event.

Reopen the transport decision only when the target service must use the
fleet-owned Kafka/Redpanda infrastructure or requires partition-key ordering.
Reopen RabbitMQ only when broker-managed quorum-queue poison handling and
at-least-once dead-letter exchange transfer dominate pull-consumer control.

## Immutable configuration and trust boundary

The selected profile adds this shape to `internal/config`; `MESSAGING=none`
removes the shape, defaults, validation, examples, tests, and every caller:

```go
type MessagingConfig struct {
    Enabled              bool                  `koanf:"enabled"`
    URLs                 string                `koanf:"urls"`
    CredentialsFile      string                `koanf:"credentials_file"`
    RootCAFile           string                `koanf:"root_ca_file"`
    AllowPlaintext       bool                  `koanf:"allow_plaintext"`
    AllowUnauthenticated bool                  `koanf:"allow_unauthenticated"`
    Stream               string                `koanf:"stream"`
    MaxPayloadBytes      int                   `koanf:"max_payload_bytes"`
    MaxPendingPublishes  int                   `koanf:"max_pending_publishes"`
    Worker               MessagingWorkerConfig `koanf:"worker"`
}

type MessagingWorkerConfig struct {
    Consumer             string        `koanf:"consumer"`
    FilterSubject        string        `koanf:"filter_subject"`
    DeadLetterSubject    string        `koanf:"dead_letter_subject"`
    MaxConcurrency       int           `koanf:"max_concurrency"`
    MaxDeliveryBytes     int           `koanf:"max_delivery_bytes"`
    HandlerTimeout       time.Duration `koanf:"handler_timeout"`
    RetryDelays          string        `koanf:"retry_delays"`
    DeadLetterRetryDelay time.Duration `koanf:"dead_letter_retry_delay"`
    DrainTimeout         time.Duration `koanf:"drain_timeout"`
}
```

The selected profile defaults to `enabled=false`. Enabling requires non-empty
URLs and source stream. URLs are a canonical comma-separated list, and URL
userinfo is forbidden. Every URL must use `tls` or `wss` unless
`allow_plaintext=true`; a credentials file is required unless
`allow_unauthenticated=true`. Those two escape hatches are explicit and
independent so changing one value cannot silently disable both transport
security and authentication. `root_ca_file` optionally extends system roots.
URLs, credentials, and paths never enter logs, spans, or metrics.

The common/producer defaults are 256 KiB payloads and 64 admitted synchronous
publishes. Worker defaults are eight concurrent handlers, a 1 MiB maximum
encoded source delivery,
ceiling, a 30-second handler budget, retry delays `1s,5s,30s,2m`, a 30-second
DLQ-transfer retry delay, and a 20-second drain budget. The retry list is one
canonical comma-separated operational input; its length plus the initial
delivery is the finite handler-attempt budget. Validation requires one to nine
positive delays, positive finite values, `max_delivery_bytes` to fit the
application payload plus fixed header ceiling, and
`max_concurrency * max_delivery_bytes` to remain below the package's 64 MiB
resident-wire-data cap. Tests may select short positive delays without changing the
algorithm.

Common and producer configuration is validated by `internal/config` because
either process consumes it. Worker-only required fields and cross-field rules
are validated by `cmd/worker/internal/bootstrap` before connection: non-empty
consumer, filter and DLQ subjects, separate source and DLQ streams, and all
worker bounds. The HTTP process therefore never requires a consumer name or
DLQ topology. The worker rejects `enabled=false`, a nil handler, and every
invalid worker value before broker mutation. After connecting but still before
consumer lookup/creation, worker admission requires the operator-owned source
stream to set a positive `MaxMsgSize <= max_delivery_bytes`; an unbounded or
larger stream is incompatible because it could deliver a message outside the
process memory envelope. The DLQ stream must accept at least the configured
application payload plus the fixed source-metadata/header ceiling. These are
read-only topology checks.

Connection, publish, acknowledgement, retry, and header policy are capability
constants because no current owner needs to tune them:

| Bound | Value |
| --- | --- |
| connect and JetStream management call | 5 seconds |
| synchronous publish and confirmed ACK | 5 seconds |
| reconnect wait / attempts | 1 second / 60 |
| reconnect publish buffer | disabled (`ReconnectBufSize(-1)`) |
| encoded headers | 8 KiB |
| pull expiry | 5 seconds |
| consumer acknowledgement wait | `handler_timeout + 11s` |
| handler attempts | `1 + len(retry_delays)`; default 5 |
| explicit delayed retries | configured validated list; default 1s, 5s, 30s, 2m |
| exhausted-message DLQ transfer retry | configured; default 30 seconds until broker ACK |

Each process completes its applicable config validation before a connection or
consumer mutation. Existing
`DATABASE`, `GRPC`, `AUTHN`, and `OUTBOUND_HTTP` choices remain orthogonal.

## Broker topology boundary

Streams are operator-owned durable storage. The capability never creates,
updates, purges, or deletes a stream. Startup requires the configured source
stream to exist and the dead-letter subject to resolve to a stream. This keeps
replica count, retention, storage class, account limits, and disaster recovery
with the infrastructure owner rather than hiding them in application startup.

The worker owns exactly one application cursor. It first reads the named
consumer. When absent, it creates it with the exact configuration below. When
present, it performs no update: every listed field must already match or
startup fails. Cursor deletion, start-policy change, and repair are explicit
operator actions, never application startup side effects.

The exact application-owned consumer fields are:

- the configured stream and filter subject;
- `AckExplicitPolicy` (never `AckAll`);
- `DeliverAllPolicy` for first creation;
- unlimited broker delivery count, while application handler attempts remain
  finite;
- `MaxAckPending` equal to `max_concurrency`;
- `MaxRequestBatch=1` and `MaxRequestMaxBytes=max_delivery_bytes`;
- `MaxRequestExpires` equal to the five-second pull expiry;
- `AckWait` equal to `handler_timeout +` the five-second DLQ-publish budget
  `+` the five-second confirmed-source-ACK budget `+` one second of scheduling
  headroom. The same ceiling safely covers the shorter success and retry paths;
- `MaxWaiting=2`, while the worker still owns only one local fetch loop. A
  canceled request can remain broker-side until its five-second expiry during
  disconnect; the second bounded slot lets the next single fetch recover rather
  than turning that stale request into a terminal `MaxWaiting` failure;
- `BackOff` empty, `HeadersOnly=false`, `FilterSubjects` empty,
  `InactiveThreshold=0`, `RateLimit=0`, `PauseUntil=nil`, no priority policy or
  groups, no push delivery subject/group/flow-control/heartbeat, inherited
  storage/replicas, and the remaining v1.52 delivery-affecting fields at their
  explicit zero/default values.

Validation is full-structure and fail-closed, not a subset check. Bootstrap
builds the complete desired `jetstream.ConsumerConfig`, canonicalizes nil versus
empty slices/maps, and compares it to `ConsumerInfo.Config` after clearing only
`Description` and `Metadata`, which do not change delivery. Every other field
must equal the desired config. Thus an existing `BackOff` cannot override
`AckWait`, `HeadersOnly` cannot remove the payload, an inactive threshold cannot
delete the durable cursor, and a future non-zero behavior field cannot pass
silently. A client/server upgrade that changes normalized defaults reopens this
single comparison and its real-broker fixture.

Unlimited broker delivery is intentional: after the fifth handler attempt the
handler is never invoked again, but the source remains available until the DLQ
publication receives a JetStream `PubAck`. A finite broker `MaxDeliver` could
strand the last crash or DLQ outage behind an advisory with no recoverable
handoff. Existing incompatible consumer configuration fails startup; the pack
does not update, delete, or recreate a cursor.

`Fetch(1, FetchContext(ctx))` is the only acquisition primitive. Its nats.go
result channels are sized for one message; the capability never calls
`FetchBytes`, whose v1.52 implementation allocates channels for its one-million
sentinel batch even when the server later enforces a small byte limit. The loop
may admit another one-message fetch while fewer than `max_concurrency` handlers
are active, but it stops fetching at that count. The explicit source-stream
`MaxMsgSize` bounds every fetched message by `max_delivery_bytes`. Continuous
`Consume`/`Messages` are not used because their internal refill loop would own
acquisition timing.

The HTTP process validates only the source stream. The worker additionally
validates the DLQ stream and consumer. A missing or incompatible topology is a
startup failure before listener admission.

## Concrete Go boundary

`internal/infra/natsjs` exports concrete types, not an interface hierarchy:

```go
type Event struct {
    Subject, MessageID, PublicationID string
    Type, Schema, OrderingKey         string
    CreatedAt                         time.Time
    Payload                           []byte
}

type Producer struct { /* connection-owned JetStream handle */ }

func NewID() string
func (p *Producer) Publish(context.Context, Event) (PublishResult, error)

type PublishResult struct {
    Stream    string
    Sequence  uint64
    Duplicate bool
}

type Message struct { /* private immutable envelope, payload, and metadata */ }

func (m Message) Subject() string
func (m Message) MessageID() string
func (m Message) PublicationID() string
func (m Message) Type() string
func (m Message) Schema() string
func (m Message) OrderingKey() string
func (m Message) CorrelationID() string
func (m Message) CreatedAt() time.Time
func (m Message) Payload() []byte // returns a clone
func (m Message) Metadata() DeliveryMetadata

type DeliveryMetadata struct {
    Stream, Consumer                 string
    StreamSequence, ConsumerSequence uint64
    NumDelivered, NumPending         uint64
    StoredAt                          time.Time
}

type Handler func(context.Context, Message) error
func Permanent(error) error

type Worker struct { /* one concrete durable pull consumer */ }
func (w *Worker) Run(context.Context) error
func (w *Worker) StartDrain()
func (w *Worker) Shutdown(context.Context) error
```

`NewID` returns `crypto/rand.Text()`. The caller supplies both IDs and a
non-zero UTC creation time before publishing. `MessageID` is the feature event
identity and survives
ordinary retry, DLQ, and redrive. `PublicationID` is the NATS deduplication ID
for one publication attempt and is reused only while resolving an ambiguous
publish. An operator redrive preserves `MessageID` and creates a new
`PublicationID`, so the stream duplicate window cannot suppress the redrive.

Future outbox code calls the concrete `Producer.Publish` and persists the two
IDs with its own record. Future inbox code implements a duplicate-safe feature
handler and returns success only after its own transaction commits. Neither
future capability causes an interface, persistence hook, or transaction owner
inside `natsjs` now.

`Message` has no exported fields. Construction clones broker bytes and every
payload accessor returns another clone, so handler code cannot mutate shared
delivery state. `DeliveryMetadata` exposes broker facts but no connection or
ack primitive. `PublishResult` is populated only from a conclusive JetStream
`PubAck`; `Duplicate=true` means the broker accepted the attempt as a duplicate
within its deduplication window, not that a feature side effect ran.

The feature owns payload encoding and event meaning. The transport carries the
payload as opaque bytes and maps only fixed headers:

| Header | Owner |
| --- | --- |
| `Nats-Msg-Id` | `PublicationID`, set through `jetstream.WithMsgID` |
| `Message-Id` | logical feature event identity |
| `Publication-Id` | current broker publication-attempt identity |
| `Event-Type`, `Event-Schema` | feature-owned type and schema identifier |
| `Ordering-Key` | optional feature-owned key; metadata only |
| `Created-At` | required original UTC RFC3339Nano creation time |
| `Correlation-Id` | trusted request ID from `reqctx`, when present |
| `traceparent`, `tracestate` | explicit OTel `propagation.TraceContext{}` |

No bearer token, baggage, principal, arbitrary application header, or payload
field is copied. Header names and values are length and control-character
validated. `nats.Msg.Size()` verifies the encoded header and total-message
ceilings before publish and before handler admission. The configured stream is
also sent with `jetstream.WithExpectStream`.

NATS stream sequence is observable metadata, not a feature ordering promise.
Concurrent pull processing can reorder all messages, including equal ordering
keys. A service that needs per-key order reopens the transport decision rather
than adding a local lock map.

## Producer completion and connection lifecycle

`Client.Connect` is synchronous and starts no repository goroutine. Broker
unavailability, authentication failure, TLS failure, missing source stream, or
startup deadline aborts before serving. `nats.go` owns socket readers, pings,
and reconnect attempts. The capability configures disconnect, reconnect,
asynchronous error, and close callbacks; callbacks only update atomic state,
fixed metrics, and a one-element terminal-error channel.

`Client.Run(ctx)` is registered in the existing service supervisor. A transient
disconnect makes readiness false while `nats.go` reconnects. A successful
reconnect restores readiness after a bounded management probe. Exhausting 60
attempts closes the connection; `Run` returns a classified error, the
supervisor marks readiness failed, and the process drains. There is no second
reconnect loop and no local publish queue.

`Producer.Publish` builds one immutable NATS message and calls synchronous
`jetstream.PublishMsg` under the earlier of the caller deadline and five
seconds. A server `PubAck` is the only accepted result. Local validation and a
pre-dispatch cancellation or definite JetStream API rejection are `rejected`;
cancellation, deadline, disconnect, or loss after dispatch without a
conclusive acknowledgement are `ambiguous`.
There is no automatic publish retry. A caller may retry an ambiguous operation
with the same `PublicationID`; a fresh logical publication or redrive must use
a new one.

The producer is safe for concurrent callers under the selected `nats.go`
connection and JetStream contracts. Package code adds no producer mutex or
queue. One non-blocking token semaphore admits at most
`max_pending_publishes`; capacity exhaustion returns rejected before cloning
the payload. The finite producer-owned resident payload ceiling is therefore
`max_pending_publishes * (max_payload_bytes + 8 KiB headers)`.

During service shutdown, bootstrap first stops new producer admission, then
HTTP drains so already-admitted handlers retain publish access. The messaging
owner waits for admitted calls within the dependency budget, joins the
supervisor, and calls connection `Drain`; it waits for the closed callback
within the shared shutdown budget and uses `Close` on deadline.

## Pull processing, acknowledgement, retry, and DLQ

One fetch loop issues one-message pulls until `max_concurrency` handlers are
active, then waits on a completion channel of exactly that capacity before
fetching again. At most one pull is outstanding and it allocates result channels
of size one. The source stream's admitted `MaxMsgSize` bounds each active wire
message by `max_delivery_bytes`, so repository/client resident wire data is at
most `max_concurrency * max_delivery_bytes` plus one fixed-size pull result and
bounded per-message metadata. There is no payload queue, unbounded channel,
weighted admission structure, or client sentinel allocation. A slow handler
can consume one slot but does not block refill of other slots; reopen the
acquisition model only when representative throughput evidence shows the
one-message round trips are insufficient.

For each message:

1. Validate encoded size and the fixed envelope before invoking the handler.
2. Extract trace context and correlation ID, start one consumer span, and call
   the feature handler under `handler_timeout`.
3. On success, call `DoubleAck` under the five-second acknowledgement budget.
   A failed or ambiguous acknowledgement leaves the source eligible for
   redelivery.
4. On a transient failure before the configured final delivery, call
   `NakWithDelay` using the configured delay for that attempt. If the NAK is
   lost, the consumer acknowledgement
   timeout still causes redelivery.
5. On a permanent error, malformed poison message, or a handler failure on the
   configured final delivery,
   publish a DLQ copy synchronously. The copy preserves the exact payload,
   logical identity, event metadata, trace headers, and correlation ID, and adds
   bounded original subject, stream, consumer, sequence, delivery count, and
   fixed reason headers. `Original-Subject` is mandatory so a wildcard
   consumer's redrive can reconstruct the destination. The DLQ envelope's
   current `Publication-Id`/`Nats-Msg-Id` is a
   stable transfer ID derived from source stream, sequence, stored timestamp,
   and original publication ID. `Original-Publication-Id` preserves the source
   publication attempt explicitly. This makes an ambiguous DLQ transfer
   deduplicable without confusing it with the original publication or a later
   operator redrive.
6. Only after the DLQ `PubAck`, confirm the source with `DoubleAck`. If the DLQ
   publish or source acknowledgement is ambiguous, leave the source unconfirmed
   and retry transfer after the configured DLQ delay. Duplicate DLQ records
   remain possible across an acknowledgement loss and are explicitly documented
   for operators.

The finite budget is the durable JetStream `NumDelivered` count, not a
process-local failure counter. Every well-formed delivery from one through the
configured final number invokes the handler once; a success on the final
delivery still receives a confirmed ACK. Crashes, lost ACKs, and handler calls
whose outcome was not durably acknowledged consume a delivery because the
transport cannot prove they did no work. A later delivery beyond the budget
does not invoke the handler again and goes directly through the same DLQ
handoff. This is the only crash-stable finite budget without adding the
explicitly excluded inbox persistence.

An oversized source message whose exact payload cannot fit the configured DLQ
is never replaced with a metadata-only record and never acknowledged or
terminated. Readiness becomes degraded, a fixed metric/log identifies the
bounded failure class, and the worker returns a terminal error so process
supervision performs a non-zero restart. A definite DLQ rejection does the
same after leaving the source unconfirmed; an ambiguous connection failure
leaves the source for redelivery while normal reconnect handling owns recovery.
Operators must raise the selected pack's
explicit payload bound or repair/redrive the retained source. This is a visible
poison condition, not silent loss.

The handler may receive duplicate `MessageID` values after process failure,
acknowledgement loss, or redrive. The transport does not claim exactly-once
processing. It exposes the logical ID and attempt; feature code or a future
inbox owns duplicate-safe effects.

## Readiness, degradation, and observability

The `Client` readiness probe requires a connected NATS status and a bounded
JetStream stream lookup. The worker probe additionally requires the durable
consumer lookup and no currently retained local terminal condition. Initial
admission uses the same probe that the cached health watcher later refreshes.
Disconnect, exhausted reconnect, incompatible topology, and an undeliverable
oversized poison message make readiness false. They never make liveness false.

The service appends the client and existing supervisor probes to its current
`health.Service`. The worker owns its own `health.Service` and reuses the
existing private diagnostics server pattern for `/health/live`,
`/health/ready`, and `/metrics`; it does not import the service bootstrap.

OTel scope `service.messaging.nats` owns these low-cardinality instruments:

- `messaging.publish.operations` and `messaging.publish.duration` by
  `outcome=accepted|rejected|ambiguous`;
- `messaging.connection.events` by
  `event=disconnected|reconnected|closed|async_error`;
- an observable `messaging.readiness` gauge by fixed
  `role=producer|worker`, updated from the complete applicable readiness
  verdict (connection, topology, consumer, registration, drain, and terminal
  state), not merely socket status;
- `messaging.fetch.messages` and `messaging.fetch.bytes`, without dynamic
  attributes;
- `messaging.consume.active`, without attributes;
- `messaging.handler.operations` and `messaging.handler.duration` by fixed
  `outcome=success|retryable|permanent|timeout|canceled`;
- `messaging.redeliveries` and `messaging.retries`, without dynamic attributes;
- `messaging.dlq.transfers` by `outcome=accepted|ambiguous|rejected`;
- `messaging.drain.operations` by `outcome=graceful|forced|failed` and
  `messaging.forced_shutdowns`, without dynamic attributes.

No metric uses subject, stream, consumer, event type, logical ID, error text,
or payload. Producer and consumer spans use OpenTelemetry messaging semantic
attributes only for configured names and fixed operation types; message IDs
are span attributes but never metric labels. The pack uses
`propagation.TraceContext{}` directly rather than the global baggage carrier.

Every publish terminal record contains operation, message ID, subject, outcome,
duration, and a fixed failure reason. Every delivery terminal record contains
operation, message ID, subject, configured consumer, attempt, outcome,
duration, and a fixed failure reason. Lifecycle records cover disconnect,
reconnect, readiness transition, drain, force-close, and process-terminal
consumer failure. Logs contain no payload, credential, URL, arbitrary header
value, event type, or raw broker error.

## Worker drain and forced shutdown

The worker is admitted only after config, telemetry, broker, source stream, DLQ
stream, durable consumer, diagnostics, and initial readiness succeed. A signal
or fatal supervised task performs this sequence inside
`messaging.worker.drain_timeout`:

1. mark readiness draining and close the worker's stop-fetch channel;
2. cancel the current pull request without cancelling handler contexts;
3. wait for in-flight handlers and their confirmed acknowledgements;
4. drain the NATS connection and wait for its closed callback;
5. stop diagnostics and flush telemetry last.

If the deadline expires, `Worker.Shutdown` takes its private forced-close path:
it cancels handler contexts, closes the NATS connection immediately, records a
forced-shutdown outcome, and returns a non-nil shutdown error. An uncooperative
feature handler may run until the process exits; its message remains
unacknowledged and redelivers elsewhere. The transport never reports a clean
drain in that state.

## Package and profile placement

| Responsibility | Exact owner |
| --- | --- |
| client, producer, envelope, consumer, retry/DLQ, metrics | `internal/infra/natsjs` |
| HTTP producer composition and lifecycle | `cmd/service/internal/bootstrap/startup_messaging.go` plus marker-scoped calls in `run.go` |
| consumer process composition | `cmd/worker/main.go`, `cmd/worker/internal/bootstrap` |
| immutable runtime config | marker-scoped sections in `internal/config` and `env/.env.example` |
| selected runtime image and commands | marker-scoped `Dockerfile`, `Makefile`, CI/profile wiring |
| operator and feature integration contract | selected-only `docs/durable-messaging.md` and marker-scoped README/architecture text |
| real broker semantics | `test/nats_messaging_integration_test.go` using Testcontainers |
| structural profile proof | `scripts/init-module.sh`, `scripts/ci/template-init-check.sh`, `template.lock` |

The initializer accepts exactly `MESSAGING=none|nats-jetstream`, validates all
profiles before mutation, and then removes marker blocks and selected-only
paths. `MESSAGING=none` deletes `internal/infra/natsjs`, `cmd/worker`, the NATS
integration test, messaging documentation, worker build/run/image/CI commands,
all config and bootstrap sections, and the NATS module dependency through
`go mod tidy`. No messaging path is template-owned because template sync would
reintroduce removed source.

The selected profile builds `/service` and `/worker`; the none profile builds
only the current binaries. Repeated initialization must produce byte-identical
trees. Invalid profile values and combinations fail before copying or editing
the target.

## Proof and reopen conditions

Unit proof owns validation, envelope bounds, outcome classification, retry
classification, drain ordering, and profile transformations. A real pinned
NATS process owns startup outage, post-start loss and reconnect, publish ACK and
timeout ambiguity, pull saturation, redelivery, duplicate delivery, retry
exhaustion, DLQ, poison, ordering observation, trace propagation, and drain.
Race and goleak proof cover the package and worker lifecycle. Process tests own
producer-only service and consumer-worker admission and forced termination.

Reopen this design for a feature requiring partition-key ordering, transactions
spanning database state and publication, broker-managed DLQ guarantees,
multiple independently scaled consumers in one worker binary, payloads above
the selected cap, or a production deployment whose measured concurrency and
drain budget cannot fit the fixed/default bounds. Those are new accepted
outcomes, not hidden extension points in V1.

Independent technical-design review: successive `FAIL` findings closed
count/byte acquisition feasibility, complete settlement timing, durable attempt
semantics, immutable envelope and lineage, process-specific validation,
fail-closed consumer topology comparison, and complete telemetry. The final
fixed candidate received `PASS` with no surviving finding.

A later root boundedness audit rejected `FetchBytes` because nats.go v1.52
allocates sentinel-sized result channels. The repaired `Fetch(1)` plus explicit
stream `MaxMsgSize` and active-handler product bound received a focused
independent `PASS` with no surviving finding.
