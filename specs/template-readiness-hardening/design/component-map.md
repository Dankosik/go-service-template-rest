# Component Map

## Makefile / OpenAPI Runtime Contract Guard

Affected:
- `Makefile`
- `internal/infra/http/openapi_contract_test.go`

Design:
- Prefer changing the test name, not the Makefile target.
- Keep `openapi-runtime-contract-check` selecting `^TestOpenAPIRuntimeContract`.
- Rename the security-decision guard into that naming family.
- If the implementation broadens the Makefile selector instead, it must update `docs/build-test-and-development-commands.md` and justify why broader selection is better.

Stable:
- `openapi-check` remains the composite OpenAPI proof target.
- `internal/api/openapi.gen.go` remains generated and untouched by hand.

## Protected Endpoint Guidance

Affected:
- `internal/api/README.md`
- `internal/infra/http/openapi_contract_test.go`

Design:
- The README recipe should show the path:
  1. OpenAPI contract and security decision,
  2. app use-case behavior in `internal/app/<feature>`,
  3. generation,
  4. handler mapping in `internal/infra/http`,
  5. scoped auth middleware or explicitly designed equivalent for protected operations,
  6. contract/policy tests.
- Avoid making handlers the business-logic owner.
- Avoid broad root middleware that accidentally protects health/metrics or public sample routes.
- Tighten the protected-operation 401/403 Problem-response helper to require the canonical Problem schema.

Stable:
- No new auth package or middleware implementation is introduced in this task.
- Normal API routes stay generated; no manual `/api/...` chi routes.

## Ping History Sample Limit

Affected:
- `internal/infra/postgres/ping_history_repository.go`
- `internal/infra/postgres/ping_history_repository_test.go`

Design:
- Add a package-local sample max limit constant.
- Validate `limit > 0` and `limit <= max`.
- Reuse the validation in both `ListRecent` and `createAndListRecentInTx`.
- Add focused tests proving over-limit rejection and error classification.

Stable:
- No migration changes.
- No sqlc query changes.
- `ping_history` remains a sample repository, not production ping behavior.

## Redis Readiness Policy

Affected:
- `internal/config/types.go` or a nearby config-owned file.
- `internal/config/validate.go`
- `internal/config/config_test.go`
- `cmd/service/internal/bootstrap/startup_probe_helpers.go`
- `cmd/service/internal/bootstrap/startup_dependencies.go`
- bootstrap tests as needed.

Design:
- Add config-owned Redis mode constants/methods.
- Use config-owned `StoreMode` / readiness predicate in validation and bootstrap.
- Remove bootstrap's independent mode normalization path when it becomes redundant.
- Keep bootstrap telemetry labels and dependency criticality values in bootstrap because they are startup-runtime concerns.

Stable:
- Redis remains a guard-only extension stub; no Redis adapter is introduced.
- Config remains the owner of validated runtime snapshot policy.

## Documentation And Placement Discoverability

Affected:
- `README.md`
- `CONTRIBUTING.md`
- `docs/repo-architecture.md`
- `docs/project-structure-and-module-organization.md`
- `docs/build-test-and-development-commands.md`
- `test/README.md`
- `research/coverage-audit.md`

Design:
- Keep `docs/project-structure-and-module-organization.md` as the canonical placement guide.
- Add links or compact recipes where authors enter the repo, but do not duplicate the full placement guide.
- Use `research/coverage-audit.md` as the checklist for planned, deferred, already-covered, and rejected research points.
- Keep deferred maintainability cleanups out of implementation unless explicitly reopened.

Stable:
- README remains a high-level entrypoint, not a full architecture manual.
- Command docs remain a command reference, not the owner of placement rules.
