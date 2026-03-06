## Context

Recent bootstrap changes introduced two production-risk regressions in `cmd/service/internal/bootstrap`:
- process startup now blocks on an external `/health/ready` request before the service considers itself ready;
- shutdown adds a fixed readiness propagation sleep on top of the configured `http.shutdown_timeout`.

The goal of this fix is to restore the repository's documented startup/shutdown contract without widening scope beyond bootstrap lifecycle behavior and directly related tests.

## Scope / Non-goals

In scope:
- bootstrap startup admission and readiness state transitions;
- shutdown budget accounting and pre-ready shutdown behavior;
- targeted test coverage for the corrected lifecycle paths.

Non-goals:
- redesigning health endpoint semantics;
- changing deployment policy values in `railway.toml` or config defaults;
- broad refactors of dependency probing or network policy rules beyond what the lifecycle fix requires.

## Constraints

- Startup must not depend on an external readiness poll to remain alive.
- Readiness must still reflect actual core-traffic readiness and fail closed during shutdown.
- Shutdown must stay within the existing configured timeout budget.
- Changes stay inside the bootstrap package and directly impacted tests unless a narrow seam is required for deterministic verification.

## Decisions

- Startup admission success will be recorded from an internal bootstrap readiness check after the HTTP listener is up, not from an external `/health/ready` request.
- Runtime `/health/ready` requests will continue to evaluate readiness, but they will no longer own startup liveness or admission success.
- Shutdown propagation delay will consume the existing shutdown timeout budget instead of extending total shutdown time.
- If shutdown begins before the instance was admitted ready, propagation delay is skipped and shutdown starts immediately.

## Open Questions / Assumptions

- Assumption: the existing startup dependency initialization remains the source of truth for dependency readiness; the internal post-listen readiness check is a confirmation step, not a new dependency policy.

## Implementation Plan

1. Introduce a bootstrap-local readiness admission helper that performs the internal post-listen readiness check and records startup success exactly once.
   Completion criteria: startup no longer waits for an external readiness request; success metrics/span/telemetry are driven internally.

2. Simplify the runtime readiness hooks so `/health/ready` continues enforcing ingress/runtime checks without owning process lifecycle.
   Completion criteria: readiness still returns `503` on runtime ingress violations and during drain, but startup survival no longer depends on that path.

3. Rework shutdown budgeting so readiness propagation delay is capped by the configured shutdown timeout and skipped when the instance never became ready.
   Completion criteria: total shutdown path stays within `http.shutdown_timeout`, and pre-ready failures do not sleep before shutdown.

4. Extend tests for the corrected lifecycle behavior.
   Completion criteria: targeted tests cover internal startup admission, bounded shutdown delay, and the relevant non-ready shutdown path.

## Validation

Planned commands:
- `go test ./cmd/service/internal/bootstrap/...`
- `go test ./cmd/service -run 'Test(DrainAndShutdown|ServeHTTPRuntime|Admission)' -count=1`

## Outcome

Planned; implementation not started in this artifact yet.
