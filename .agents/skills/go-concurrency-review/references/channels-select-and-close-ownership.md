# Channels, Select, And Close Ownership Examples

## When To Load
Load this when a review touches channel close ownership, send-on-closed risk, result channels, `select` default branches, nil channels, buffered channel capacity, fan-in, fan-out, or queue backpressure.

## Review Lens
Identify the channel owner. The owner decides when all sends are complete and whether close means broadcast, end-of-stream, or shutdown. Buffered channels are finite queues, not proof that a blocked send cannot happen.

## Bad Review Example
```text
[medium] broker/broker.go:131
Closing this channel might be risky if another goroutine uses it.
```

Why it fails: it does not prove a send can race the close, does not anchor the owner mismatch, and does not offer a smallest safe correction.

## Good Review Example
```text
[critical] [go-concurrency-review] broker/broker.go:131
Issue:
Axis: Channel Ownership, Select Behavior, And Blocking; `Unsubscribe` closes `sub.ch` while `Publish` can still send to that same channel from another goroutine, and there is no mutex, owner goroutine, or done handshake ordering "last send" before close.
Impact:
Concurrent publish/unsubscribe can panic with send-on-closed-channel in the broker's hot path, so this is a merge blocker.
Suggested fix:
Make the broker event loop the sole closer and sender for subscription channels, or hold the broker mutex across removal and all sends so close happens only after the sender set is stable. A receiver should not close a channel it does not own.
Reference:
Go memory model channel close/send synchronization and race detector unsynchronized send/close guidance; validate with `go test -race ./internal/broker -run TestPublishUnsubscribeRace -count=100`.
```

## Failure Mode
Write a finding when:
- more than one goroutine can close the same channel;
- a receiver closes a channel that senders still own;
- a send or receive can block forever because no cancellation, close, drain, or bounded buffer policy exists;
- `select { default: ... }` in a loop creates busy-spin or silently drops backpressure;
- a nil channel can accidentally disable the only progress case;
- a buffer capacity is tuned to one example rather than the real maximum number of unconsumed sends.

## Smallest Safe Correction
Prefer corrections like:
- establish one closer and document that all sends finish before close;
- move sends and closes into a single owner goroutine;
- add a `done`/`ctx.Done()` case to sends and receives that may be abandoned;
- replace a default spin loop with a blocking receive, ticker, condition signal, or explicit drop policy;
- close a broadcast-only `done` channel rather than closing a data channel owned by senders;
- make the queue bound and full-queue behavior explicit: block, drop, fail, or shed.

## Validation Evidence
Use validation that stresses send, close, and early return interleavings:
```bash
go test -race ./internal/broker -run TestPublishUnsubscribeRace -count=100
go test ./internal/broker -run TestPublishDoesNotBlockAfterUnsubscribe -count=100 -timeout=5s
```

If the defect is a protocol deadlock, add a liveness assertion with completion channels or a test timeout; `-race` may not catch a deadlock without a data race.

## Source Links From Exa
- [The Go Memory Model](https://go.dev/ref/mem)
- [The Go Language Specification: select statements](https://go.dev/ref/spec#Select_statements)
- [Data Race Detector: unsynchronized send and close operations](https://go.dev/doc/articles/race_detector.html)
- [Go Concurrency Patterns: Pipelines and cancellation](https://go.dev/blog/pipelines)

