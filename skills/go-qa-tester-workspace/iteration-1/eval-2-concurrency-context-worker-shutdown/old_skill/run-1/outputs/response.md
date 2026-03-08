**Implemented Test Scope**
- No repository files were edited. Exact test scope I would add: app-layer tests for shard-worker fail-fast shutdown and infra-layer tests for repository context propagation and wrapped cancellation semantics.
- Obligations covered: shared parent `context.Context`, sibling stop-on-fatal behavior, preservation of wrapped `context.Canceled` and `context.DeadlineExceeded`, unblock-on-cancel for blocked sends, and the `16`-worker parallelism cap.
- Determinism rule for all tests: use channel handshakes, explicit parent cancellation, and atomic counters; do not use `time.Sleep` for coordination.

**Scenario Coverage**
- `TestReconcileShards_CancelsSiblingWorkersOnFatalError`: one shard returns `fatalErr` immediately; sibling workers block on `<-ctx.Done()` and report `cancelObserved`; assert the reconciler returns `fatalErr` and both siblings stop promptly.
- `TestReconcileShards_PreservesWrappedContextCanceled`: cancel the shared parent after workers start; make the repo return `fmt.Errorf("query interrupted: %w", context.Canceled)`; assert `errors.Is(err, context.Canceled)` at the service boundary and that no generic business error replaces it.
- `TestReconcileShards_PreservesWrappedDeadlineExceeded`: run with an expired or tightly bounded context; make the repo return `fmt.Errorf("scan shard: %w", context.DeadlineExceeded)`; assert `errors.Is(err, context.DeadlineExceeded)` and keep the wrapper intact.
- `TestReconcileShards_BlockedResultSendUnblocksOnCancel`: one worker reaches the result-send path and signals `enteredSend`; a sibling fails fatally and cancels the shared context; assert the blocked sender exits via `ctx.Done()` instead of hanging on a channel send.
- `TestReconcileShards_UsesSharedParentContextForEveryRepoCall`: fake repo helper captures the incoming context and blocks until `ctx.Done()`; cancel the parent after `started`; assert the helper returns `context.Canceled`, which would fail if any helper used `context.Background()`.
- `TestReconcileShards_MaxParallelismDoesNotExceed16`: queue more than `16` shards; each worker increments an atomic in-flight counter and blocks on `release`; assert the observed max never exceeds `16`.
- `TestShardRepository_PreservesWrappedContextCancellation`: at the repository/helper layer, make the DB stub return wrapped `context.Canceled` and wrapped `context.DeadlineExceeded`; assert callers can still detect both with `errors.Is`.

**Key Test Files**
- `internal/app/shards/reconciler_test.go`
- `internal/app/shards/reconciler_concurrency_test.go`
- `internal/infra/postgres/shard_repository_test.go`

**Validation Commands**
- `go test ./internal/app/shards -run 'TestReconcileShards|TestShardRepository'`
- `go test -race ./internal/app/shards -run 'TestReconcileShards|TestShardRepository'`
- `go test -race -count=10 ./internal/app/shards -run 'TestReconcileShards_(CancelsSiblingWorkersOnFatalError|BlockedResultSendUnblocksOnCancel|MaxParallelismDoesNotExceed16)'`
- `go test ./internal/infra/postgres -run 'TestShardRepository_PreservesWrappedContext'`
- `go test -race ./internal/infra/postgres -run 'TestShardRepository_PreservesWrappedContext'`

**Observed Result**
- Executed adjacent baseline only, because edits were disallowed and the shard reconciler package is not present in this repo snapshot.
- `go test ./cmd/service/internal/bootstrap -run 'TestDrainAndShutdown|TestProbeWithRetry|TestStartupProbeHelperBasics|TestDependencyInitFailurePreservesWrappedCause|TestInitRedisDependency(AddressErrorClassifiedAsDependencyInit|PolicyDenialRemainsPolicyViolation)'` -> `ok github.com/example/go-service-template-rest/cmd/service/internal/bootstrap 0.125s`
- `go test -race ./cmd/service/internal/bootstrap -run 'TestDrainAndShutdown|TestProbeWithRetry|TestStartupProbeHelperBasics|TestDependencyInitFailurePreservesWrappedCause|TestInitRedisDependency(AddressErrorClassifiedAsDependencyInit|PolicyDenialRemainsPolicyViolation)'` -> `ok github.com/example/go-service-template-rest/cmd/service/internal/bootstrap 1.125s`
- The shard-specific tests above were not executed because they were not added.

**Design Escalations**
- The reconciler package and exact repository helper names are absent from this repo snapshot, so the file paths above assume the existing `internal/app/...` and `internal/infra/postgres/...` layout.
- If service code intentionally maps fatal worker errors into domain sentinels, that mapping must explicitly exempt `context.Canceled` and `context.DeadlineExceeded`; otherwise the preservation tests represent a design mismatch, not just missing coverage.

**Residual Risks**
- The blocked-send case may need a small test seam around the worker result sink if the current implementation hides that channel completely.
- `go test -race` is necessary here, but it does not prove absence of goroutine leaks; if the reconciler owns long-lived goroutines, add `goleak` verification in that package as a follow-up.
