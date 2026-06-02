# Queue, Worker, Broker, Topics, And Lag Proof

Status: targeted full-rollout research complete
Date: 2026-06-02

## Questions

Which broker strategy, topics, producer/consumer ownership, and worker
deployment facts must specification reopen decide for the full Railway rollout?

## Broker Findings

Live Railway read-only inventory on 2026-06-02 found no billing-specific broker
service in production. Railway template search found Kafka templates, including
single-node KRaft options, but found no template matching `redpanda`.

The billing-service code uses Kafka protocol clients from `segmentio/kafka-go`
(`internal/infra/redpanda/kafka_client.go:11`). The runtime names the config
`redpanda`, but the implementation requires broker host:port values and Kafka
consumer/producer protocol compatibility, not a Redpanda-specific API.

Default topic and group names are already in config:

- terminal topic: `billing.microlease.terminal.v1`;
- checkpoint topic: `billing.microlease.checkpoint.v1`;
- close topic: `billing.microlease.close.v1`;
- billing facts topic: `billing.microlease.facts.v1`;
- consumer group: `billing-service-microleases`.

When Redpanda is enabled, all four topics and the consumer group must be
non-empty (`internal/config/validate.go:260`). There is no repository code that
creates topics or reads back topic retention, partition count, or lag state.

## Worker Findings

The `billing-worker` binary exists in source, but the current Docker image does
not build or copy it. `build/docker/Dockerfile:15` builds `/service`,
`build/docker/Dockerfile:23` builds `/migrate`, and the final stage copies only
those binaries plus migrations. Railway can override a Dockerfile service start
command, but a `/billing-worker` start command would fail until the image
contains that binary.

`cmd/billing-worker/internal/bootstrap/run.go:45` exits successfully when
`microlease.worker_enabled` is false. Therefore a Railway worker service can
look healthy while doing no work unless production variables explicitly enable
worker mode and dependencies.

The worker opens Postgres, builds three Kafka consumers, and builds a Kafka
producer (`cmd/billing-worker/internal/bootstrap/runtime.go:54`). It consumes
terminal, checkpoint, and close topics with the same configured consumer group
(`cmd/billing-worker/internal/bootstrap/runtime.go:84`) and relays billing
facts through the outbox producer (`cmd/billing-worker/internal/bootstrap/runtime.go:242`).

The accepted inbound producer identity is currently hard-coded to
`gonka-proxy` (`cmd/billing-worker/internal/bootstrap/runtime.go:184`). Worker
outbox events use producer identity `billing-service`
(`cmd/billing-worker/internal/bootstrap/runtime.go:18`). Each worker task has
`MaxConcurrency: 1`, including the three consumers, inbox retry, outbox relay,
stale reconciliation, and admission-control renewal
(`cmd/billing-worker/internal/bootstrap/runtime.go:191`).

## Sibling Proxy Evidence

The sibling `gonka-proxy` checkout is dirty and contains untracked microlease
draft files. Focused searches found no Kafka or Redpanda producer implementation
and no Kafka/Redpanda dependency in the current source. The untracked durable
microlease allocator commits child debits and terminal obligations locally, but
it does not publish terminal/checkpoint/close events.

## Evidence Limits

Absence of a producer is based on the current dirty checkout and package files,
not a clean provider contract. It must be rechecked when proxy work is cleaned
or approved.

## Handoff Implications

Specification reopen must choose one broker strategy:

- strict Redpanda via custom Docker/image/service, including persistence and
  internal DNS; or
- Kafka-compatible broker via Railway Kafka template, with explicit acceptance
  that config names stay `redpanda` while the deployed service is Kafka.

The spec must also decide topic creation/read-back ownership, partition count,
retention, consumer group ownership, lag proof, producer identity policy, and
whether scaling beyond one worker replica is allowed before partition/lag proof.
