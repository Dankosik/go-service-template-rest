# Design Overview

## Goal

Prepare the repository for future production business-code integration by making the remaining template-readiness rules explicit, local, and testable.

The design keeps the existing architecture:

- `cmd/service/internal/bootstrap` remains the composition root.
- `internal/app` remains transport- and driver-agnostic.
- `internal/infra/http` owns HTTP mapping, middleware, generated-route integration, and Problem responses.
- `api/openapi/service.yaml` remains the REST contract source of truth.
- `internal/config` owns config defaults, snapshot, validation, and source policy.
- `internal/infra/postgres` remains the sqlc/Postgres adapter surface.
- `internal/infra/telemetry` owns tracing and metrics setup.

## Chosen Approach

Use small local hardening changes rather than a new architecture layer:

- Add local HTTP options/error-handler helpers instead of a transport framework wrapper.
- Add OpenAPI security-decision checks instead of placeholder auth.
- Add network-policy declaration checks in bootstrap instead of deployment-specific platform code.
- Add config drift tests and package-local vocabulary instead of reflection-driven runtime config mapping.
- Add docs-only recipes for browser security, runtime dependencies, and persistence ports instead of fake abstractions.

## Artifact Index

- `component-map.md`: affected packages and files.
- `sequence.md`: startup, request, config, and validation flows after the changes.
- `ownership-map.md`: source-of-truth and dependency boundaries.
- `contracts/http-security-and-generated-errors.md`: HTTP contract/security marker and generated-error details.
- `../plan.md`: implementation phases.
- `../tasks.md`: executable task ledger.
- `../test-plan.md`: validation strategy.

## Readiness Summary

Implementation may start from Phase 1 in `../plan.md`.

No design decision requires adding business semantics. If implementation starts needing a real identity provider, tenant policy, browser session model, separate metrics listener, or parallel readiness probes, stop and reopen planning.
