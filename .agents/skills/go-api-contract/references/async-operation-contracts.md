# Async Operation Contracts

## When To Load

Load this when `202 Accepted` is under consideration, when work outlives the
request that started it, or when this service publishes events to consumers who
will build on their delivery.

## Behavior Change Thesis

Without this file, `202` is answered as soon as the work is handed to a
goroutine, a channel, or a queue client. `202` is a promise that the service
will complete the work or report why it did not, so acceptance has to be durable
before the response is written. In this repository the boundary is a
`postgresoutbox` row committed in the same transaction as the state change and
published by `cmd/outbox-relay` — anything earlier means a process restart
between the response and the enqueue silently drops work the client was told was
accepted.

## Decision Rubric

- Answer `202` only once the service owns completion or failure reporting.
  Name the commit that makes that true; if there is no such commit, the honest
  answer is a synchronous status the client can act on.
- Give the client exactly one durable recovery path and say which it is: an
  operation resource, the authoritative business resource with its lifecycle
  state, or a notification backed by a reconciliation read. Two paths without an
  authority ruling is an ambiguity, not redundancy.
- Publish `Location` for the operation or created resource, and `Retry-After`
  when polling cadence matters.
- Model states clients can act on differently. `pending`, `running`, `succeeded`,
  and `failed` are the base; add `canceling`, `canceled`, or `expired` only when
  a client can reach them. A single `completed` state hides the failed and
  cancelled cases behind the word clients read as success.
- Define retention and what a read after expiry answers — `410 Gone`, a
  business-resource read, or a durable result reference. An operation URI that
  becomes `404` is indistinguishable from one that never existed.
- Cancellation is a separate promise from rollback. State whether it is best
  effort, compensating, or impossible past a named point.
- An event this service publishes carries a stable event id for deduplication
  and a monotonic resource version, and names the read that reconciles it.
  Consumers lose, duplicate, and reorder deliveries; a notification with neither
  identity nor version cannot be reconciled against a read.

## Reject

- Describing the execution side here — lease, visibility timeout, retry budget,
  poison handling, drain. `durable-background-jobs` owns it, and a second
  description is where the two disagree.
- Writing the outbound delivery of a webhook as a contract clause: posting to a
  consumer endpoint makes this service the caller of a system it does not
  control, which is `external-api-integration`. This file owns the event shape
  and the reconciliation promise only.

## Validation Shape

A consumer-runnable proof: kill the process between the accepted response and
the work starting, then show the operation still reaches a terminal state.
Read the operation URI after its retention window and observe the documented
expiry answer rather than a bare `404`.
