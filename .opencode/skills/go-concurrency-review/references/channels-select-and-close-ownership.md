# Channels, Select, And Close Ownership

Behavior Change Thesis: When loaded for channel close, select, or blocked send/receive symptoms, this file makes the model assign a single channel owner and explicit progress policy instead of trusting receiver-side close, buffer capacity, or a `default` branch.

## When To Load
Symptom: the diff touches channel closing, result channels, fan-in/fan-out sends, `select` defaults, nil-channel gating, buffered channel capacity, blocked receives, or sender/receiver abandonment.

## Decision Rubric
- Name the channel owner. The owner is the goroutine or object that knows when all sends are finished and what close means.
- Treat receiver-side close as suspicious unless the receiver is explicitly the channel owner and all senders are ordered before close.
- Treat buffers as finite queues, not liveness proof. Ask what happens when the buffer is full or consumers return early.
- A send or receive that may outlive its peer needs cancellation, close observation, drain, or a bounded fail/drop policy.
- `select { default: ... }` inside a loop is a smell unless the default branch is an explicit non-blocking policy with a sleep, signal, backoff, or accounting story.
- Nil channels are valid select gates only when the code proves they cannot disable the last progress case accidentally.

## Imitate
```text
[critical] [go-concurrency-review] broker/broker.go:131
Issue:
Axis: Channel Ownership, Select Behavior, And Blocking; `Unsubscribe` closes `sub.ch` while `Publish` can still send to that same channel from another goroutine, and there is no mutex, owner goroutine, or done handshake ordering the last send before close.
Impact:
Concurrent publish/unsubscribe can panic with send-on-closed-channel in the broker hot path, so this is a merge blocker.
Suggested fix:
Make the broker event loop the sole sender and closer for subscription channels, or hold one broker mutex across removal and all sends so close happens only after the sender set is stable. A receiver should not close a data channel it does not own.
Reference:
Validate with `go test -race ./internal/broker -run TestPublishUnsubscribeRace -count=100`.
```

Copy the shape: it proves a send can race the close, identifies who should own close, and offers a local correction without rewriting the whole broker.

## Reject
```text
[medium] broker/broker.go:131
Closing this channel might be risky if another goroutine uses it.
```

Reject this shape: it gestures at danger without proving an interleaving, owner mismatch, failure mode, or smallest safe correction.

```go
select {
case jobs <- job:
default:
}
```

Reject this as "non-blocking and therefore safe" unless the silent drop is an explicit contract. Most review findings should ask for a named full-queue policy and validation around the drop/block/fail behavior.

## Agent Traps
- Do not recommend closing from the receiver just to "notify" senders; use a separate done channel or context when close is a broadcast signal.
- Do not treat a buffer size that fits today's test fixture as proof that production sends cannot block.
- Do not miss the dual of send-on-closed: abandoned receivers can strand sender goroutines even when no panic occurs.
- Do not call a `default` branch a fix if it creates a busy loop or hides backpressure loss.

## Validation Shape
- Stress the send/close or send/early-return interleaving with gates, not sleeps.
- Use `go test -race` for send/close races; add a liveness test with a completion channel and timeout for blocked send/receive protocols.
- Good commands look like `go test -race ./internal/broker -run TestPublishUnsubscribeRace -count=100` and `go test ./internal/broker -run TestPublishDoesNotBlockAfterUnsubscribe -count=100 -timeout=5s`.
