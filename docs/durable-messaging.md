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
`natsjs.ErrRejected`; cancellation, timeout, disconnect, or loss before a
conclusive acknowledgement returns `natsjs.ErrAmbiguous`. The pack does not
retry automatically. Retry an ambiguous attempt only with the same
`PublicationID`.

At most `MAX_PENDING_PUBLISHES` synchronous calls are admitted. Each event has
a required logical `MessageID`, publication/deduplication ID, type, schema,
non-zero UTC creation time, and opaque bounded payload. `OrderingKey` is
metadata only: JetStream stream sequence and concurrent handlers do not promise
per-key ordering.

## Worker

Replace the deliberate `nil` in `cmd/worker/main.go` with one feature-owned,
duplicate-safe `natsjs.Handler`, then run:

```bash
make run-worker
make build-worker
```

The production image includes `/worker`; run it with an entrypoint override or
a separate workload definition. It exposes `/health/live`, `/health/ready`, and
`/metrics` on the configured diagnostics listener and owns no HTTP API routes.

The worker uses one-message pulls, caps active handlers at
`MAX_CONCURRENCY`, and rejects a configuration whose resident wire-data bound
exceeds 64 MiB. Success is confirmed with `DoubleAck`. Retryable failures use
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
`nats.go` retries connection 60 times at roughly one-second intervals with no
offline publish buffer. Exhaustion is terminal and starts process drain.

On shutdown, new fetches and publishes stop first. In-flight handlers may finish
within `WORKER__DRAIN_TIMEOUT`; expiry cancels them, closes the connection, and
leaves unacknowledged messages for redelivery. Use the low-cardinality
`messaging.*` metrics and correlated logs/traces for diagnosis. Redrive preserves
the logical message ID and original subject but uses a new publication ID.

Real broker validation is part of the profile:

```bash
REQUIRE_DOCKER=1 make test-integration
go test -vet=off -race ./internal/infra/natsjs ./cmd/worker/internal/bootstrap
```
