# Durable Messaging V1 Research

status: complete

Valid as of: 2026-08-01. Refresh before implementation if a selected broker/client reaches a newer incompatible major release, enters deprecation, or repository `go.mod`/profile machinery materially changes.

## Decision questions

1. Which one transport can meet the accepted durable producer, pull-consumer, retry/DLQ, bounded-resource, reconnect, readiness, and drain behavior without a generic façade or an unavailable fleet decision?
   - Owner affected: System / Integration Design.
   - Live substitutes: NATS JetStream, Apache Kafka, RabbitMQ; an additional fleet-authoritative option survives only if current repository or fleet evidence identifies one.
   - Leading hypothesis: JetStream is the smallest coherent pack for a general Go microservice template because its maintained official Go client exposes acknowledged publish, explicit count/byte-bounded pull fetches, per-message ACK/redelivery, server-side limits, reconnect events, and client-native drain without Kafka's partition/group burden or RabbitMQ's credit-driven receive and application-tracked drain.
   - Falsifier: an official limitation prevents bounded pull consumption, explicit retry exhaustion/DLQ, safe reconnect/drain, or adequate Go-client concurrency; or current repository/fleet authority mandates another already-operated transport.
2. Does a direct official Go client dominate a maintained framework such as Watermill for the selected single transport?
   - Owner affected: Go Code / Ownership Design.
   - Leading hypothesis: direct integration dominates because broker semantics and lifecycle are part of the capability, while a framework adds an abstraction surface the accepted outcome explicitly rejects.
   - Falsifier: the official client lacks a required safe lifecycle/telemetry/concurrency primitive that a maintained framework supplies without hiding or weakening transport semantics.
3. What exact failure and recovery contract remains application-owned despite broker delivery or exactly-once claims?
   - Owner affected: Specification and Test Design.
   - Leading hypothesis: at-least-once delivery with ACK only after successful handler completion; duplicates remain possible after effect-before-ACK failure; message IDs support correlation and future inbox adapters but are not an idempotency guarantee; retry is finite and poison/exhausted messages reach a separately owned DLQ.
   - Falsifier: official semantics can atomically cover arbitrary application side effects without outbox/inbox participation.
4. Which existing repository owners and profile seams can carry the capability without contaminating `MESSAGING=none`?
   - Owner affected: System / Integration and Go Ownership Design.
   - Leading hypothesis: concrete adapter under `internal/infra/<transport>`, producer composition under `cmd/service/internal/bootstrap`, separate `cmd/worker`, existing config/health/telemetry/background owners, and the existing initialization transform/test harness are sufficient.
   - Falsifier: an owner or proof seam cannot be made profile-local without changing a template-owned verbatim path or leaving messaging dependencies/sources behind.

## Lens map

- Current repository lifecycle/profile baseline: researched with CodeGraph first; independent repository lane in progress.
- Current official broker/protocol contracts: independent primary-source lane in progress.
- Current Go clients and Watermill: independent maintainer-source lane in progress.
- Representative production Go-service integration: independent implementation-evidence lane pending capacity.
- Distributed failure and recovery: independent adversarial lane pending capacity.
- Conflict/freshness: triggered because client releases and broker contracts are current and the selection is hard to reverse; an independent synthesis challenge is required before Specification.
- Empirical behavior: documentation cannot prove real reconnect/redelivery/drain behavior in this repository; Test Design must retain real-broker proof.

## Candidate frame

The decision slot is the one concrete durable broker transport shipped by the optional initialization pack. Kafka, RabbitMQ, and JetStream are substitutes at that slot. Watermill is not a transport substitute; it is an optional client-side framework over a chosen transport. Outbox/inbox are complementary persistence patterns and explicitly outside this workflow. Feature event schemas, business idempotency, and domain ordering rules remain feature-owned prerequisites/consumers of the transport contract.

Scanned rungs: repository reuse (no messaging owner found); Go standard library (no durable broker); native broker capabilities (the three live substitutes); already-operated fleet infrastructure (no authoritative decision found yet); maintained OSS clients/frameworks; custom broker/lowest-common-denominator abstraction rejected by scope and YAGNI.

## Established repository baseline

- `cmd/service/internal/bootstrap` owns staged configuration, telemetry, dependencies, startup admission, HTTP/gRPC composition, and bounded shutdown.
- `internal/background.Supervisor` owns supervised long-lived tasks, readiness failure reporting, and deadline-bounded shutdown.
- `internal/health.Service` owns cached readiness and transition probing; external dependency failure is not liveness.
- `internal/infra/telemetry` owns bounded-cardinality runtime metrics and OpenTelemetry composition.
- Optional profile initialization is pre-mutation validated and physically removes source/config/docs/tests/dependencies/tooling, with repeated-initialization stability checked by `scripts/ci/template-init-check.sh`.
- `internal/infra/postgres/pgtest` already establishes a Testcontainers pattern for real dependency proof.
- Profile-specific markers cannot live in `template-owned.paths` entries; initialization-owned documentation and source must remain outside that verbatim sync surface.
- `cmd/service/internal/bootstrap` is Go-`internal` scoped to the service command. A sibling `cmd/worker` needs its own composition package and may share only repository-wide `internal/...` adapters, telemetry primitives, configuration types retained by the selected profile, and feature-owned behavior. Its binary-local handler adapter maps the NATS transport message into that behavior.
- `internal/background.Supervisor` deliberately outlives the HTTP drain and is stopped before dependency close and telemetry flush. Producer-side supervised work must preserve that order; worker consumption needs an equivalent worker-local drain owner rather than a second generic supervisor.
- The runtime Dockerfile and `Makefile` currently build only `/service` (and `/migrate` in the database profile). A worker binary is not real until the selected profile builds and copies it explicitly; existing Railway configuration does not prove or authorize a second deployment.
- A removed configuration struct field becomes an unknown `APP__...` key during immutable config load. That is the fail-closed runtime oracle for `MESSAGING=none`, in addition to filesystem/module/command absence.
- Bounded local fleet search found direct Kafka use in `usage-history-service` and `analytics-service` through `franz-go v1.21.5`, and in `billing-service` through `kafka-go v0.4.48`. This makes Kafka the only currently observed fleet-operated candidate; exact lifecycle/operational applicability remains under independent source inspection.

## Current Go client and framework evidence

- NATS JetStream has an official Apache-2.0 Go client, [`nats.go v1.52.0`](https://github.com/nats-io/nats.go/releases/tag/v1.52.0). Its current API exposes context-bounded synchronous publish acknowledgement, bounded pull `Fetch`, explicit ACK/double-ACK/NAK/delayed-NAK/termination, delivery metadata, concurrent-safe connections, bounded reconnect controls, headers, and drain. DLQ transfer remains application-owned.
- Kafka's vendor-owned Go client is [`confluent-kafka-go v2.15.0`](https://github.com/confluentinc/confluent-kafka-go/releases/tag/v2.15.0); bounded producer completion requires consuming delivery reports, it brings cgo/librdkafka, and `Flush` is millisecond-budgeted rather than context-owned. The fleet-used pure-Go [`franz-go v1.21.5`](https://proxy.golang.org/github.com/twmb/franz-go/@latest) has the better direct Go surface (`ProduceSync`, bounded polling, manual commits, hooks), but is community rather than Apache/Confluent owned. Both require an application-owned DLQ/offset handoff.
- RabbitMQ has the official BSD-2-Clause [`amqp091-go v1.13.0`](https://github.com/rabbitmq/amqp091-go/releases/tag/v1.13.0), publisher confirms, pull `Get`, QoS, manual ACK/NACK, headers, and broker DLX. Its new automatic reconnect/topology recovery is explicitly experimental; no drain primitive closes the whole application lifecycle.
- Watermill is eliminated for this pack. Its common publisher has no per-call context, its examined NATS and AMQP adapters use push consumption, its NATS adapter uses the legacy JetStream API, and its generic message/router boundary hides transport semantics that this task must expose. Reopen only if a later accepted outcome independently requires the Watermill Router/CQRS ecosystem and relaxes direct bounded-pull ownership.

## Cross-source synthesis

### Delivery and recovery invariant

All viable transports remain at-least-once at the feature-effect boundary. A successful broker publish acknowledgement proves broker acceptance under that broker's durability configuration; an error or timeout does not prove absence. A consumer must acknowledge only after the feature effect has reached its durable duplicate-safe completion boundary. A crash after that effect and before broker acknowledgement can repeat the effect. No broker-level deduplication window, Kafka transaction, double-ACK, or DLQ policy makes an arbitrary external side effect exactly once.

Outbox and inbox remain separate workflows. This pack must therefore expose stable message identity and raw feature payloads without claiming state-plus-event atomicity or consumer idempotency. A future outbox can call the concrete producer with its already-stable message ID; a future inbox can persist the received message and return handler success before the runtime ACKs it.

### Same-level candidate comparison

| Candidate | Evidence-backed fit | Material cost / rejection pressure | Design implication |
| --- | --- | --- | --- |
| NATS JetStream + `nats.go` | Official current Go client; context-bounded server-acknowledged publish; explicit bounded pull fetch; per-message ACK/double-ACK/NAK-with-delay/termination; durable consumers; finite `MaxAckPending`/`MaxDeliver`; built-in reconnect events and drain; stream/account size limits. | `MaxDeliver` retains the source and emits an advisory rather than atomically moving it to a DLQ; deduplication is only a rolling window; concurrent pulls/redelivery do not preserve feature-effect ordering; no current fleet operation was found. | Survives when DLQ handoff is explicitly application-owned and ordering is not overstated. This is the smallest direct client/lifecycle surface matching every mandatory runtime primitive. |
| Kafka/Redpanda + `franz-go` | Already used on authoritative fleet mains; pure-Go synchronous publish, pull polling, partition groups, stable per-key partition ordering, rebalance control, and proven worker lifecycle patterns. | No Apache-owned Go client; ACK is offset/partition state rather than an individual-message primitive; finite delayed retry/DLQ requires retry topics or partition-blocking application policy; fetch byte settings admit an exceptional oversized first batch; topology and group/rebalance operation are heavier. | Dominates only when an initialized service is required to reuse the observed Kafka/Redpanda fleet or needs partition-order/replay as a primary contract. That fleet requirement is not part of this transport-neutral template outcome. |
| RabbitMQ 4.3 quorum queues + official AMQP 1.0 Go client | Publisher confirms, explicit settlement, bounded link credit, finite poison delivery limits, linear delayed retry, consumer timeouts, and conditional at-least-once DLX transfer provide the strongest broker-native retry/dead-letter path. The source retains dead letters until target confirms. | Activating at-least-once DLX requires a quorum source, `dead-letter-strategy=at-least-once`, `overflow=reject-publish`, and a configured DLX; explicit routing is optional. Durable target configuration and persistent source publication separately govern restart survival. An unavailable or unroutable target retains source messages against queue limits and periodic transfer can duplicate at the target. The broker DLQ worker keeps a bounded prefetch of bodies in memory. AMQP 0-9-1 `basic.get` is discouraged and not prefetch-bounded; AMQP 1.0 `Receive(ctx)` is credit-driven rather than an explicit count/byte fetch, and the current Go client does not supply the end-to-end pause/unsettled-work drain sequence. | Dominates if broker-native delayed retry/DLQ is the primary invariant and credit-bounded receive plus application-tracked drain are accepted. It does not dominate this outcome's explicit bounded pull and graceful-drain infrastructure criterion. |
| PostgreSQL LISTEN/NOTIFY or table queue | PostgreSQL is an existing optional repository profile. | `LISTEN/NOTIFY` is session notification, not durable messaging; a `SKIP LOCKED` table queue would be a new custom broker plus retention/repair system. | Eliminated: not the requested durable transport and violates the smallest-owner/YAGNI boundary. |

The design implication is one direct NATS JetStream pack using `nats.go v1.52.0`. The decisive accepted criteria are an explicit application-issued count/byte-bounded pull operation and a maintained Go client with native reconnect events plus drain. RabbitMQ 4.3 is better at broker-owned delayed retry and DLQ transfer, but satisfying the same pull/drain contract requires a credit-driven receiver plus more application lifecycle and topology policy. JetStream's smaller missing piece is one explicit, testable DLQ-publish-before-source-termination handoff whose duplicate window is already part of the accepted at-least-once contract. This is not a fleet topology decision and creates no NATS deployment: the initialized service must point at operator-supplied JetStream.

Reopen the transport decision if the accepted target service is required to reuse the observed Kafka/Redpanda fleet, if broker-managed delayed retry and conditional at-least-once DLX become more important than explicit fetch/drain ownership, if credit-driven `Receive(ctx)` is accepted as the intended pull contract and current RabbitMQ Go lifecycle proof closes the drain gap, or if per-key ordered horizontal processing becomes a stronger invariant than explicit per-message ACK and bounded pull.

Watermill remains eliminated because its common publisher lacks a per-call context contract, the examined NATS adapter uses the legacy JetStream push path, and its abstraction hides the exact broker completion and lifecycle semantics this pack exists to demonstrate.

### JetStream failure and recovery consequences

- Publish with a stable `Nats-Msg-Id`. On timeout, retry only the same logical message ID; a later `PubAck.Duplicate` proves deduplication only inside the configured stream duplicate window.
- Use a named durable pull consumer with `AckExplicit`, finite `MaxAckPending`, finite `MaxDeliver`, and one fetch count/byte/in-flight budget. Ephemeral ordered consumers are inspection tools, not restart recovery or load-balanced workers.
- Use delayed NAK for transient handler failure; plain NAK bypasses `AckWait`/backoff. Work must finish within the ACK budget or explicitly report progress under a separately bounded policy.
- A handler success triggers double-ACK. Its timeout remains ambiguous, so feature handlers still require a durable natural-idempotency, uniqueness, or version-fencing boundary.
- Deterministic poison and exhausted retry publish a stable DLQ record first and require its `PubAck`; only then terminate/ack the source. A crash between those operations can duplicate the DLQ handoff. Reversing the order can lose it. DLQ consumers and redrive must therefore use the stable source identity and remain duplicate-safe.
- `MaxDeliver` advisories are observability, not the sole durable handoff. The retained source record is the recovery authority when application DLQ transfer did not complete.
- Concurrency gives no global or completion order. `AckAll` is prohibited. Ordering-sensitive features must serialize their own consumer or enforce feature-owned version/idempotency rules; the opaque ordering key is propagated but does not become a false transport guarantee.
- Drain stops acquisition, lets admitted work finish inside the worker shutdown budget, then drains/closes the connection. Deadline exhaustion cancels handlers and force-closes; unacknowledged messages redeliver and an already-completed effect may repeat.

### Quantities and provenance

- `nats.go v1.52.0`: current external release, as of 2026-08-01.
- NATS server default `max_payload` 1 MiB and configurable maximum 64 MiB: external limit. The pack must choose a smaller explicit application envelope limit and manage stream `MaxMsgSize`; defaults are not accepted capacity targets.
- JetStream default stream duplicate window 2 minutes and consumer `MaxAckPending` default 1000: external defaults, not accepted safety values. Design must configure explicit finite values derived from its retry/drain envelope.
- Local fleet versions `franz-go v1.21.5` and `kafka-go v0.4.48`: measured repository baselines from authoritative main checkouts, not recommendations for this pack.
- Throughput, backlog, message-size distribution, partition count, and broker SLO: unknown and not required to select a bounded opt-in reference pack. Any production capacity claim reopens Performance/Test Design with a representative workload.

## Source boundary and limits

Primary contracts: [JetStream concepts](https://docs.nats.io/nats-concepts/jetstream), [JetStream consumers](https://docs.nats.io/nats-concepts/jetstream/consumers), [NATS reconnect](https://docs.nats.io/using-nats/developer/connecting/reconnect), [NATS drain](https://docs.nats.io/using-nats/developer/receiving/drain), [`nats.go v1.52.0`](https://github.com/nats-io/nats.go/tree/v1.52.0), [Kafka design](https://kafka.apache.org/43/design/design/), [RabbitMQ acknowledgements/confirms](https://www.rabbitmq.com/docs/confirms), and the current [RabbitMQ 4.3 quorum queue delayed-retry, poison, and at-least-once DLX contract](https://www.rabbitmq.com/docs/quorum-queues#delayed-retry). Maintainer-source comparison also inspected current Confluent, franz-go, RabbitMQ Go, and Watermill releases. Representative implementation evidence inspected fixed authoritative fleet mains and Redpanda Connect; those examples establish workable ownership patterns, not universal policy.

No official source can prove real reconnect timing, redelivery, drain, goroutine cleanup, memory bounds, or process composition in this repository. Those remain real-broker and process-level Test Design obligations.

## Stop rationale and downstream disposition

Every named transport family, client authority, repository owner, delivery ambiguity, lifecycle boundary, and strongest viable counterexample has a disposition. Further generic broker searches are unlikely to change the chosen decision slot. Specification receives: at-least-once only; stable identity; duplicate-safe feature boundary; explicit bounded pull/ACK/retry/DLQ/drain behavior; no ordering or exactly-once overclaim; and structural `MESSAGING=none` absence. System Design receives the direct JetStream implication and the three objective transport reopen conditions above.

Independent synthesis challenge: the initial candidate failed because it understated RabbitMQ 4.3 delayed retry, poison limits, and at-least-once DLX. After the focused repair, a fresh challenger returned `CONCERNS` only for activation-versus-survival wording; the sentence above corrects that bounded concern. No research-owned blocker remains.
