**Objective**
Investigate and fix a flaky shutdown/drain bug in the Go service template without papering over it with a longer timeout.

**User Intent And Context**
The user reports a rare flake around shutdown or draining. The failure appears intermittent: sometimes `context canceled` is swallowed, or a worker does not stop and the test hangs. They explicitly do not want a timeout increase. The desired approach is to first find where the shutdown path breaks, likely in bootstrap or readiness/health logic, then fix it carefully. They also want the race or integration angle checked if that is the right proof path.

**Confirmed Signals And Exact Identifiers**
- `flaky shutdown`
- `shutdown / drain`
- `context canceled`
- `worker does not stop`
- `test hangs`
- `bootstrap`
- `health/readiness`
- `race`
- `integration`
- Non-goal: “Don’t just increase timeouts.”

**Relevant Repository Context**
This repo’s durable lifecycle surface is in `cmd/service/internal/bootstrap/`, and the context-selection map says shutdown/readiness issues should start there, with `internal/config/` and `internal/app/health/` as nearby surfaces.
Validation signals from the repo profile and context map:
- `make test-race` is the most relevant check when shutdown or concurrency is implicated.
- `make test-integration` is relevant if the lifecycle behavior crosses integration boundaries.
- A focused `go test` on the affected package(s) should come first.

**Inspect First**
Start with:
- `cmd/service/internal/bootstrap/`
- `internal/config/`
- `internal/app/health/`
- Nearby shutdown or lifecycle tests, plus any integration tests under `test/` that exercise startup/shutdown or readiness

If the failure path is still ambiguous, inspect the smallest related test surface before widening the search.

**Requested Change / Problem Statement**
Reproduce and diagnose the shutdown flake, identify the exact point where shutdown/drain semantics fail, and fix the bug so the worker stops cleanly and `context canceled` is handled correctly. Prefer a root-cause fix in the lifecycle wiring over any timing-based workaround.

**Constraints / Preferences / Non-goals**
- Do not “fix” this by increasing timeouts.
- Preserve the shutdown contract; do not hide the cancellation signal.
- If the bug is in bootstrap, readiness, or health gating, address it there rather than only patching a downstream symptom.
- Use race or integration testing only if it meaningfully proves the fix.

**Acceptance Criteria / Expected Outcome**
- The flaky hang no longer reproduces under the relevant shutdown path.
- `context canceled` is handled in the intended way and is not silently lost.
- The worker stops reliably during shutdown/drain.
- The fix is backed by the smallest relevant test proof, ideally including a focused `go test` and, if appropriate, `make test-race` or a targeted integration test.