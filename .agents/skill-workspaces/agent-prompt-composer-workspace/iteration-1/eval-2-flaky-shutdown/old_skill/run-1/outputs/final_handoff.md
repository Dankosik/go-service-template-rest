**Objective**  
Investigate and fix a flaky shutdown/drain bug in the service lifecycle, with emphasis on finding the actual breakage point before changing timeouts or masking the failure.

**User Intent And Context**  
The user reports a “rare flake” around shutdown/drain. The symptoms are that `context canceled` may be swallowed, or a worker does not stop and the test hangs. They explicitly do not want a timeout increase as the solution. The requested approach is to first locate where it breaks, likely in bootstrap or the health/readiness path, then fix it carefully and verify whether race or integration coverage is the right proof path.

**Confirmed Signals And Exact Identifiers**  
- `context canceled`
- `shutdown / drain`
- `bootstrap`
- `health/readiness`
- `race`
- `integration`
- `timeout`
- “Don’t just increase timeouts”

**Relevant Repository Context**  
- This repo is a Go REST service template.
- Lifecycle wiring lives under `cmd/service/internal/bootstrap/`.
- Readiness and health logic is likely under `internal/app/health/` and/or `internal/infra/http/`.
- Shutdown/concurrency issues are a good fit for `make test-race`.
- Integration-level lifecycle behavior may require `make test-integration`.

**Inspect First**  
- `cmd/service/internal/bootstrap/`
- `internal/app/health/` `likely`
- `internal/infra/http/` `likely`
- Nearby tests covering shutdown, drain, readiness, or worker lifecycle `likely`

**Requested Change / Problem Statement**  
Find the root cause of the flaky shutdown behavior instead of papering over it. Determine where `context canceled` is being lost or where a worker keeps running during shutdown, then fix the lifecycle handling so the test no longer hangs.

**Constraints / Preferences / Non-goals**  
- Do not solve this by increasing timeouts.
- Prefer a minimal, carefully justified fix over broad lifecycle changes.
- Preserve correct shutdown semantics under cancellation.
- Treat race or integration evidence as part of the investigation if needed.

**Validation / Verification**  
- Run the smallest targeted tests that reproduce or cover shutdown/drain behavior.
- Use `make test-race` if concurrency or cancellation ordering is implicated.
- Use `make test-integration` if the failure crosses process or lifecycle boundaries.
- Confirm the fix by proving the hang no longer occurs and cancellation is handled explicitly.