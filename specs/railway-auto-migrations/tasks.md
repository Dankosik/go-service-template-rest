## Implementation Handoff

Consumes: approved `spec.md`, `design/`, and `rollout.md`.
Implementation readiness: PASS.
First task: `T001`.
Accepted concerns: same-deploy schema changes must stay mixed-version compatible while Railway overlap remains enabled.
Reopen target: specification if the repo needs a different migration owner than Railway pre-deploy.

## Tasks

- [x] T001 [Phase 1] Add a dedicated migration execution path in `cmd/migrate/` and `internal/infra/postgres/` that reuses the template's Postgres DSN safety rules and applies `env/migrations/` once per run.
  Proof: `go test ./cmd/migrate ./internal/infra/postgres`

- [x] T002 [Phase 1] Update `build/docker/Dockerfile` so the final image includes `/service`, `/migrate`, and `/env/migrations`.
  Depends on: `T001`
  Proof: `go test ./cmd/migrate ./internal/infra/postgres`

- [x] T003 [Phase 1] Wire `railway.toml` and `scripts/ci/required-guardrails-check.sh` to the new one-migrator pre-deploy policy, and document the operator prerequisites in `docs/railway-deployment-profile.md`.
  Depends on: `T001`, `T002`
  Proof: `make guardrails-check`

- [x] T004 [Phase 2] Validate the new migration path against ephemeral Postgres and rerun repository-owned migration rehearsal.
  Depends on: `T001`, `T002`, `T003`
  Proof: `APP__POSTGRES__ENABLED=true APP__POSTGRES__DSN=... go run ./cmd/migrate`, `make migration-validate`, `go test ./...`
