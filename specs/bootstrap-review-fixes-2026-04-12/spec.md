# Bootstrap Review Fixes Spec

## Context

The previous review of `/Users/daniil/Projects/Opensource/go-service-template-rest/cmd/service/internal/bootstrap` accepted four findings:

- Mongo degraded startup logs degraded mode but does not set the matching `startup_dependency_status` metric.
- Network policy parsing currently reads `NETWORK_*` directly in bootstrap, which is not documented as a deliberate exception to the normal typed `internal/config` snapshot.
- `networkPolicyErrorLabels` lives in production code while behaving like a test-only classifier.
- `dependencyInitFailure` contains a context-cancellation branch that returns the same error as the default branch.

The goal of this task is to prepare the correct implementation approach for a later implementation session, not to change code now.

## Scope / Non-goals

In scope:

- Keep dependency degraded-mode metrics and logs consistent for Redis and Mongo bootstrap admission paths.
- Make the `NETWORK_*` source-of-truth story explicit and fail-closed.
- Resolve the production/test ownership of network policy error labels.
- Remove misleading control flow from `dependencyInitFailure`.
- Add targeted tests that prove the intended behavior.

Out of scope:

- Adding Redis or Mongo runtime adapters.
- Moving Redis/Mongo guard-only extension behavior into feature semantics.
- Broad rewrite of bootstrap lifecycle orchestration.
- Changing public HTTP API behavior.
- Changing persisted data, migrations, or generated code.

## Constraints

- Bootstrap remains the service composition root and owns startup/shutdown flow, dependency admission, and runtime policy.
- `internal/config` remains the source of truth for ordinary application runtime config from YAML, `APP__...`, and loader flags.
- `NETWORK_*` must remain fail-closed and operator-controlled. A missing public ingress declaration must still be distinguishable from an explicit private-ingress assertion.
- Metric labels must remain low cardinality.
- The implementation must keep existing error wrapping with `config.ErrDependencyInit` where startup dependency admission currently relies on it.
- No implementation should depend on exact sleep timing or wall-clock timing beyond existing tests.

## Decisions

1. Do not migrate `NETWORK_*` into ordinary typed `internal/config.Config` in this fix.

   Rationale: the network policy variables are operator admission controls, not normal application feature config. They include public-ingress declaration semantics and break-glass exception metadata where explicit presence matters. Moving them into normal YAML/`APP__...` config would either lose the "missing vs explicit false" distinction or require a larger config-shape change. The correct minimal fix is to document `NETWORK_*` as a deliberate operational network-policy channel with source, precedence, fail-closed behavior, and example keys.

2. Document the `NETWORK_*` channel in `docs/configuration-source-policy.md` and make it discoverable from env examples.

   The docs must state that `NETWORK_*` is read directly by bootstrap after the typed config snapshot is built, is not controlled by YAML overlays or `APP__...`, and has no precedence chain beyond process environment values. `env/.env.example` should include commented examples only, so local/dev copies do not silently assert production ingress policy.

3. Keep `networkPolicyErrorLabels` production-owned only by making production code consume it.

   The preferred implementation is to use the helper in the `bootstrapNetworkPolicyStage` configuration-error path to add structured low-cardinality log fields such as `policy.class` and `reason.class`. If implementation finds that production logging should not carry those labels, the fallback is to move the helper into the test file and stop presenting it as a runtime classification seam. Do not leave the helper production-defined but test-only.

4. Add a same-package degraded dependency startup helper.

   Use one helper for the non-aborting degraded dependency paths that both:
   - sets `startup_dependency_status` for the effective degraded/feature-off mode to `true`, and
   - emits the existing `startup_dependency_degraded` warning with the same component, operation, dependency, and mode fields.

   Redis cache degraded startup and Mongo degraded startup must both use this helper. Abort paths caused by context cancellation, deadline exhaustion, or low remaining startup budget must keep using the rejection path and return an error.

5. Simplify `dependencyInitFailure` by removing the redundant context-error branch unless implementation intentionally gives context cancellation/deadline a different error message or classification.

## Open Questions / Assumptions

- [assumption] `NETWORK_*` staying outside typed config is acceptable because repository docs already describe public ingress as an operational network admission rule, and this task will make that exception explicit in the configuration policy.
- [assumption] `startup_dependency_status{dep="mongo",mode="degraded_read_only_or_stale"} 1` is the desired metric for Mongo degraded-but-serving startup, matching the Redis cache `feature_off` degraded behavior.
- [assumption] No separate observability metric taxonomy change is required; this task only makes existing logs and gauge updates consistent.

## Plan Summary / Link

Implementation plan: `plan.md`
Task ledger: `tasks.md`
Technical design entrypoint: `design/overview.md`

## Validation

Required proof after implementation:

- `go test ./cmd/service/internal/bootstrap`
- `go test ./internal/config ./cmd/service/internal/bootstrap`
- `go vet ./cmd/service/internal/bootstrap`
- `gofmt -l` on touched Go files returns no files.

Targeted behavioral proof:

- Mongo degraded-but-serving startup emits `startup_dependency_status{dep="mongo",mode="degraded_read_only_or_stale"} 1`.
- Redis cache degraded path still emits `startup_dependency_status{dep="redis",mode="feature_off"} 1`.
- Network policy configuration errors include the original parse/config cause and the chosen structured classification path is production-owned.
- `dependencyInitFailure` still wraps `config.ErrDependencyInit` and preserves wrapped causes.

## Outcome

T001-T008 implemented on 2026-04-12. Verification passed for the planned bootstrap/config check set.
