**Implemented Test Scope**
- No repository files were edited. This is the exact test scope I would add for the shard reconciler change: app-layer tests for fail-fast sibling shutdown, shared-parent context propagation, unblock-on-cancel behavior, and the `16`-worker cap; infra-layer tests for wrapped database/context error preservation.
- The shard reconciler package is not present in this repo snapshot, so the file paths below are proposed placements that match the existing `internal/app/...` and `internal/infra/postgres/...` layout.

**Scenario Coverage**
- `TestReconcileShards_CancelsSiblingWorkersOnFatalError`: one shard returns a fatal sentinel immediately; sibling workers signal `started`, block on `<-ctx.Done()`, then signal `stopped`; assert the reconciler returns the fatal error and both siblings observe cancellation before the group completes.
- `TestReconcileShards_PropagatesParentContextToRepoHelpers`: use subtests `canceled_parent` and `expired_parent`; pass an already canceled or already expired parent context into the reconciler; fake repo helper immediately checks `ctx.Err()` and returns `fmt.Errorf("repo helper: %w", ctx.Err())`; assert `errors.Is(err, context.Canceled)` or `errors.Is(err, context.DeadlineExceeded)`. This catches any helper that swaps in `context.Background()`.
- `TestReconcileShards_PreservesWrappedContextCanceled`: after workers start, cancel the shared parent; make the repo return `fmt.Errorf("query shard: %w", context.Canceled)`; assert the service surface still satisfies `errors.Is(err, context.Canceled)` and does not collapse into a generic business error.
- `TestReconcileShards_PreservesWrappedDeadlineExceeded`: use an already expired context instead of a sleep-based timeout; make the repo return `fmt.Errorf("scan shard: %w", context.DeadlineExceeded)`; assert `errors.Is(err, context.DeadlineExceeded)` through the service layer.
- `TestReconcileShards_BlockedResultSendUnblocksOnCancel`: use an unbuffered result path or a gated fake sink; one worker signals `enteredSend`, a sibling then returns fatal error, and the blocked sender must exit through `ctx.Done()`; assert the worker returns without hanging. Coordination is channel-driven with `enteredSend` and `released` barriers, not `time.Sleep`.
- `TestReconcileShards_MaxParallelismIsBoundedTo16`: enqueue more than `16` shards; each worker increments an atomic in-flight counter, updates an atomic max, signals `started`, then blocks on `release`; assert the observed max never exceeds `16`.
- `TestShardRepository_PreservesWrappedContextErrors`: table-driven repo test with cases `canceled` and `deadline_exceeded`; DB stub returns wrapped errors via `%w`; assert callers can still detect them with `errors.Is`.

**Key Test Files**
- Proposed: [reconciler_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/shards/reconciler_test.go)
- Proposed: [reconciler_concurrency_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/shards/reconciler_concurrency_test.go)
- Proposed: [shard_repository_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/infra/postgres/shard_repository_test.go)
- Style references already in repo: [config_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/config/config_test.go), [startup_dependencies_additional_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/cmd/service/internal/bootstrap/startup_dependencies_additional_test.go), [main_shutdown_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/cmd/service/internal/bootstrap/main_shutdown_test.go)

**Validation Commands**
- `go test ./internal/app/shards -run 'TestReconcileShards|TestShardRepository'`
- `go test -race ./internal/app/shards -run 'TestReconcileShards|TestShardRepository'`
- `go test -race -count=10 ./internal/app/shards -run 'TestReconcileShards_(CancelsSiblingWorkersOnFatalError|BlockedResultSendUnblocksOnCancel|MaxParallelismIsBoundedTo16)'`
- `make check`
- `make test-race`

**Observed Result**
- Executed adjacent baseline only, because the shard reconciler package is not present in this repository snapshot and edits were disallowed.
- `go test ./internal/config ./cmd/service/internal/bootstrap -run 'TestLoadDetailedWithContextCanceled|TestDegradedDependenciesAbortOnExpiredStartupDeadline|TestDrainAndShutdown'` passed:
```text
ok  	github.com/example/go-service-template-rest/internal/config	0.007s
ok  	github.com/example/go-service-template-rest/cmd/service/internal/bootstrap	0.079s
```
- `go test ./cmd/service/internal/bootstrap -run 'TestDegradedDependenciesAbortOnCanceledStartup|TestDegradedDependenciesAbortOnExpiredStartupDeadline'` passed:
```text
ok  	github.com/example/go-service-template-rest/cmd/service/internal/bootstrap	0.014s
```
- The shard-specific commands above were not executed because those tests were not added.

**Design Escalations**
- The exact reconciler package path and helper names are unresolved in this repo snapshot, so the proposed file paths should be adjusted to the actual package once that code is present.
- If the result channel is fully hidden, add a minimal unexported test seam around result publication instead of weakening the blocked-send test with sleeps or broad end-to-end coverage.
- If the service intentionally maps fatal worker failures into domain sentinels, that mapping must explicitly exempt `context.Canceled` and `context.DeadlineExceeded`; otherwise the preservation tests will expose a design mismatch, not just missing coverage.

**Residual Risks**
- Without the reconciler implementation, I cannot verify whether the real failure surface is `errgroup`, a manual worker pool, or a result-channel fan-in, so one proposed test may need to land one layer lower.
- `go test -race` is necessary here but does not prove absence of goroutine leaks; if the reconciler owns long-lived goroutines, add leak detection in that package as a follow-up.
- The blocked-send regression test depends on a controllable barrier immediately before result publication; if that seam does not exist, introduce the smallest internal seam rather than relying on timing.
