# Durable messaging with NATS JetStream

Select the pack only for a service that uses NATS JetStream:

```bash
make template-init \
  MODULE=github.com/acme/orders \
  CODEOWNER=@acme/backend \
  MESSAGING=nats-jetstream
```

`MESSAGING=none` is the default and removes this guide, `cmd/worker`,
`internal/infra/natsjs`, messaging configuration and tests, NATS dependencies,
worker build/image commands, and all shared profile blocks. Profile selection
is permanent for the initialized checkout.

## Broker contract

The application never creates or updates streams. Before startup, the broker
operator must provide:

- a source stream named by `APP__MESSAGING__STREAM`, covering the worker filter
  subject, with a positive `MaxMsgSize` no greater than
  `WORKER__MAX_DELIVERY_BYTES`;
- a different stream covering `WORKER__DEAD_LETTER_SUBJECT`, large enough for
  `MAX_PAYLOAD_BYTES` plus the fixed 8 KiB envelope ceiling;
- `APP__MESSAGING__MIN_STREAM_REPLICAS` and
  `APP__MESSAGING__MIN_STREAM_RETENTION`, chosen from the deployment's failure,
  detection, repair, and catch-up budgets rather than left at their rejecting
  zero defaults;
- credentials allowed to publish to the source and dead-letter subjects, read
  both streams' configuration and state, and read/create the one named durable
  consumer.

Both streams use file storage, meet the configured replica and retention
minimums, acknowledge JetStream publications, and have a positive duplicate
window. `InterestPolicy`, sealed streams, and per-message TTL are rejected.
Finite message, byte, or per-subject limits must use `DiscardNew`: capacity
exhaustion then rejects a new publish so a durable outbox can retry it, rather
than silently evicting older work. `MaxAge=0` is unlimited; a finite `MaxAge`
must cover `MIN_STREAM_RETENTION`. The application reads and validates this
operator-owned topology at startup and readiness but never reconciles it.

The worker creates the named consumer only when absent. An existing consumer
must match the pack's complete explicit-ack, pull, delivery, and capacity
configuration; startup never rewrites or deletes it.

TLS (`tls://` or `wss://`) and a credentials file are required by default.
`ALLOW_PLAINTEXT` and `ALLOW_UNAUTHENTICATED` are separate, explicit local or
trusted-network escape hatches. URLs, credential paths, broker errors, payloads,
and arbitrary headers are not emitted to telemetry.

## Producer

`cmd/service` owns a producer-only connection. Compose feature publication at
the service bootstrap root with the concrete `*natsjs.Producer`; payload schema
and event meaning remain feature-owned. A publish is accepted only after a
JetStream `PubAck`. Validation or a definite API rejection returns
`natsjs.ErrRejected`; cancellation detected before dispatch is rejected too.
Cancellation, timeout, disconnect, or loss after dispatch but before a
conclusive acknowledgement returns `natsjs.ErrAmbiguous`. The pack does not
retry automatically. Retry an ambiguous attempt only with the same
`PublicationID`. Broker deduplication applies only inside the stream's configured
duplicate window; consumer idempotency still owns retries, replay, and redrive
beyond it.

At most `MAX_PENDING_PUBLISHES` synchronous calls are admitted. Each event has
a required logical `MessageID`, publication/deduplication ID, type, schema,
non-zero UTC creation time, and opaque bounded payload. `OrderingKey` is
metadata only: JetStream stream sequence and concurrent handlers do not promise
per-key ordering. `TestNATSOrderingKeyDoesNotSerialize` pins that as behavior
rather than an omission, so a later change that starts serializing a key has to
argue with a test.

The River outbox worker derives its concurrency from this same admission
capacity: it uses at most 16 workers and never more than
`MAX_PENDING_PUBLISHES`. A capacity refusal remains an ordinary River retry.

### The outbox adds no ordering

A typed outbox event has no ordering key or sequence. Adding producer-only
ordering would not order concurrent JetStream handler effects, so the template
does not create that partial guarantee. A service that needs per-key order end
to end owns the last hop and has two shapes to choose between:

- run the worker at `MAX_CONCURRENCY=1` per key space — one worker process per
  key partition, which trades throughput for order;
- or keep concurrency and re-sequence in the handler, using a key and revision
  carried by the typed payload against state the handler owns.

## Worker

Replace the deliberate `nil` passed to `bootstrap.Run` in `cmd/worker/main.go`
with a binary-local `bootstrap.HandlerBuilder`. The builder receives the loaded
`config.Config` and logger, then returns one binary-local `natsjs.Handler`
adapter that invokes duplicate-safe behavior under `internal/<feature>`, plus
optional `func(context.Context)` dependency cleanup. Feature packages remain
transport-agnostic and do not import `internal/infra/natsjs`. Bootstrap invokes
the cleanup on startup failure and shutdown; it must honor the supplied
deadline.
Then run:

```bash
make run-worker
make build-worker
```

The production image includes `/worker`; run it with an entrypoint override or
a separate workload definition. It exposes `/health/live`, `/health/ready`, and
`/metrics` on the configured diagnostics listener and owns no HTTP API routes.

The worker pulls up to one message per free handler slot, caps active handlers
at `MAX_CONCURRENCY`, and rejects a configuration whose resident wire-data bound
exceeds 64 MiB. Success is confirmed with `DoubleAck`.

`MAX_CONCURRENCY` is therefore both the handler ceiling and the acquisition
batch. A pull costs one broker round trip whatever it returns, so a worker that
asked for one message per pull was capped at one message per round trip however
large its concurrency was; asking for every free slot makes a round trip cover a
batch. The resident bound is unchanged, because a pull never asks for a slot a
running handler holds — `TestFetchBatchShrinksToFreeSlots` pins that. What
remains is one round trip per batch, and a handler faster than that round trip
is still acquisition-bound; raise `MAX_CONCURRENCY` to widen the batch, within
the 64 MiB resident limit.

The durable consumer states the same two bounds as `MaxRequestBatch` and
`MaxRequestMaxBytes`, and startup rejects an existing consumer that does not
match. **Upgrading a running deployment past this change therefore fails
startup against the consumer the previous version created.** Delete the durable
consumer and let the worker recreate it, or update it in place to
`max_batch = MAX_CONCURRENCY` and
`max_bytes = MAX_CONCURRENCY * MAX_DELIVERY_BYTES`, before rolling the new
binary out. Unacknowledged messages are unaffected: they are stream state, not
consumer state, and are redelivered to the recreated consumer. Retryable failures use
the configured delayed NAK sequence. A permanent error
(`natsjs.Permanent(err)`), malformed message, or exhausted attempt budget is
copied to the DLQ and the source is acknowledged only after a DLQ `PubAck`.
The exact source payload and logical identity are preserved. A lost source ACK
can duplicate a DLQ record; consumers and operator replay must be idempotent.

`NumDelivered`, not process memory, owns retry exhaustion, so restarts and lost
ACKs still consume the finite handler budget. Broker delivery remains unlimited
until the DLQ handoff is confirmed. Messages can be delivered more than once;
the pack does not claim exactly-once processing and provides no outbox or
consumer-side database idempotency.

## Lifecycle and operations

Broker unavailability, missing topology, or incompatible consumer configuration
fails startup before listener admission. A disconnect flips readiness false;
`nats.go` retries each server URL in its client pool (configured or discovered)
at most 60 times, waits roughly one second after traversing that pool, and keeps
no offline publish buffer. A connect attempt may additionally spend the fixed
five-second connect timeout, so exhaustion duration grows with the number and
network behavior of pool URLs. Exhaustion is terminal and starts process drain.

On shutdown, new fetches and publishes stop first. In-flight handlers may finish
within `WORKER__DRAIN_TIMEOUT`; expiry cancels them, closes the connection, and
leaves unacknowledged messages for redelivery. Startup rejects a worker whose
drain plus diagnostics, background join, feature cleanup, and telemetry flush
cannot fit inside `HTTP__GRACE_PERIOD`; every shutdown stage draws from that
single process deadline. Use the low-cardinality `messaging.*` metrics and
correlated logs/traces for diagnosis.

The worker's successful readiness probe caches one attribute-free broker
observation for metrics collection; metric callbacks perform no broker I/O.
`messaging.consumer.pending` and `messaging.consumer.ack_pending` expose queued
and unfinished deliveries. `messaging.stream.messages` / `.messages.limit` and
`messaging.stream.storage` / `.storage.limit` expose capacity and headroom; a
limit of `-1` is unlimited. `messaging.stream.oldest.timestamp` and
`messaging.observation.timestamp` are Unix timestamps, so an operator can
distinguish old retained data from a stalled observer. Readiness becomes false
on topology drift while the last successful observation remains available for
diagnosis. The metrics have no stream, consumer, subject, tenant, or message
labels.

Spans follow the OpenTelemetry messaging convention, so they are named
`publish {subject}` and `process {filter subject}`. River adds a linked
`river.work` span, while the outbox worker restores the original producing
context before the NATS publish. The consume span is named after the
worker's configured filter rather than the delivered subject, because a
wildcard filter would otherwise put an unbounded value in the field a tracing
backend groups on; the delivered subject is still on the span as
`messaging.destination.name`. A trace query or dashboard keyed on the previous
static `messaging publish` / `messaging consume` names must be updated.

### Redriving a dead-letter record

A dead-letter record keeps the original payload byte for byte plus the headers
needed to rebuild the publication it came from, so the way back is a republish
rather than a hand-written message. `natsjs.RestoreDeadLetter` is that inverse:
give it a message read from the dead-letter stream and it returns the
`natsjs.Event` to hand to `Producer.Publish`.

```go
reason := natsjs.DeadLetterReason(record)   // "malformed" | "exhausted" | "permanent"
event, err := natsjs.RestoreDeadLetter(record)
if err != nil {
    return err                              // not restorable; inspect it by hand
}
_, err = producer.Publish(ctx, event)
```

Two identities move differently, and both are deliberate. The logical
`MessageID` is preserved, so a consumer deduplicating on its own durable key
treats the redrive as the delivery it already refused. The publication ID is
replaced,
because reusing it would have the broker recognize a duplicate and store
nothing. Its replacement is derived from the dead-letter record's own stream and
sequence rather than minted fresh, so restoring one record twice yields one
publication: a redrive retried after an ambiguous publish deduplicates instead
of delivering the same work twice.

`DeadLetterReason` decides whether a redrive is worth attempting at all. Only
`exhausted` describes a failure a later attempt may survive unchanged.
`permanent` was the handler's own verdict and `malformed` never decoded — the
latter returns `natsjs.ErrRejected` from `RestoreDeadLetter`, because the
envelope that failed to decode is exactly the identity a restore would need.
Address the cause before republishing either.

This pack ships no redrive binary. Which records deserve one, and when, is a
service decision; the transformation is here so it is not re-derived from header
names against every operator's memory.

Real broker validation is part of the profile:

```bash
REQUIRE_DOCKER=1 make test-integration
make test-messaging-race
```
