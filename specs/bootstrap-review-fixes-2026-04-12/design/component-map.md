# Component Map

## Affected Components

`cmd/service/internal/bootstrap/startup_dependencies.go`

- Add one local helper for degraded dependency startup status/logging.
- Use it in the Redis cache degraded path and the Mongo degraded path.
- Remove the redundant context-error branch in `dependencyInitFailure`.
- Keep abort/rejection behavior unchanged for context cancellation, deadline exhaustion, and low startup budget.

`cmd/service/internal/bootstrap/startup_dependencies_additional_test.go`

- Add or extend coverage for Mongo degraded-but-serving startup metric output.
- Keep existing Redis and abort-path tests intact.

`cmd/service/internal/bootstrap/network_policy_parsing.go`

- Keep parsing logic local to bootstrap for `NETWORK_*`.
- Keep `networkPolicyErrorLabels` only if production code consumes it after this change.

`cmd/service/internal/bootstrap/startup_bootstrap.go`

- Preferred: use `networkPolicyErrorLabels` in the `loadNetworkPolicyFromEnv` error path to add structured low-cardinality log fields for policy class and reason class.
- Avoid changing enforcement behavior or error wrapping.

`cmd/service/internal/bootstrap/startup_common.go`

- Optional change surface if implementation chooses to extend `rejectStartupForPolicyViolation` with variadic structured fields. Keep existing call sites source-compatible.

`cmd/service/internal/bootstrap/startup_common_additional_test.go`

- Add or extend log assertion coverage only if structured policy-class fields are wired into production logging.

`docs/configuration-source-policy.md`

- Add an operational network-policy channel section that explains why `NETWORK_*` is outside `APP__...` config, when bootstrap reads it, fail-closed declaration semantics, and example variables.

`docs/repo-architecture.md` and `docs/project-structure-and-module-organization.md`

- Update only if needed to link back to the new configuration-source-policy section or to mention egress allowlist policy alongside existing ingress policy notes.

`env/.env.example`

- Add commented `NETWORK_*` examples only. Do not add uncommented defaults that would silently declare public ingress state.

## Stable Components

- `internal/app/**`: unchanged.
- `internal/infra/http/**`: unchanged.
- `internal/infra/postgres/**`: unchanged.
- `internal/infra/telemetry/metrics.go`: unchanged unless tests reveal the existing metric helper cannot express the intended gauge state.
- `internal/config/**`: no typed config migration in this fix.
- API contracts and generated code: unchanged.
