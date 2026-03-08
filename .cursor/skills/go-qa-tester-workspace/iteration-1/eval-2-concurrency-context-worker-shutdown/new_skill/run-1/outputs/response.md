**Implemented Test Scope**
No repository files were edited. I would add a focused unit-test suite in the reconciler package, with local scriptable fakes for shard work and repository calls, because the changed behavior is coordination and error semantics rather than storage integration. All coordination would be deterministic: unbuffered/bounded channels for start barriers and exit signals, `context.WithTimeout` only as a test guard, atomic in-flight counters for the fan-out limit, and no `time.Sleep` or scheduler-dependent polling.

**Scenario Coverage**
- `TestShardReconciler_LimitsConcurrentWorkersTo16`: run `32` shards through a fake worker that increments an atomic in-flight counter, records `maxInFlight`, signals `entered`, and blocks on `release` or `ctx.Done()`. Fail immediately if any worker observes `inFlight > 16`. Assert `maxInFlight <= 16` after completion.
- `TestShardReconciler_CancelsSiblingWorkersAndRetainsFatalDBCause`: start one fatal worker and at least two sibling workers under the same parent context. The fatal worker returns `fmt.Errorf("reconcile shard 3: %w", errDB)`. Siblings block on `select { case <-ctx.Done(): return ctx.Err() }`. Assert the reconciler error satisfies `errors.Is(err, errDB)` or `errors.As(err, *typedDBErr)` if the repo uses a typed driver error, and assert each sibling observed `context.Canceled`.
- `TestShardReconciler_PreservesContextCanceled`: cancel the parent context before or during reconcile, have workers return `ctx.Err()`, and assert the final error still satisfies `errors.Is(err, context.Canceled)`. Do not compare strings and do not accept translation into an uninspectable generic business error.
- `TestShardReconciler_PreservesContextDeadlineExceeded`: use an already-expired or test-owned deadline context, block workers on `ctx.Done()`, and assert the returned error satisfies `errors.Is(err, context.DeadlineExceeded)`.
- `TestShardReconciler_PropagatesSharedContextToRepositoryAndBlockedRepoCallsExitOnCancel`: give the parent context a sentinel value and a deadline. The fake repository helper asserts that the value and deadline are visible, then blocks on `ctx.Done()`. Trigger a fatal sibling error and assert the repo helper exits with `context.Canceled`. This catches accidental `context.Background()` substitution without relying on pointer equality.
- `TestShardReconciler_BlockedResultSendExitsOnCancel`: park one worker at the result-publication boundary with a `readyToSend` signal and an intentionally unconsumed sink or hookable send path. Make another worker fail fatally. Assert the parked worker exits via `ctx.Done()` and the reconcile call returns, proving the send path uses `select` with cancellation instead of a plain blocking send.

**Key Test Files**
- `shard_reconciler_test.go` in the same package as the reconciler, containing the full concurrency/error suite plus local fakes such as `fakeShardRepo`, `fakeWorkerHooks`, and small channel-based barriers.
- If the repository helper that previously used `context.Background()` is independently callable, add one focused companion test in its existing `_test.go` file; otherwise keep that propagation assertion in `shard_reconciler_test.go` and avoid spreading the harness.

**Validation Commands**
- `go test ./... -run 'TestShardReconciler_(LimitsConcurrentWorkersTo16|CancelsSiblingWorkersAndRetainsFatalDBCause|PreservesContextCanceled|PreservesContextDeadlineExceeded|PropagatesSharedContextToRepositoryAndBlockedRepoCallsExitOnCancel|BlockedResultSendExitsOnCancel)$'`
- `go test -race ./... -run 'TestShardReconciler_'`
- `make test`
- `make test-race`

**Observed Result**
Not run. This task required a no-edit deliverable, so the proposed tests were not added to the workspace and there was nothing new to validate.

**Design Escalations**
- These tests assume a fatal worker failure should surface the original wrapped worker or repository cause, while preserving `errors.Is` for `context.Canceled` and `context.DeadlineExceeded`. If the service intentionally normalizes fatal errors into a stable domain sentinel, that contract needs to be specified before final assertions are fixed.
- The blocked-send regression should be tested only with a real cancellation-aware seam. If the send path is too inlined to hold deterministically at the boundary, add a minimal package-private helper or hook before writing the test rather than falling back to sleep-based timing.

**Residual Risks**
- This unit suite proves coordinator behavior, cancellation propagation, and wrapped-error semantics, but it does not prove that the real database driver aborts promptly on cancel. If that path is critical, add one integration-tagged cancellation test against the real repository boundary.
- The suite above focuses on shutdown, concurrency limit, and error semantics. If shard enumeration, deduplication, or retry classification also changed, those need separate obligation-driven tests.
