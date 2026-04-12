# Component Map

## Documentation Surfaces

- `README.md`: add or link a short first-production-feature checklist from the human and agent quickstart areas.
- `docs/project-structure-and-module-organization.md`: strengthen the "Where to Put New Code" section with the production-shaped path, bootstrap proof shape, feature telemetry placement, sample replacement warning, protected-endpoint guidance, integration trigger matrix, and config test placement row.
- `docs/repo-architecture.md`: preserve as the stable architecture baseline; update only if the current extension seams need a compact cross-link to the new checklist.
- `docs/configuration-source-policy.md`: clarify Redis/Mongo guard-only semantics and the rule that cache/store behavior belongs to a future feature-owned adapter.
- `internal/api/README.md`: clarify protected-operation wiring expectations without adding auth design.
- `test/README.md`: add trigger guidance for endpoint plus real persistence plus bootstrap wiring scenarios.

## Code And Test Surfaces

- `internal/infra/http/router_test.go`: add a route-tree guard that fails if manual root routes bypass the documented manual-route helper or introduce `/api/...` manual registrations.
- `internal/infra/http/router.go`: canonicalize `Allow` header emission if route-policy tests are being touched.
- `cmd/service/internal/bootstrap/startup_common.go`: add the missing error attribute to dependency-probe startup rejection logs.
- `cmd/service/internal/bootstrap/*_test.go`: add or update focused coverage for the dependency-probe rejection log envelope.
- `scripts/ci/required-guardrails-check.sh`: add an app/domain import boundary guardrail if it can be expressed with existing shell dependencies.

## Stable Surfaces

- `api/openapi/service.yaml`: no selected change.
- `internal/api/openapi.gen.go`: no selected hand edit.
- `env/migrations/*`: no selected change.
- `internal/infra/postgres/sqlcgen/*`: no selected hand edit.
- `internal/domain`: stays empty unless a real shared contract appears in a future feature.
