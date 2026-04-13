## Summary

- What changed and why?

## Scope

- [ ] API contract changed (`api/openapi/service.yaml`)
- [ ] Runtime behavior changed (`cmd/`, `internal/app/`, `internal/infra/http/`)
- [ ] CI/CD workflow or quality gates changed (`.github/workflows/`, `Makefile`)
- [ ] Database schema/migrations changed (`env/migrations/`)

## Test Evidence

- [ ] `make fmt-check`
- [ ] `make lint`
- [ ] `make test`
- [ ] `make pr-check`, `make check-full`, or CI evidence linked before merge
- [ ] `make test-report` or CI `test-coverage` evidence when coverage changed or risk is non-trivial
- [ ] `make openapi-check` (when API/runtime contract changed)
- [ ] `make test-race` (when concurrency-sensitive code changed)
- [ ] `make test-integration` (when integration behavior changed)
- [ ] `make sqlc-check` (when SQL queries or migrations changed)
- [ ] `make migration-validate` (when migrations changed)
- [ ] `make mocks-drift-check` (when mockgen directives or interfaces changed)
- [ ] `make stringer-drift-check` (when stringer directives or enum values changed)

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
