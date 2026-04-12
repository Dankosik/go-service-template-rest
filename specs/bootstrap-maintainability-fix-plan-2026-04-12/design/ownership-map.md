# Ownership Map

## Source Of Truth

| Concern | Owner | Consumers |
| --- | --- | --- |
| Startup rejection reason metric | `internal/infra/telemetry` | `cmd/service/internal/bootstrap` |
| Config failure type labels | `internal/config.ErrorType` | `cmd/service/internal/bootstrap`, telemetry labels |
| Readiness participation predicates | `internal/config.Config` methods | config validation and bootstrap probe inclusion |
| Dependency probe labels | `cmd/service/internal/bootstrap.startupDependencyProbeLabels` | bootstrap probe rejection logging/metrics/spans |
| Startup/shutdown lifecycle sequence | `cmd/service/internal/bootstrap.serveHTTPRuntime` | `cmd/service/internal/bootstrap.Run` and tests |
| Network policy declaration semantics | `cmd/service/internal/bootstrap` network policy parsing/enforcement | bootstrap startup policy stage |

## Dependency Direction

- `cmd/service/internal/bootstrap` may depend on `internal/config`, `internal/app/health`, `internal/infra/http`, `internal/infra/postgres`, and `internal/infra/telemetry` because it is the composition root.
- `internal/config` must not import bootstrap or telemetry.
- `internal/infra/telemetry` must not import bootstrap or config.
- New helpers should stay in the owning package; do not create `common`, `util`, or cross-package helper buckets.

## Ownership Decisions

- Startup rejection metric naming and normalization belong to telemetry because it owns shared instruments.
- The decision to increment startup rejection reasons belongs to bootstrap because it owns process lifecycle failure classification.
- Readiness predicate semantics belong to config because validation and bootstrap must consume the same config policy.
- Dependency label derivation belongs to bootstrap because labels are local to startup dependency orchestration.

## Rejected Ownership Moves

- Do not move bootstrap lifecycle helpers into `internal/infra/http`; HTTP runtime construction remains an adapter consumed by bootstrap.
- Do not move network policy env parsing into `internal/config`; repository docs define `NETWORK_*` as a bootstrap-owned operator policy channel outside normal config precedence.
- Do not create a shared telemetry-policy package; the affected policy is local to telemetry instruments plus bootstrap call sites.
