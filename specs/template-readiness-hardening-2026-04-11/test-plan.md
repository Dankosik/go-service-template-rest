# Test Plan

## Always Required

- `make check`

## Config And Clone Proof

- `go test ./internal/config -count=1`
- Add/verify a test that `.env.example` values load successfully through the config loader path.
- Add tests for secret-like key rejection:
  - `client_secret`
  - `jwt_secret`
  - `api_key`
  - `private_key`
  - top-level `token`

## Bootstrap / Readiness / Shutdown Proof

- `go test ./cmd/service/internal/bootstrap -count=1`
- `go test ./internal/app/health -count=1`
- Add/verify tests for:
  - external readiness is 503 before admission ready,
  - internal admission check does not depend on admission already being ready,
  - partially initialized dependencies clean up on later dependency failure,
  - budget validation rejects impossible shutdown/write/readiness combinations if validation is added.

## HTTP / OpenAPI / Metrics Proof

- `go test ./internal/infra/http -count=1`
- `make openapi-check` when `api/openapi/service.yaml` changes.
- Add/verify tests for:
  - no production fallback dependencies in router construction,
  - `/metrics` root-router exception is intentional and future manual/generated overlaps fail,
  - CORS preflight remains fail-closed,
  - malformed request details do not echo raw parser/attacker-controlled text when a current request shape can trigger it.

## Persistence / SQLC Proof

- `make sqlc-check`
- If native sqlc remains blocked by `pg_query_go` on macOS, run `make docker-sqlc-check`.
- Run `make test-integration` only when migrations, sqlc-generated behavior, repository integration behavior, or integration helpers change.
- If `ping_history` is removed or moved, confirm generated sqlc drift is intentional and committed.

## Docs / Make Proof

- `make help` should show the new feature validation group.
- If docs drift tooling applies in PR context, run `make docs-drift-check BASE_REF=<base> HEAD_REF=<head>`.
- Manual doc review should confirm:
  - new endpoint flow includes security decision,
  - new integration flow says bootstrap, not `cmd/service/main.go`,
  - test placement is layer-specific,
  - `/metrics` exposure policy is explicit,
  - persistence/sqlc flow is clear and does not present sample schema as hidden production behavior.

