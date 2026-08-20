---
name: reliable-messaging
description: "Broker publish, consume, identity, ordering, acknowledgement, redrive, replay, and business-effect depth reference."
---

# Reliable Messaging

Use only after the active phase identifies a broker-backed delivery decision.
Inherit its authority, artifact, review, proof, output, and completion contract;
do not select a work mode or authorize broker operations here.

Core invariant: state each guarantee from its start through durable publish,
broker acceptance, durable effect, acknowledgement, and recovery. One logical
message identity survives retry, redelivery, redrive, and replay; effectively
once applies only to a named durably idempotent business effect.

Load one branch:

- boundary guarantee, identity, publish/effect commits, or acknowledgement ->
  [boundary.md](references/boundary.md);
- Kafka, RabbitMQ, SQS, or JetStream behavior -> the one matching broker
  reference;
- ordering, backpressure, retry, quarantine, redrive, replay, drain, retention,
  or signals -> [operations.md](references/operations.md);
- executable failure proof for a multi-boundary claim ->
  [failure-proof.md](references/failure-proof.md).
