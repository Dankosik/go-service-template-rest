# Flaky Reproduction Controls For Go

## Purpose
Shrink intermittent Go failures into a repeatable signal without masking the real bug.

## Repetition And Order Controls

Use the narrowest package and test first:

```bash
go test ./path/to/pkg -run '^TestName$' -count=100
go test ./path/to/pkg -run '^TestName$' -shuffle=on -count=100
go test ./path/to/pkg -run '^TestName$' -race -count=50
go test ./path/to/pkg -run '^TestName$' -cpu=1,4
```

Notes:
- `-count` helps estimate frequency and confirms whether the signal is real.
- `-shuffle=on` surfaces order dependence only when multiple tests or subtests are actually in scope. Capture the printed seed and reuse it with `-shuffle=<seed>` once a failing order is found.
- If you narrow to one top-level test with `-run '^TestName$'`, `-shuffle` often stops being meaningful for package-level order. In that case use a wider pattern or the full package for the order experiment, and keep the narrow command for local race or lifecycle proof.
- `-race` is essential when shared-state or goroutine ownership is even mildly suspicious.
- `-cpu=1,4` is useful when scheduler sensitivity or accidental parallelism is part of the failure mode.

## Common Flake Classes

### Sleep-Based Coordination
- `time.Sleep` guesses readiness instead of proving it
- background work sometimes finishes before the sleep and sometimes after it
- fix by waiting on a condition, event, or channel, not by inflating the delay

### Shared State Leakage
- env vars, temp dirs, ports, global singletons, caches, or registries persist between tests
- order changes the visible state
- fix by isolating state and cleaning up with `t.Cleanup`

### Time And Randomness
- test uses wall clock, implicit `time.Now`, default random seed, or real timers
- slower CI or fast local machines move behavior across thresholds
- fix with fake clocks, explicit seeds, and deterministic timer control when possible

### Goroutine Lifecycle
- worker goroutines outlive the test
- channel senders or receivers remain blocked across test boundaries
- `t.Parallel` overlaps cleanup and background work
- fix by owning shutdown and waiting for completion explicitly

### External Dependency Leakage
- test depends on real network, DB, cache, or clock skew
- local pass is luck; CI fails under latency or ordering variation
- fix by isolating the dependency or at least making its timing and cleanup explicit

## Good Artifact Hygiene
For each flake run, record:
- exact command
- package and test name
- `-count`, `-shuffle`, `-race`, `-cpu`, and timeout settings
- failing seed or order if known
- the first distinct failure symptom, not every duplicate line

## Escalation Clues
Escalate beyond test-only repair when:
- the flake is really a production concurrency bug that the test happened to expose
- the only plausible fix changes timeout, retry, or API contract behavior
- the test cannot be made deterministic without first changing the production ownership model
