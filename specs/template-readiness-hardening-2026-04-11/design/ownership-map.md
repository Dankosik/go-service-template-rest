# Ownership Map

## Source Of Truth

| Concern | Owner | Implementation Notes |
| --- | --- | --- |
| REST contract | `api/openapi/service.yaml` | Regenerate `internal/api`; do not hand-edit generated code. |
| HTTP route policy | `internal/infra/http` | Normal API routes generated; root exceptions documented and tested. |
| Service composition | `cmd/service/internal/bootstrap` | Owns concrete dependency construction, startup admission, cleanup, and router wiring. |
| App behavior | `internal/app/<feature>` | Transport-agnostic; owns local ports/interfaces when it consumes adapters. |
| Shared domain contracts | `internal/domain` | Only when a contract/type is shared across app packages and is not adapter-specific. |
| Config schema/defaults/validation | `internal/config`, `env/config`, `env/.env.example` | Examples must validate against real config policy. |
| Schema/query generation | `env/migrations`, `internal/infra/postgres/queries`, `internal/infra/postgres/sqlcgen` | Generated code is derived from migrations/queries. |
| Operational telemetry | `internal/infra/telemetry`, `internal/infra/http`, deployment docs | Metrics exposure must be explicit. |
| Validation workflow | `Makefile`, `docs/build-test-and-development-commands.md`, `test/README.md` | `make help` should expose feature validation targets. |

## Dependency Direction

- `cmd/service/main.go -> cmd/service/internal/bootstrap -> internal/app + internal/infra`
- `internal/infra/http -> internal/api + internal/app`
- `internal/app -> local ports and optionally internal/domain`
- `internal/infra/postgres -> sqlcgen + pgx`
- `internal/app` must not import `internal/infra/postgres`, `internal/infra/http`, or `sqlcgen`.

## Helper Extraction Rule

Extract helpers only in the owning package when they prevent policy drift:

- bootstrap ingress policy helper in `cmd/service/internal/bootstrap`,
- bootstrap dependency rejection telemetry helper in `cmd/service/internal/bootstrap`,
- config test env-key derivation in `internal/config` tests.

Do not create a cross-package utility bucket.

