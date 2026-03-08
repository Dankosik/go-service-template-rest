**Implemented Test Scope**
No repository files were edited. The test addition would be a focused unit suite around the shard reconciler coordinator, using local scriptable fakes for shard work and repository calls. The goal is to prove fan-out bounds, sibling cancellation, wrapped error preservation, and shutdown/unblock behavior deterministically. Coordination would use explicit channel barriers, atomic in-flight counters, test-owned contexts, and `t.Cleanup`; no `time.Sleep`, scheduler polling, or timing-based assertions.

**Scenario Coverage**
- `TestShardReconciler_LimitsConcurrentWorkersTo16`: run `32` shards through a fake worker that increments an atomic in-flight counter, records `maxInFlight`, signals entry on a channel, and blocks on `release` or `ctx.Done()`. Fail immediately if any worker observes `inFlight > 16`. Assert `maxInFlight <= 16`.
- `TestShardReconciler_CancelsSiblingWorkersAndRetainsFatalDBCause`: start one fatal worker and multiple sibling workers under the same parent context. The fatal worker returns `fmt.Errorf("reconcile shard 3: %w", errDB)`. Siblings wait on `ctx.Done()` and return `ctx.Err()`. Assert the reconcile error still satisfies `errors.Is(err, errDB)` or `errors.As(err, *driverErr)` if the repo exposes a typed DB error, and assert each sibling observed `context.Canceled`.
- `TestShardReconciler_PreservesContextCanceled`: cancel the parent context before or during reconcile, have workers return `ctx.Err()`, and assert the final error satisfies `errors.Is(err, context.Canceled)`. Reject string matching or conversion into a generic business sentinel.
- `TestShardReconciler_PreservesContextDeadlineExceeded`: use an already-expired or test-controlled deadline context, block workers on `ctx.Done()`, and assert the returned error satisfies `errors.Is(err, context.DeadlineExceeded)`.
- `TestShardReconciler_PropagatesSharedContextToRepositoryAndBlockedRepoCallsExitOnCancel`: pass a sentinel context value plus a deadline into reconcile. The fake repository helper asserts that the value and deadline are present, then blocks on `ctx.Done()`. Trigger a fatal sibling error and assert the repository helper exits with `context.Canceled`. This catches accidental `context.Background()` substitution without relying on pointer equality.
- `TestShardReconciler_BlockedResultSendExitsOnCancel`: hold one worker at the result-publication boundary with a `readyToSend` signal and an intentionally unconsumed sink or hookable send seam. Make another worker fail fatally. Assert the parked worker exits through `ctx.Done()` and the reconcile call returns, proving the send path is cancellation-aware instead of a plain blocking send.

**Key Test Files**
- `shard_reconciler_test.go` in the reconciler package, containing the coordinator concurrency/error suite and the local fakes or hook structs needed for deterministic barriers.
- If the repository helper that previously used `context.Background()` is independently callable, add one focused companion test in that helper’s existing `_test.go` file; otherwise keep the propagation assertion inside `shard_reconciler_test.go`.

**Validation Commands**
- `go test ./... -run 'TestShardReconciler_(LimitsConcurrentWorkersTo16|CancelsSiblingWorkersAndRetainsFatalDBCause|PreservesContextCanceled|PreservesContextDeadlineExceeded|PropagatesSharedContextToRepositoryAndBlockedRepoCallsExitOnCancel|BlockedResultSendExitsOnCancel)$' -count=1`
- `go test -race ./... -run '^TestShardReconciler_' -count=1`
- `make test`
- `make test-race`

**Observed Result**
Not run. This task explicitly disallowed repository edits, so no new tests were added and no fresh validation was executed.

**Design Escalations**
- These tests assume a fatal worker failure should surface the original wrapped worker or repository cause, while still preserving `errors.Is` recognizability for `context.Canceled` and `context.DeadlineExceeded`. If the service intentionally normalizes fatal errors into a stable domain sentinel, that contract needs to be specified before locking assertions.
- The blocked-send regression should only be tested through a real cancellation-aware seam. If the send boundary is too inlined to hold deterministically, add a minimal package-private hook or helper before implementing the test instead of falling back to sleep-based synchronization.

**Residual Risks**
- This suite would prove coordinator behavior, shared-context propagation, and wrapped-error semantics, but it would not prove that the real database driver aborts promptly on cancellation. If that is critical, add one integration-tagged cancel/shutdown test at the real repository boundary.
- The proposed coverage is scoped to worker shutdown, unblock-on-cancel behavior, concurrency limits, and context error preservation. If shard enumeration, retry classification, or partial-progress semantics also changed, those need separate obligation-driven tests.
