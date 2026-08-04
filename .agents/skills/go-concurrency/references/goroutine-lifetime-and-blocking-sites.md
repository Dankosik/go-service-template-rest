# Goroutine Lifetime And Blocking Sites

## Behavior Change Thesis
When loaded for goroutine, channel, context, or shutdown symptoms, this file makes the model enumerate every blocking site in the goroutine and name what unblocks each one, instead of accepting "it takes a context" or "the channel gets closed" as a lifetime story.

## When To Load
Symptom: the diff starts goroutines, closes channels, sends or receives on them, uses `errgroup`, worker loops, background watchers, request cancellation, shutdown joins, or early returns that can abandon a peer.

## Decision Rubric
- List the blocking sites inside the goroutine — channel send, channel receive, `Wait`, lock acquisition, sleep, network and database calls — and name the signal that unblocks each. Cancellation unblocks only the sites that observe it; a select without a `ctx.Done()` arm and a downstream call that never received the context are both still parked.
- Every goroutine needs an owner, a stop signal, and a join point, or a stated reason it is process-lifetime and abandoned. `internal/background.Supervisor` is this repository's owner for process-lifetime work: it holds the cancelable context, contains panics, joins through the group, and reports a failed task to readiness. Work that belongs there should not be launched with a bare `go`.
- Read `internal/background` before proposing changes to it. It uses a bare `errgroup.Group` rather than `errgroup.WithContext` on purpose, so one failing task does not cancel its siblings, and `Shutdown` reports a task that outruns the budget rather than blocking the rest of the drain. Both look like defects until the recorded reason is read.
- The context from `errgroup.WithContext` is canceled when the first function returns a non-nil error **or when `Wait` returns**, whichever comes first. Cleanup or follow-up work after `Wait` needs its own context.
- Name the channel's owner: the goroutine or object that knows all sends are finished. Close is a broadcast to receivers; a sender that needs to be told to stop needs a context or a separate done channel, never a close on the channel it sends to.
- Review both halves of an abandoned pair. Send-on-closed panics and is loud; a receiver that returns early strands its senders silently, which is the half diffs usually miss.
- `govet`'s `lostcancel` runs in `make lint` and catches a discarded `CancelFunc`. Spend review on the paths it cannot see: a `cancel` stored, passed on, or called on only one return path.

## Reject
- "It uses `errgroup.WithContext`, so siblings stop on error" — cancellation reaches a sibling only where the sibling observes it while blocked and passes it into downstream calls.
- "The goroutine exits when the channel is closed" — that is a claim about the closer, not this goroutine. Show the path on which close always happens, including the error and panic paths.
- A `select { default: }` described as non-blocking and therefore safe — the default branch is a policy, and the review needs its drop, retry, or accounting story.

## Validation Shape
- `go.uber.org/goleak` runs `VerifyTestMain` in `internal/background`, `internal/infra/http`, `internal/infra/natsjs`, `internal/infra/postgres`, `internal/infra/telemetry`, `cmd/service/internal/bootstrap`, and `cmd/worker/internal/bootstrap`, so a leak in those packages already fails their tests. A new package that starts goroutines has no such gate until one is added.
- For a lifetime claim, park the goroutine at the blocking site with a handshake, cancel or stop, then assert the exported call returns; a test timeout is the diagnostic, not the proof.
