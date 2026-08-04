---
name: reliable-messaging
description: "Delivery discipline for application-to-broker-to-application boundaries. Use when designing or changing publish/consume behavior, diagnosing loss, duplicates, ordering, retries, redrive, or replay, or proving a broker-backed business effect across Kafka, RabbitMQ, Amazon SQS, NATS JetStream, or an equivalent broker. Broker operations belong here when they change that delivery boundary."
---

# Reliable Messaging

Delivery is an application lifecycle, not a broker feature:

`business state -> message identity -> durable publish -> broker acceptance -> consume -> durable effect -> acknowledge -> recover -> prove`

State every guarantee with its start, durable commit, end, and accepted loss or duplicate window. Use “effectively once” only for a named business effect protected by durable idempotency. Reserve “exactly once” for a proven atomic boundary; broker deduplication alone is not an end-to-end guarantee.

## Choose the depth

Follow only the path needed by the request:

- **Review or diagnose:** reconstruct the affected boundary and its immediate dependencies, locate the earliest durable step that violates the claimed guarantee, and report the smallest correction and falsifier. Keep state unchanged.
- **Design:** define each in-scope boundary, assumption, mechanism, and proof. Use a ledger when several boundaries interact; one row is enough for a narrow design.
- **Build or fix:** patch the shared root cause locally and run the smallest test that fails on the old boundary and passes on the new one.
- **Operate:** publishing, configuration changes, redrive, replay, purge, deletion, or credential rotation requires explicit authorization for the exact environment and bounds. Preflight, perform only the authorized action, and verify with fresh readback.

Do not expand a narrow acknowledgement, identity, or routing issue into a full messaging redesign unless another boundary can change the diagnosis or make the fix unsafe. A production-readiness claim, new end-to-end design, migration, redrive, or replay does require the complete in-scope lifecycle.

Honor an explicitly requested artifact or test count. Add another item only when omitting it would make the requested result unsafe; otherwise record the extra concern as a gap.

## Load the matching contract

Read only the references that match the observed broker and requested concerns, and verify version-specific behavior against current primary documentation:

- [references/kafka.md](references/kafka.md) for Kafka acceptance, offsets/groups, partition order, retention, replay, or configuration.
- [references/rabbitmq.md](references/rabbitmq.md) for routing/confirms, acknowledgements/requeue, prefetch, quorum queues, recovery, or configuration.
- [references/sqs.md](references/sqs.md) for Standard/FIFO delivery, visibility/deletion, group order, deduplication, DLQ/redrive, or configuration.
- [references/jetstream.md](references/jetstream.md) for JetStream publish acknowledgement, consumers, acknowledgement/redelivery, retention, replay, or configuration.
- [references/operations.md](references/operations.md) when ordering/concurrency/backpressure, retry/quarantine, redrive/replay/reconciliation, deploy/drain, retention/capacity, security, or operational signals are in scope.

For an unfamiliar broker, derive the same lifecycle from its current official contract and label unverified behavior.

## Frame the boundary

Start from the business operation and protected effect. Record loss and duplicate tolerance, latency, outage window, fan-out, replay need, ordering scope, throughput, and the current evidence. First decide whether a broker earns its cost:

- use a synchronous call when the caller needs the result and both sides share availability;
- use a database-backed job when one database owns the work and independent fan-out or long replay is unnecessary;
- use a broker for required durable buffering, failure isolation, independent consumers, fan-out, smoothing, or replay.

Classify the message as an immutable **event**, a single-owner **command**, or a competing-worker **work item**. Write the guarantee as:

`<semantic> from <start> after <durable evidence> through <end>; <effect> is idempotent by <key and scope>`

For a multi-boundary design or readiness review, keep one row per publish or consume boundary. For a narrow task, retain only the affected row and mark decision-changing unknowns:

| Boundary | Guarantee/start/end | Identity/scope | Durable commit | Ambiguity | Recovery owner | Signal | Falsifier |
| --- | --- | --- | --- | --- | --- | --- | --- |

## Define identity and schema

Give one logical message a producer-assigned identity that survives publish retries, redelivery, redrive, and replay. Keep it separate from delivery tags, offsets, receipt handles, sequence numbers, and broker-generated IDs.

Include only envelope fields that change behavior: message ID/type, schema version, occurred-at time, correlation and causation IDs, tenant/security scope, ordering key when needed, and trace or payload-integrity metadata when useful. A correlation ID groups work; it is not a deduplication key. Scope idempotency by `(tenant, consumer/effect, message_id)` or a stronger immutable business key.

Scope by the stable logical effect contract, not a mutable process, deployment, or consumer-instance name. When changing an existing deduplication key, seed the new identity for effects already applied before replaying only proven missing identities; otherwise the migration can reapply the first effect it was meant to preserve.

Name the schema owner and compatibility rule. Prefer additive evolution with tolerant consumers deployed before new producers. Define the retirement condition for old readers/writers and protect sensitive payloads and metadata.

## Make publication durable

Locate the first durable publish-intent commit. When business state and intent share a database, commit the state change and outbox row together. The outbox carries the final identity and immutable routing/schema data.

Relay with bounded claims and retries: publish through the broker’s durable acceptance mechanism, mark published only after acceptance is observed, reuse the same logical identity when the response is lost, and reclaim expired claims. Broker transactions do not join an unrelated business database transaction.

If no atomic outbox or equivalent exists, name the loss/duplicate window and reconciliation owner. Bound publisher queues and in-flight requests so backpressure reaches ingress or durable storage rather than an unbounded memory queue.

## Commit the effect before acknowledgement

Assume another attempt can exist whenever acknowledgement, visibility, ownership, or replay can repeat delivery. For a transactional effect, atomically commit the business effect and an inbox/idempotency record protected by a unique key. For an external effect, pass the same durable business key downstream or enqueue it through a local outbox.

Acknowledge, delete, or commit an offset only after the durable effect commits. A crash after effect commit but before acknowledgement must converge on the recorded effect. Treat a lost acknowledgement response as ambiguous: reread durable state and accept safe redelivery. A check-then-act query without a unique constraint or atomic downstream operation leaves a concurrent duplicate gap.

Choose deduplication retention from the longest broker retention, redelivery, redrive, offline-recovery, and replay horizon plus margin. Keep a permanent business key when repeating the effect is never valid.

## Prove the claimed boundary

For a multi-boundary design, readiness review, migration, replay or redrive plan, or a task requesting runnable code or tests, read [references/proof-receipt.md](references/proof-receipt.md) and complete its private receipt before finalizing. Skip it for a single-boundary factual diagnosis unless the user requests a proof artifact.

Choose tests from the guarantee, not from the mechanism’s happy path. At minimum, inject failures on both sides of each in-scope durable commit:

- business commit versus publish intent and broker acceptance versus relay completion;
- effect commit versus acknowledgement, including lost responses and concurrent duplicates;
- ownership expiry or transfer, poison input, retry exhaustion, and replay when those paths are in scope;
- mixed versions, authorization denial, and cross-tenant attempts when those risks are in scope.

If the design introduces quarantine, enumerate a separate bounded-redrive test in the Proof artifact; do not leave it implicit in rollout prose. The test preserves original identity, checks duplicate suppression and business state, and stops on a predeclared invariant.

Pin broker/client versions, topology, durability settings, workload, concurrency, and fault injection. Verify business state, logical identity, broker progress, attempts, quarantine, and recovery signals. A falsifier is useful only if removing the claimed commit or idempotency mechanism makes it fail.

For a narrow review, a causal finding, minimal target, recovery owner, and runnable falsifier complete the task. For production readiness, every ledger row must define its guarantee, identity, durable commits, ambiguous outcome, bounded recovery, signal, and executable falsifier.

## Report

Lead with the verdict and current state: proposed, implemented locally, deployed, or verified live. Use only sections that carry decision-changing information:

```markdown
## Verdict
## Boundary and guarantee
## Root cause or design
## Recovery and operations
## Proof and remaining gaps
## Authorization required
```
