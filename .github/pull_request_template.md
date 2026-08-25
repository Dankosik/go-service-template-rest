## Summary

- What changed and why?

## Scope

- [ ] API contract changed (`api/openapi/service.yaml`)
- [ ] Runtime behavior changed (`cmd/`, `internal/<feature>/`, `internal/infra/`)
- [ ] CI/CD workflow or quality gates changed (`.github/workflows/`, `Makefile`)
- [ ] Database schema/migrations changed (`migrations/`)

## Test Evidence

- [ ] `make verify` passed, or an exact passing receipt was reused
- [ ] Additional checks only for claims `make verify` marked not applicable
- [ ] `ALLOW_FULL=1 make check` only when the claim spans the full repository
- [ ] Unverified remainder named, or none

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
