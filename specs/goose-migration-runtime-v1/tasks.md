# Goal

status: done

Completion: the template has one Goose v3.27.3 SQL migration owner from
canonical source through local validation, production `/migrate`, serialized
append-only image publication, profile initialization, and current
claim-scoped proof, with no live `golang-migrate` owner or stale operator
contract.

Blocked stop: if Goose Provider or GHCR cannot implement the accepted locked
prefix check, bounded cleanup, or durable marker protocol, record the narrow
failing TD scenario and reopen the corresponding Execution, Reliability, or
Publication decision. Live image publication and Railway deployment remain
unclaimed external writes.

Global constraints: one fixed-width Goose SQL file with Up and Down; no Go
migrations, environment substitution, `NO TRANSACTION`, production Down, SQL
logging, automatic version-table mutation, or application-startup migration.
Broad Go/Docker gates run serially. Existing derived services are outside this
template cutover.

- [x] T1: Goose is the sole local and production migration owner with canonical source, locked prefix-safe execution, bounded lifecycle, safe results, profile-pure tooling, and matching runtime/recovery documentation
  - Source: `spec.md` Source/Production/Concurrency/Timeout/State/Result sections; `design/overview.md` Source, Production flow, Result, Go ownership; TD-1..TD-9, TD-11..TD-13
  - Owner/surface/resources: canonical runtime/package owners `internal/infra/postgresmigrate/**`, `cmd/migrate/**`, reached callers `internal/infra/postgres/pgtest/**`; config owners `internal/config/{types,defaults,validate,snapshot_contract,validate_test}.go` and reached environment/config fixtures; canonical module owners `go.mod`, `go.sum`, `tools/go.mod`, `tools/go.sum`; source/generation owners `migrations/**` when present and `internal/infra/postgres/sqlc.yaml`; local gates and harness owners `Makefile`, `scripts/ci/project-structure-check.sh`, new migration source/history scripts and their self-tests, `test/postgres_migrate_runner_integration_test.go`; Docker/image and profile owners `build/docker/Dockerfile`, `scripts/init-module.sh`, `scripts/ci/template-init-check.sh`, `scripts/ci/fixtures/postgres-post-feature.patch`; runtime, source, configuration, architecture, recovery, Railway pre-deploy, and developer-command contract owners `README.md`, `docs/build-test-and-development-commands.md`, `docs/configuration-source-policy.md`, `docs/project-structure-and-module-organization.md`, `docs/first-production-feature.md`, `docs/railway-deployment-profile.md`, `docs/repo-architecture.md`; mutable resources are one repository worktree, one serialized Docker/PostgreSQL aggregate, and generated module/sqlc output
  - Depends on: none
  - Proof: canonical negative sources fail before database open; valid empty and populated sources reach prefix state; Up/no-op/Down/Up, transaction rollback, inconsistent state, contention, timeouts, cancellation, and cleanup match TD-1..TD-9; sqlc consumes Goose Up sections; database-none removes Goose and PostgreSQL retains it; no production/test/module path imports `golang-migrate`; the service dependency graph excludes Goose and `postgresmigrate`, and process proof reaches readiness without a migration attempt; examples use six-digit one-file Goose syntax and recovery uses fail-closed `goose_db_version` prefix diagnosis without dirty/force mutation; `go test -vet=off ./internal/config ./internal/infra/postgresmigrate ./cmd/migrate ./cmd/service/... ./internal/infra/postgres/pgtest`; `go list -deps ./cmd/service` negative dependency check; `go test -vet=off -race ./internal/infra/postgresmigrate ./cmd/migrate`; `go test -count=1 -tags=integration ./test/... -run PostgresMigrate`; `make migration-check`; `make migration-validate`; `make sqlc-check`; `make mod-tidy-check`; `make template-init-check`; migration-focused negative text scan; `git diff --check`
  - Reopen if: Provider cannot keep preflight and Up under one session lock or cleanup cannot fit one detached reserve; Execution or Reliability Design
  - Acceptance: PASS — focused/full unit, race/liveness, real PostgreSQL integration, production-image rehearsal, sqlc, module, lint, security, and both initialized template profiles passed on the complete local candidate.

- [x] T2: Main and release publication plus their operator contract share one fail-closed OCI migration-history authority and never advance a public alias before it
  - Source: `spec.md` Decisions and success criterion 7; `design/overview.md` durable publication boundary and delivery; TD-10
  - Owner/surface/resources: `.github/workflows/ci.yml`, `.github/workflows/cd.yml`, new `scripts/ci/migration-image-history-check.sh` and self-test, Makefile publication-check target, publication/retention/operator sections in `docs/ci-cd-production-ready.md` and `docs/railway-deployment-profile.md`; mutable resources are the shared GHCR package marker and channel tags, represented locally only by directory/API fixtures; no live registry write is authorized
  - Depends on: T1 — output handoff — needed to start
  - Handoff: T1 provides the canonical admitted image corpus, source/history checks, production image layout, and migration rehearsal consumed by both publication channels
  - External input/gate: repository owner sets `ENABLE_GHCR_PUBLISH=true` for publication and sets `MIGRATION_HISTORY_BOOTSTRAP_SHA` to exactly the first authorized candidate only after local/CI readiness; live execution additionally requires an authorized external publication
  - Proof: exact same-SHA push CI is required; owner/package pagination succeeds completely; `401`/`403`/`404`/truncation and any existing package reject bootstrap; exact first SHA with a proved absent package passes; existing trusted marker accepts additions and rejects byte/path changes; main/release share non-cancelling serialization; candidate trust and marker digest are verified before any public alias; marker-success/alias-failure remains conservative; docs distinguish local readiness from live publication and name bootstrap/retention/recovery inputs; `make migration-publication-check`; `make actionlint`; `make zizmor`; `git diff --check`
  - Reopen if: a new image channel, registry, package retention policy, or workflow can publish without the shared marker protocol; Publication Design
  - Acceptance: PASS — publication corpus/API self-tests, workflow static checks, actionlint, zizmor, and ShellCheck passed; live GHCR publication remains an explicitly unclaimed external write.

- [x] T3: The complete Goose candidate passes current equal-scope proof and independent acceptance
  - Source: all success criteria and TD-1..TD-13
  - Owner/surface/resources: bounded worktree diff; Go, Docker, PostgreSQL, linter, scanner, and generated-output gates are non-concurrent host resources
  - Depends on: T2 — proof gate — needed to start
  - Proof: inspect the complete diff and dependency graph; run focused unit/integration/race checks, `make migration-check`, `make migration-publication-check`, `make migration-validate`, `make sqlc-check`, `make template-init-check`, `make mod-tidy-check`, changed-code lint/delivery checks, then the narrowest repository aggregate justified by final surfaces, all serially; independent acceptance returns PASS or every material finding is repaired and re-proved
  - Reopen if: a failure changes source, state, lifecycle, publication, profile, or proof semantics; the corresponding narrow upstream owner
  - Acceptance: PASS — independent final review verified that only a successful `push` CI event can authorize main publication, the repaired guard is checked inside the actual job condition, and no alternate marker or public-alias route exists. Live GHCR publication and Railway deployment remain outside local acceptance.
