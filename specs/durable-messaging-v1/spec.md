# Opt-in durable messaging with a separate worker lifecycle

status: complete

## Scope and non-goals

An initialized service chooses exactly one structural profile: `MESSAGING=none` or `MESSAGING=nats-jetstream`.

`none` is the baseline and contains no messaging-owned source, module dependency, configuration key, environment example, binary, command, test, integration fixture, CI/profile branch, image artifact, or documentation. It is not a disabled runtime branch.

`nats-jetstream` retains one concrete, direct JetStream capability for service-side production and a separately composed consumer worker. Feature event subjects, payload schemas, compatibility, business ordering, side-effect idempotency, and handler behavior remain feature-owned.

Outbox/inbox persistence, broker deployment, cloud resources, fleet rollout, public ingress, production capacity, schema-registry selection, generic broker APIs, and a second transport are non-goals. A later outbox/inbox may consume the narrow message and handler boundary defined here without changing its at-least-once meaning.

## Behavior and contract delta

### M01 — Structural initialization

- Profile values outside `none|nats-jetstream` are rejected before any file mutation.
- Successful initialization records the selected value, removes the unselected profile physically, deletes generator-only sources, and leaves a valid initialized repository.
- Repeating the same valid initialization is byte-stable. Repeating with a different value fails before mutation.
- Profile-owned markers or selected-only content never enter verbatim template-sync paths.
- `MESSAGING` composes orthogonally with every currently valid `DATABASE`, `GRPC`, `AUTHN`, and `OUTBOUND_HTTP` selection. It introduces no new invalid combination and does not weaken or retain content removed by another profile.

### M02 — Runtime activation and composition

- Retaining the pack does not silently require a broker for unrelated HTTP work. Service-side producer activation is explicit runtime configuration.
- When activated, the producer connection is a startup dependency: unusable configuration, unavailable broker, rejected credentials, or unusable required topology fails startup before readiness.
- The service process owns producer-only usage. It never runs a consumer loop.
- The separate worker process owns consumer connection, acquisition, handler admission, readiness, drain, and forced shutdown. It requires at least one consumer-owned handler registration and rejects an empty or conflicting registration before broker mutation.

### M03 — Message identity and feature payload boundary

- Each logical message has one stable, non-empty message ID across ordinary delivery, duplicate delivery, DLQ, and redrive.
- Each broker publication has a publication-attempt ID used for broker-ingestion deduplication. Every retry after an ambiguous result reuses that publication-attempt ID. A redrive preserves the logical message ID but uses a new operator-supplied redrive/publication-attempt ID, so a prior broker deduplication window cannot suppress the requested delivery.
- The transport envelope carries: logical message ID, publication-attempt ID, event type, schema version, optional ordering key, correlation/request ID, trace context, creation time, subject, and opaque feature-owned payload bytes.
- Transport validation rejects absent/invalid required metadata, reserved-header conflicts, an invalid subject, or any configured message/header bound before publication.
- Logical message ID is the feature correlation/idempotency input. Publication-attempt ID is the broker-ingestion deduplication input. Neither proves that a feature side effect is idempotent.
- Consumers receive an immutable message view plus broker delivery metadata. They do not receive a broker connection or a public ACK primitive.

### M04 — Producer completion

- A publish succeeds only after the broker accepts the message into the expected durable stream and returns its acknowledgement within the caller deadline or the smaller configured publish budget.
- Validation, pre-dispatch cancellation, a definite API rejection, stream mismatch, or resource-limit response is rejected before or conclusively at dispatch. Cancellation, deadline, disconnect, or loss after dispatch without a broker acknowledgement is ambiguous about whether the broker stored the message.
- Producer cancellation stops waiting and never converts an unknown result into success.
- The producer has a finite pending-publication and memory bound. Capacity exhaustion rejects new work; it never grows an unbounded queue or reports false acceptance.

### M05 — Pull delivery and acknowledgement

- The worker uses a named durable pull consumer and admits no more than the configured count, byte, concurrency, and handler-time budgets.
- At the concurrency or admitted-byte bound, the worker issues no new fetch and leaves backlog in the broker; it creates no unbounded local queue.
- A handler return of success means its feature-owned duplicate-safe completion boundary is durable; only then does the runtime issue a confirmed positive acknowledgement.
- A handler error is not acknowledged as success. Cancellation or forced shutdown leaves incomplete work eligible for redelivery.
- Duplicate delivery is valid after publish ambiguity, consumer loss, ACK ambiguity, retry, DLQ handoff interruption, or redrive. Feature handlers must use natural idempotency, durable uniqueness, or version fencing when repeated effects would be unsafe.

### M06 — Retry, poison, exhaustion, DLQ, and redrive

- Retryable handler failures receive delayed redelivery under one finite attempt budget. The attempt count includes the initial delivery.
- Deterministic envelope corruption, unsupported transport metadata, and a consumer-declared permanent feature error are poison and skip transient retries.
- Poison or final-attempt failure is copied to the configured DLQ with stable source identity, bounded failure classification, original delivery metadata, trace/correlation metadata, and the exact original opaque payload.
- The source is terminated/acknowledged only after the DLQ publish is broker-acknowledged. If DLQ publication fails or is ambiguous, the source remains recoverable and may redeliver.
- A metadata-only diagnostic record is never a successful DLQ handoff and never authorizes source termination. When the exact payload cannot fit the configured DLQ envelope, the source remains unacknowledged/retained, readiness exposes terminal handling failure, and operator recovery reads the retained source record.
- The crash window after DLQ acceptance and before source termination can duplicate the DLQ record. DLQ consumers and redrive therefore use stable source identity and remain duplicate-safe.
- Broker max-delivery advisory is diagnostic evidence, not the only durable dead-letter record. If application handoff did not complete, the retained source record remains the recovery authority.
- Redrive is an explicit operator/feature action. It preserves the logical message ID, requires a new unique redrive/publication-attempt ID, and checks feature-owned schema compatibility. Success means the broker accepted a new deliverable publication rather than returning a duplicate acknowledgement for the old attempt. The pack does not auto-redrive or delete DLQ data.

### M07 — Ordering and horizontal scale

- The transport preserves and propagates an opaque ordering key but promises neither global ordering nor feature-effect completion order under concurrent delivery, redelivery, or multiple worker instances.
- Explicit per-message acknowledgement is mandatory; acknowledgement-all behavior is forbidden.
- A feature needing stronger order must use a dedicated serialized consumer or feature-owned version/idempotency guards. That choice is outside this pack and cannot be inferred from the ordering key alone.

### M08 — Reconnect, readiness, and degradation

- Initial connection/topology admission is fail-closed. After a successful start, a connection loss triggers bounded-memory reconnect behavior with jitter and no false publish success.
- Service readiness is false while an activated producer cannot reach the required JetStream account/topology. Producer-disabled HTTP service readiness is unaffected.
- Worker readiness is false until the connection, required stream, durable consumer, and handler registrations are usable; it becomes false before drain and throughout reconnect or terminal consumer failure.
- Liveness never fails solely because the remote broker is unavailable. Exhausted reconnect or an unrecoverable consumer/runtime failure is surfaced to process supervision and causes a non-zero worker/service exit as owned by its composition root.
- Readiness recovery requires a fresh successful broker/topology observation; a stale pre-disconnect success is insufficient.

### M09 — Trace, correlation, telemetry, and data safety

- Publication injects W3C trace context and the repository request/correlation ID into reserved transport headers. Consumption extracts them before starting a consumer span and invoking the handler.
- Untrusted incoming propagation is validated by the installed propagator; transport-reserved metadata cannot be overridden by feature headers.
- Logs identify operation, message ID, subject, consumer, attempt, outcome, and low-cardinality failure reason without payloads, credentials, broker URLs containing secrets, arbitrary event type, or raw error text as metric labels.
- Metrics cover publish outcome/latency, reconnect state, readiness, fetched/in-flight work, handler outcome/latency, redelivery, retry, DLQ outcome, drain outcome, and forced shutdown using fixed-cardinality labels.

### M10 — Graceful drain and forced shutdown

- Shutdown first fails readiness and stops new publication/acquisition admission.
- The worker drains already-admitted work and confirmed acknowledgements within its finite shutdown budget, then drains the broker connection.
- When the deadline expires, handler contexts are cancelled, pending fetches stop, the connection is force-closed, and the process exits within the remaining platform budget. Unacknowledged work remains redeliverable.
- Producer-only service shutdown stops new publish admission, waits only for already-admitted publish completions within the owned dependency budget, then drains or closes the connection before telemetry flush.
- A successful drain and a forced shutdown are observably different outcomes; neither reports incomplete work as acknowledged.

## Invariants and edge cases

- End-to-end delivery is at-least-once. No text, API, metric, or test may describe arbitrary feature effects as exactly once.
- A business commit and direct publication are not atomic. A feature requiring state-plus-message atomicity must add an outbox in a separate workflow.
- A successful feature effect followed by process loss before confirmed ACK may repeat; handler correctness cannot depend on in-memory deduplication.
- Broker/server limits and application limits both apply. The application rejects above its smaller bound even when a broker would accept more.
- Malformed and oversized deliveries received from a non-pack publisher never reach the feature handler. They are dead-lettered only when the exact original payload and required bounded metadata can be accepted; otherwise the source remains retained/recoverable and readiness/telemetry expose the terminal handling failure.
- Missing DLQ topology, rejected DLQ publish, or DLQ outage never causes source acknowledgement.
- Recreating a durable consumer under a different name or incompatible start policy is not recovery; startup rejects incompatible required topology rather than silently replaying another range.
- `MESSAGING=none` treats every messaging environment key as unknown configuration and has no worker or messaging build target.

## Decisions, constraints, and authorities

- Transport/client implication: ready research selects direct NATS JetStream with `nats.go v1.52.0`; System Design owns the exact topology and lifecycle realization.
- Broker-accepted data and durable-consumer state are transport authorities. Feature side-effect success, schema compatibility, poison classification, ordering requirement, and idempotency are feature authorities.
- The initialized source tree and `template.lock` are profile authorities. Runtime flags cannot restore content removed by `MESSAGING=none`.
- Configuration is one immutable startup snapshot. Secrets come only from the repository's accepted secret sources and never from committed examples.
- Numeric defaults must be positive, finite, internally consistent, and below the documented broker/server envelope. System Design may choose values only if they preserve every observable bound and the test plan can falsify them.

## Success criteria and proof expectations

- A real broker proves startup failure, post-start connection loss, reconnect, accepted/rejected/timed-out publish, producer cancellation, bounded saturation, success/failure ACK behavior, redelivery, duplicates, finite retry, poison, DLQ, and the two DLQ handoff crash directions.
- Process/lifecycle proof shows readiness transitions, producer-only service composition, consumer worker composition, graceful in-flight drain, deadline exhaustion, forced shutdown, no leaked goroutines, and race-clean concurrent operation.
- Bound proof shows oversized/malformed messages never reach handlers, configured concurrency is never exceeded, and admitted message memory cannot exceed the configured count/byte envelope.
- Propagation proof correlates producer span, wire headers, consumer span, request ID, logs, and fixed-cardinality metrics without payload leakage.
- Initialization proof demonstrates invalid-profile pre-mutation rejection, `none` physical purity and dependency/config/command absence, selected producer and worker builds/tests, orthogonal minimal/maximal combinations with existing profile dimensions, preservation of all previously valid combinations, and repeated byte stability.
- Completion evidence is limited to local capability semantics and profile structure. It does not claim deployed broker availability, production throughput, fleet adoption, or a rollout.

## Risks, assumptions, and reopen conditions

- Application-owned JetStream DLQ handoff is accepted because the outcome requires explicit dead-letter handling, not atomic broker-native DLX. Reopen transport selection if broker-managed delayed retry/DLX becomes the primary invariant or RabbitMQ's credit-driven receive and lifecycle close the required pull/drain contract.
- Kafka fleet precedent does not bind this transport-neutral template. Reopen transport selection for a target service required to reuse that Kafka/Redpanda fleet or to preserve per-key partition ordering across horizontal workers.
- Feature handlers are assumed able to expose a duplicate-safe completion boundary. A feature with non-idempotent, non-fenceable effects blocks consumer activation until a separate durable idempotency/inbox design closes it.
- Real-process proof assumes a local container runtime. If unavailable after implementation, completion is blocked with narrower unit proof; it is never reported as real-broker acceptance.

Independent specification review: initial `FAIL` on redrive identity, payload-less DLQ termination, and existing-profile compatibility; repaired candidate received focused `PASS` with no surviving finding.
