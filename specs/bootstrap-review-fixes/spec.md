## Context
- Review of `cmd/service/internal/bootstrap` found lifecycle, telemetry, and network-policy issues that can produce false rollout signals or fail-open behavior during startup/shutdown.
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
- Once shutdown or another terminal failure path starts, in-flight readiness handlers must not be able to return a new success admission.
- Bootstrap failure paths will stop emitting synthetic rollback execution/postcheck events; rollback telemetry must remain owned by code paths with real rollback authority.
- Config unknown-key warnings remain config-load telemetry only; they are not a substitute for Railway drift detection/reconciliation telemetry.
- Degradable dependency probes must still fail startup on context cancellation, deadline exhaustion, or exhausted startup budget.
- Wildcard host rules like `*.example.com` will match subdomains only, not the apex domain.
- Single-label hosts will no longer be implicitly treated as private by default.

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
   - server startup does not report success after cancellation races;
   - shutdown keeps an explicit drain propagation window before `Shutdown`.
3. Tighten network-policy enforcement and add regression coverage.
   Completion criteria:
   - wildcard and single-label host matching follow fail-closed semantics;
   - runtime ingress exception expiry is re-checked on readiness path;
   - tests cover the corrected cases.
4. Run focused validation and record outcomes.
   Completion criteria:
   - package tests pass;
   - race run for bootstrap package passes.

## Validation
- `go test ./cmd/service/internal/bootstrap/...`
- `go test ./internal/infra/http/...`
- `go test -race ./cmd/service/internal/bootstrap/...`
- `go test -race ./internal/infra/http/...`

## Outcome
- Completed targeted bootstrap/http hardening for:
  - readiness-driven admission success;
  - startup-deadline enforcement while waiting for first readiness;
  - no late readiness success after shutdown/terminal failure begins;
  - no synthetic rollback telemetry on bootstrap failure paths;
  - startup cancellation propagation through degradable dependencies;
  - explicit shutdown propagation delay before graceful server shutdown;
  - stricter network-policy wildcard and single-label host handling;
  - runtime ingress-expiry readiness enforcement;
  - regression coverage for fail-before-ready, shutdown delay with canceled parent context, degradable dependency abort branches, and leading-dot host matcher semantics.
