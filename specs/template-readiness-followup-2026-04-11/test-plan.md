# Test Plan

## Scope

This validation plan covers the future implementation of the template-readiness follow-up. It does not prove current behavior before implementation.

## Phase 1 Tests

Run:

```bash
go test ./internal/infra/http -count=1
```

Also run:

```bash
make openapi-check
```

when `api/openapi/service.yaml` or generated `internal/api` output changes.

Expected proof:

- generated chi wrapper errors use Problem JSON,
- raw parser details are not exposed,
- every OpenAPI operation has an explicit security decision marker,
- protected operation rules are executable for future endpoints,
- no fake auth scheme is introduced.

## Phase 2 Tests

Run:

```bash
go test ./cmd/service/internal/bootstrap ./internal/config ./internal/infra/http ./internal/infra/telemetry -count=1
```

Expected proof:

- non-local wildcard bind requires explicit ingress declaration,
- public ingress still requires exception metadata when declared true,
- private ingress assertion can be represented explicitly,
- panic recovery logs do not include raw secret-like panic values,
- malformed OTLP headers do not leak raw header values,
- YAML secret placeholder behavior matches docs.

## Phase 3 Tests

Run:

```bash
go test ./internal/config ./internal/app/health ./cmd/service/internal/bootstrap -count=1
make check
```

Expected proof:

- shutdown timeout can be tuned within validated relationships, or exact lock is explicitly tested and documented,
- readiness timeout validates aggregate sequential probe budget,
- config key drift test fails for keys not represented in defaults/types,
- bootstrap helper cleanup does not change startup/dependency behavior.

## Phase 4 Tests

Run:

```bash
go test ./...
make check
```

Conditional:

```bash
make sqlc-check
make docker-sqlc-check
make migration-validate
make docker-migration-validate
make test-integration
```

Only run SQLC/migration/integration commands if implementation changes SQLC, migrations, or migration-backed runtime behavior. Docs-only mentions of those commands do not require live Postgres proof.

Expected proof:

- docs and clone-readiness polish do not break tests,
- `internal/domain/doc.go`, if added, does not introduce unwanted dependencies,
- feature-validation guidance matches Makefile targets.

## Final Proof

Required before reporting implementation complete:

```bash
go test ./cmd/service/internal/bootstrap ./internal/config ./internal/infra/http ./internal/infra/telemetry ./internal/app/... -count=1
make check
```

Required when OpenAPI changed:

```bash
make openapi-check
```

Required only when SQLC or migrations changed:

```bash
make sqlc-check
# or, if native sqlc is blocked:
make docker-sqlc-check
```

Required only when migration-backed behavior changed:

```bash
make test-integration
make migration-validate
```

## Non-Proofs

- Passing `make check` does not prove OpenAPI security-decision linting unless the new test is included and runs there.
- Passing `make sqlc-check` does not prove migration rollback/reapply behavior.
- Passing docs-only checks does not prove public ingress safety; bootstrap tests must cover the declaration policy.
