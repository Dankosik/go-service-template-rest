# Config Readiness Policy Follow-up Workflow Plan

Task: implement the `internal/config` review follow-up.

Execution shape: lightweight local implementation and validation pass.
Current phase: done.
Phase status: complete.
Session boundary reached: yes.
Ready for next session: no.
Next session starts with: n/a; task is closed unless a future reopen condition triggers.

## Scope

In scope:
- Fix the mismatch between `internal/config` readiness-budget validation and runtime Postgres readiness behavior.
- Clarify the Redis/Mongo extension-key policy so config does not look like a hidden adapter owner.
- Improve readability around config-file policy selection.
- Remove small local decision drift in Redis mode validation.
- Capture the optional Mongo probe-address branch simplification as an opportunistic cleanup if `internal/config/validate.go` is already being edited.

Out of scope:
- Implementing Redis or Mongo adapters.
- Removing existing Redis/Mongo config keys in this pass.
- Changing REST API, schema, migrations, OpenAPI generation, telemetry metric names, or bootstrap dependency criticality.

## Artifact Status

- `workflow-plans/planning.md`: complete.
- `spec.md`: approved with lightweight local clarification waiver.
- `design/overview.md`: approved.
- `design/component-map.md`: approved.
- `design/sequence.md`: approved.
- `design/ownership-map.md`: approved.
- `plan.md`: approved.
- `tasks.md`: approved.
- `workflow-plans/implementation-phase-1.md`: complete.
- `workflow-plans/validation-phase-1.md`: complete.
- `test-plan.md`: not expected; validation obligations fit in `plan.md` and `tasks.md`.
- `rollout.md`: not expected; changes are local, backward-compatible, and do not require migration or rollout choreography.

## Workflow Notes

Research mode: local, using the previous read-only review fan-out as input evidence.

Lightweight local waiver:
- The user requested a pre-implementation file bundle and a later separate implementation session.
- The task is bounded to review findings with no new product/API behavior.
- Prior subagent review already covered idiomatic Go, readability, and design angles.
- No new subagent authorization was given for this follow-up, so clarification and adequacy checks are handled locally and recorded here.

Implementation readiness: PASS.

Accepted risks:
- Redis/Mongo reserved extension keys remain in the config contract for compatibility. If the desired product decision is to remove them instead, reopen `spec.md` before implementation.
- Runtime Postgres readiness will be bounded in bootstrap rather than by changing `postgres.Pool.Check` semantics. If a later task wants every direct `Pool.Check` caller to carry its own configured timeout, reopen technical design.

## Reopen Rules

Reopen the relevant earlier phase if:
- Fixing Postgres readiness requires changing `internal/app/health.Service` sequencing semantics.
- Redis/Mongo keys are no longer intended to remain as reserved extension API.
- A change requires new API, schema, migration, or adapter behavior not described in this bundle.

## Validation And Closeout

Fresh evidence from the implementation/validation session:
- `go test ./cmd/service/internal/bootstrap -run 'Test.*Postgres.*Readiness|TestRunDependencyProbe|TestInitStartupDependenciesAllDisabled' -count=1`: passed.
- `go test ./internal/config -run 'Test(LocalAllowsSymlinkConfig|ConfigFileWithoutEnvironmentHintFailsClosed|NonLocalRejectsSymlinkConfig|RedisStoreGuard|RedisModePolicyHelpers|MongoProbeAddress)' -count=1`: passed.
- `go test ./internal/config ./cmd/service/internal/bootstrap ./internal/infra/postgres -count=1`: passed.

Completion marker:
- All `tasks.md` items are complete.
- `spec.md` `Validation` and `Outcome` reflect the implementation evidence.
- No Redis/Mongo adapter package was added; `internal/app/health.Service` and global `postgres.Pool.Check` semantics were not changed.
