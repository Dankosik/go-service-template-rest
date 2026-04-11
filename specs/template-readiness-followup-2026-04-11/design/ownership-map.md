# Ownership Map

## Source-Of-Truth Rules

| Concern | Source of truth | Derived / consuming surfaces |
| --- | --- | --- |
| REST operations and security decisions | `api/openapi/service.yaml` | `internal/api`, `internal/infra/http`, OpenAPI tests |
| Generated HTTP wrapper behavior | generated `internal/api` plus local `internal/infra/http` options | Router construction and HTTP contract tests |
| Problem response policy | `internal/infra/http/problem.go` and router error handlers | OpenAPI response declarations, HTTP tests |
| Runtime config | `internal/config` defaults/types/snapshot/validation | Bootstrap, docs, `.env` examples |
| Public ingress declaration | `cmd/service/internal/bootstrap` network policy parsing/enforcement | Config docs and startup tests |
| Dependency admission | `cmd/service/internal/bootstrap/startup_dependencies.go` | Health probes, metrics labels, docs |
| Readiness probe contract | `internal/app/health.Probe` | Bootstrap dependency probes |
| Panic/request log redaction | `internal/infra/http` middleware/router | HTTP tests and docs |
| OTLP header parsing | `internal/infra/telemetry` | Config policy docs and telemetry tests |
| Persistence extension recipe | docs plus `internal/infra/postgres` examples | Future app-owned ports and bootstrap wiring |
| Integration/migration validation | `Makefile`, `docs/build-test-and-development-commands.md`, `test/README.md` | Future feature validation plans |

## Dependency Direction

- `internal/app` must not import `internal/infra/http`, `internal/infra/postgres/sqlcgen`, pgx, or transport details.
- `internal/infra/http` may import generated `internal/api` and app services.
- `internal/infra/postgres` may import sqlcgen and pgx; it maps into app-facing records or app-owned ports.
- `cmd/service/internal/bootstrap` may import concrete infra packages and app packages because it is the composition root.
- `internal/config` should not import bootstrap or infra packages except where validation already depends on parse-only library contracts.

## Helper Extraction Rules

- Package-local constants for labels, config keys, or stage names are allowed when they protect a local source of truth.
- Package-local test reflection is allowed for config key drift detection.
- Cross-package shared constants are not allowed unless a real runtime contract crosses the package boundary.
- No `internal/common`, `internal/util`, generic transaction helper, generic repository interface, or generic dependency manager should be introduced in this task.

## Security Ownership

- This task defines security decision recording, not runtime auth.
- Endpoint authn/authz belongs to a later feature once identity, tenant, and object-authorization decisions exist.
- Browser session/CSRF runtime belongs to a later browser-facing feature.
- `/metrics` exposure belongs to deployment/security design if it needs to be internet-facing.
