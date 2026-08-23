## Summary

- What changed and why?

## Scope

- [ ] API contract changed (`api/openapi/service.yaml`)
- [ ] Runtime behavior changed (`cmd/`, `internal/<feature>/`, `internal/infra/`)
- [ ] CI/CD workflow or quality gates changed (`.github/workflows/`, `Makefile`)
- [ ] Database schema/migrations changed (`migrations/`)

## Test Evidence

- [ ] `make unit-check PKG=./<changed> FILES='...'` (acceptance unit)
- [ ] `make check` (integrated candidate; do not also rerun fmt/lint/test-all)
- [ ] claim-matched focused checks or CI evidence linked before merge
- [ ] `make openapi-check` (when API/runtime contract changed)
- [ ] `ALLOW_HEAVY=1 make test-race` (when concurrency-sensitive code changed)
- [ ] `REQUIRE_DOCKER=1 ALLOW_HEAVY=1 make test-integration` (when integration behavior changed)
- [ ] `make sqlc-check` (when SQL queries or migrations changed)
- [ ] `ALLOW_HEAVY=1 make migration-validate` (when migrations, Docker runtime, or startup changed)

Commands/output summary:

```text
paste concise command output or links to CI evidence
```

## Security Impact

- [ ] No security-sensitive changes
- [ ] Security-sensitive changes included (authn/authz/input validation/secrets/logging)

Notes:

## API/DB/Docs Impact

- [ ] No API/DB/docs impact
- [ ] API changed and OpenAPI updated
- [ ] DB changed and migration validation covered
- [ ] Docs updated (`docs/**` or `README.md`)

## Rollback Notes

- [ ] Not needed (low risk)
- [ ] Required (describe rollback or mitigation path)

Rollback plan:
