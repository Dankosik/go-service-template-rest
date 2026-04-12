# Planning Phase

## Scope

Create the pre-code context for fixing the reconciled `internal/infra` review findings. No production or test code changes are allowed in this session.

## Mode

- Research mode: local synthesis from the completed read-only review and repository architecture docs.
- Skills used: `spec-document-designer`, `go-design-spec`, `planning-and-task-breakdown`.
- Subagents: not used in this pass; the preceding review already used read-only fan-out and the user did not request additional agents.

## Decisions Captured

- Remove `resource.WithFromEnv()` rather than adding a new implicit OTEL environment channel.
- Add a tiny neutral `internal/observability/otelconfig` package for OTel config vocabulary; do not create `common` and do not make `internal/config` import `internal/infra/telemetry`.
- Keep SDK-specific sampler and exporter construction in `internal/infra/telemetry`.
- Remove telemetry-owned resource identity fallback defaults and keep those defaults in config.
- Add non-finite sampler-argument validation at the telemetry runtime boundary.
- Make HTTP `Server` nil/zero-value misuse return an inspectable error instead of panicking.
- Replace raw startup dependency status booleans with intent-named telemetry methods.
- Remove the unused transactional workflow from the replaceable `ping_history` fixture.
- Consolidate telemetry init failure reason strings behind telemetry-owned constants while preserving low-cardinality normalization.
- Treat manual root route reason consolidation as a required low-risk cleanup so every review finding has an implementation task.
- Record `research/review-findings-coverage.md` as the finding-to-task coverage audit.

## Completion Marker

Planning is complete when `spec.md`, the required `design/` bundle, `plan.md`, `tasks.md`, `test-plan.md`, and post-code phase-control files exist and name implementation readiness.

## Status

Completed.

## Handoff Rule

Stop here. Implementation starts in a separate session from `workflow-plans/implementation-phase-1.md`.
