## Context
- Review of `cmd/service/internal/bootstrap` found lifecycle, telemetry, and network-policy issues that can produce false rollout signals or fail-open behavior during startup/shutdown.
- Follow-up review found remaining gaps after the previous hardening pass:
  - pre-readiness startup-budget expiry still exits with success after shutdown;
  - dependency and network-policy failure paths still discard actionable root-cause detail;
  - Redis `store` mode is still labeled readiness-critical without participating in runtime readiness.
- Existing approved behavior is defined primarily by:
  - `specs/koanf-config-migration/55-reliability-and-resilience.md`
  - `specs/railway-deployment-resilience/15-domain-invariants-and-acceptance.md`
  - `specs/railway-deployment-resilience/50-security-observability-devops.md`
  - `specs/railway-deployment-resilience/55-reliability-and-resilience.md`

## Scope / Non-goals
- In scope:
  - startup cancellation and degradation handling;
  - admission/rollback/drift telemetry semantics inside bootstrap;
  - network-policy matcher and runtime enforcement defects in bootstrap;
  - startup error propagation and diagnosability fixes for dependency/network-policy failures;
  - readiness-critical dependency contract alignment for Redis `store` mode;
  - targeted tests for the corrected behavior.
- Non-goals:
  - introducing a new external rollout controller;
  - redesigning the full deploy-policy source-of-truth model outside bootstrap;
  - changing unrelated config loading or HTTP router behavior.

## Constraints
- Keep fixes scoped to `cmd/service/internal/bootstrap` and directly affected tests unless a narrow supporting change is required.
- Preserve existing public API behavior.
- Do not invent platform-side evidence; runtime telemetry must reflect only what bootstrap can actually know.
- Startup cancellation must remain observable through recognizable context errors.

## Decisions
- Admission success will be emitted only after the first real readiness success, not on `net.Listen`.
- Startup budget remains authoritative until the first readiness success; if it expires beforehand, bootstrap must begin shutdown and readiness must stay `503`.
- Any pre-readiness startup failure must remain the returned process error even if graceful shutdown succeeds afterward.
- Once shutdown or another terminal failure path starts, in-flight readiness handlers must not be able to return a new success admission.
- Bootstrap failure paths will stop emitting synthetic rollback execution/postcheck events; rollback telemetry must remain owned by code paths with real rollback authority.
- Config unknown-key warnings remain config-load telemetry only; they are not a substitute for Railway drift detection/reconciliation telemetry.
- Degradable dependency probes must still fail startup on context cancellation, deadline exhaustion, or exhausted startup budget.
- Dependency and network-policy startup failures must preserve original wrapped causes for operator diagnostics and programmatic inspection.
- Wildcard host rules like `*.example.com` will match subdomains only, not the apex domain.
- Single-label hosts will no longer be implicitly treated as private by default.
- Redis `store` mode will participate in runtime readiness via a lightweight probe; optional Redis/Mongo runtime probes remain feature-flag controlled.

## Open Questions / Assumptions
- Assumption: bootstrap may record readiness-gated admission based on the local `/health/ready` success path because no separate in-process deploy controller exists in this scope.
- Assumption: ingress-exception expiry can be enforced at runtime via readiness degradation without expanding the external API surface.

## Implementation Plan
1. Adjust deploy telemetry recorder semantics so admission success is readiness-driven and failure does not fabricate rollback evidence.
   Completion criteria:
   - success signal is emitted from a real readiness success path;
   - failure paths no longer emit synthetic rollback execution/postchecks.
2. Fix startup lifecycle handling for cancellation and shutdown ordering.
   Completion criteria:
   - cache/degraded dependencies do not ignore canceled or expired startup contexts;
   - server startup does not report success after cancellation races or startup-budget expiry before readiness;
   - shutdown keeps an explicit drain propagation window before `Shutdown`.
3. Tighten network-policy enforcement and add regression coverage.
   Completion criteria:
   - wildcard and single-label host matching follow fail-closed semantics;
   - runtime ingress exception expiry is re-checked on readiness path;
   - tests cover the corrected cases;
   - invalid policy metadata surfaces a concrete cause in returned errors and logs.
4. Preserve actionable root causes in startup dependency failures.
   Completion criteria:
   - wrapped dependency errors retain the underlying dial/connect/healthcheck cause;
   - callers can still use `errors.Is(err, config.ErrDependencyInit)`.
5. Align readiness-critical dependency behavior with runtime readiness contract.
   Completion criteria:
   - Redis `store` mode participates in runtime readiness through an actual runtime probe;
   - metrics/test expectations match the actual readiness contract.
6. Run focused validation and record outcomes.
   Completion criteria:
   - package tests pass;
   - race run for bootstrap package passes.

## Validation
- `go test ./cmd/service/internal/bootstrap -count=1`
- `go test -race ./cmd/service/internal/bootstrap -count=1`

## Outcome
- Completed targeted bootstrap fixes for:
  - pre-readiness startup-budget expiry now returns a process error even after graceful shutdown;
  - dependency init failures preserve wrapped root causes for diagnostics and `errors.Is`/`errors.As`;
  - invalid network-policy metadata now preserves the original cause in both returned errors and structured logs;
  - Redis `store` mode now contributes a runtime readiness probe, while optional Redis/Mongo runtime probes stay feature-flag driven;
  - regression coverage for startup-deadline failure, dependency error wrapping, Redis store readiness probing, and network-policy diagnostics.
