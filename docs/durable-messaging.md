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
- credentials allowed to publish to the source and dead-letter subjects and to
  read/create the one named durable consumer.

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
`PublicationID`.

At most `MAX_PENDING_PUBLISHES` synchronous calls are admitted. Each event has
a required logical `MessageID`, publication/deduplication ID, type, schema,
non-zero UTC creation time, and opaque bounded payload. `OrderingKey` is
metadata only: JetStream stream sequence and concurrent handlers do not promise
per-key ordering. `TestNATSOrderingKeyDoesNotSerialize` pins that as behavior
rather than an omission, so a later change that starts serializing a key has to
argue with a test.

### Ordering does not compose with the outbox

The [PostgreSQL transactional outbox](postgres-transactional-outbox.md) does
guarantee claim order per ordering key, and that guarantee ends at this pack.
The worked adapter in `test/postgres_outbox_natsjs_integration_test.go` forwards
`Event.OrderingKey` onto the JetStream envelope, so the key survives as data a
handler can read — but the relay hands one key's events to the broker in order
and nothing after that keeps them in it: JetStream assigns its own stream
sequence, and a worker with `MAX_CONCURRENCY` above one runs handlers for the
same key concurrently.

A service that needs per-key order end to end owns the last hop and has two
shapes to choose between, neither of which this pack decides:

- run the worker at `MAX_CONCURRENCY=1` per key space — one worker process per
  key partition, which trades throughput for order;
- or keep the concurrency and re-sequence in the handler, using the key and a
  sequence the publisher put in the payload, against state the handler owns.

Until one of those exists, treat the composed guarantee as at-least-once
delivery of correctly ordered *publications*, not ordered *processing*.

## Worker

Replace the deliberate `nil` passed to `bootstrap.Run` in `cmd/worker/main.go`
with a binary-local `bootstrap.HandlerBuilder`. The builder receives the loaded
`config.Config`, logger, and admitted concrete `*natsjs.Producer`, then returns
one binary-local `natsjs.Handler` adapter that invokes duplicate-safe behavior
under `internal/<feature>`, plus optional `func(context.Context)` dependency
cleanup. Feature packages remain transport-agnostic and do not import
`internal/infra/natsjs`. Bootstrap invokes the cleanup on startup failure and
shutdown; it must honor the supplied deadline.
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
the pack does not claim exactly-once processing and provides no outbox or inbox.

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
correlated logs/traces for diagnosis. Redrive preserves the logical message ID
and original subject but uses a new publication ID.

Real broker validation is part of the profile:

```bash
REQUIRE_DOCKER=1 make test-integration
make test-messaging-race
```
