# Project Structure & Module Organization

This document explains the `go-service-template-rest` repository layout: what is stored where, why it is placed there, and where to add new code.

## 1) Project Tree

```text
.
├── .agents/
│   └── skills/
├── .claude/
│   └── skills/
├── .cursor/
│   └── skills/
├── .gemini/
│   └── skills/
├── .opencode/
│   └── skills/
├── .github/
│   ├── CODEOWNERS
│   ├── dependabot.yml
│   ├── pull_request_template.md
│   ├── skills/
│   └── workflows/
│       ├── cd.yml
│       ├── ci.yml
│       └── nightly.yml
├── api/
│   ├── openapi/
│   │   └── service.yaml
│   └── proto/
│       └── service/
│           └── v1/
│               └── service.proto
├── build/
│   ├── ci/
│   │   └── README.md
│   └── docker/
│       ├── Dockerfile
│       └── tooling-images.Dockerfile
├── cmd/
│   └── service/
│       ├── main.go
│       └── internal/
│           └── bootstrap/
│               ├── run.go
│               ├── startup_bootstrap.go
│               ├── startup_common.go
│               ├── startup_dependencies.go
│               ├── startup_probe_addresses.go
│               ├── startup_probe_helpers.go
│               ├── startup_server.go
│               ├── shutdown.go
│               └── network_policy*.go
├── docs/
│   ├── llm/
│   │   └── go-instructions/
│   │       ├── README.md
│   │       ├── 10-go-errors-and-context.md
│   │       ├── 20-go-concurrency.md
│   │       ├── 30-go-project-layout-and-modules.md
│   │       ├── 40-go-testing-and-quality.md
│   │       ├── 50-go-public-api-and-docs.md
│   │       ├── 60-go-performance-and-profiling.md
│   │       └── 70-go-review-checklist.md
│   └── project-structure-and-module-organization.md
├── env/
│   ├── .env.example
│   ├── docker-compose.yml
│   └── migrations/
│       ├── 000001_init.up.sql
│       ├── 000001_init.down.sql
│       ├── 000002_ping_history_recent_index.up.sql
│       └── 000002_ping_history_recent_index.down.sql
├── internal/
│   ├── api/
│   │   ├── README.md
│   │   ├── doc.go
│   │   └── oapi-codegen.yaml
│   ├── app/
│   │   ├── health/
│   │   │   ├── service.go
│   │   │   └── service_test.go
│   │   └── ping/
│   │       └── service.go
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   ├── domain/
│   └── infra/
│       ├── http/
│       │   ├── handlers.go
│       │   ├── middleware.go
│       │   ├── openapi_contract_test.go
│       │   ├── problem.go
│       │   ├── router.go
│       │   ├── router_test.go
│       │   ├── server.go
│       │   └── goleak_test.go
│       ├── postgres/
│       │   ├── queries/
│       │   ├── sqlcgen/
│       │   ├── postgres.go
│       │   └── ping_history_repository.go
│       └── telemetry/
│           ├── metrics.go
│           ├── metrics_test.go
│           ├── tracing.go
│           └── tracing_test.go
├── scripts/
│   ├── ci/
│   │   ├── docs-drift-check.sh
│   │   └── required-guardrails-check.sh
│   ├── dev/
│   │   ├── configure-branch-protection.sh
│   │   ├── doctor.sh
│   │   ├── sync-skills.sh
│   │   └── setup.sh
│   ├── gen.sh
│   └── init-module.sh
├── test/
│   ├── README.md
│   ├── postgres_integration_test.go
│   └── postgres_sqlc_integration_test.go
├── .dockerignore
├── .gitignore
├── .golangci.yml
├── .redocly.yaml
├── AGENTS.md
├── Makefile
├── README.md
├── go.mod
└── go.sum
```

## 2) Layer and Folder Responsibilities

### `cmd/`
Thin executable entrypoints.  
Why: startup and wiring are separated from business logic. This makes it easier to reuse code and add new binaries (for example, worker, migrator, admin CLI) without duplicating domain logic.

### `cmd/service/internal/bootstrap/`
Service bootstrap implementation for the `service` binary: startup orchestration, dependency probes, shutdown flow, and deploy/network policy helpers used during process lifecycle.
Why: keeps `cmd/service/main.go` as a composition entrypoint while moving complex lifecycle logic into focused files with local tests.

### `internal/`
Private service code that is not part of the module public API.  
Why: Go `internal` enforces import boundaries and keeps the service contract controlled.

### `internal/app/`
Use-case layer: business scenarios and orchestration without transport or storage details.  
Why: this behavior can be reused by HTTP handlers, background jobs, CLI commands, and tests.

### `internal/domain/`
Minimal shared domain contracts and types only when more than one app package needs the same abstraction.
Why: consumer-owned interfaces stay beside their app consumer by default; readiness probes are owned by `internal/app/health.Probe`.

### `internal/infra/`
Infrastructure adapters: HTTP, Postgres, telemetry.  
Why: framework and integration details are isolated from business code; replacing an adapter affects minimal code.

HTTP route ownership: normal API endpoints are added through `api/openapi/service.yaml` and generated bindings first. Manual root-router routes are only for documented operational exceptions such as `/metrics`; do not add manual `/api/...` routes.

Postgres/sqlc ownership: `env/migrations/*.sql` owns schema shape, `internal/infra/postgres/queries/*.sql` owns query sources, and `internal/infra/postgres/sqlcgen` is generated output. Hand-written repositories under `internal/infra/postgres` translate generated rows into app-facing types instead of leaking `sqlcgen` into `internal/app`.

`ping_history` is retained as a template SQLC sample because the current generator setup requires at least one query to prove drift checks. It is not production business state and must not be wired into `ping` as a side effect. New services should replace the sample with real feature-owned migrations, queries, repositories, and app ports.

### `internal/config/`
Environment-based config loading and validation, including defaults.  
Why: one config source reduces local/CI/prod drift and keeps startup behavior predictable.

### `internal/api/`
Generated Go bindings from OpenAPI (`go generate`, `oapi-codegen`).  
Why: contract is maintained separately (`api/openapi/service.yaml`) and code is synchronized from a single source of truth.

### `api/openapi/`
REST API contract (source of truth).  
Why: contract-first workflow gives predictable API evolution, lint/validate/breaking checks in CI, and clear visibility for API consumers.

Security note: the baseline endpoints are public system/sample endpoints and the contract intentionally does not define placeholder auth. Each new business operation must declare one of three choices before implementation: public by design, protected by real auth middleware and OpenAPI `security` plus 401/403 Problem responses, or blocked pending a security spec.

### `api/proto/`
Optional protobuf contract for gRPC/inter-service communication.  
Why: REST and protobuf contracts are explicitly separated instead of mixed into runtime code.

### `env/`
Local environment assets: env template, docker-compose, SQL migrations.  
Why: everything needed for local reproducible runs is versioned and kept together.

Migrations are deterministic by default: use plain `CREATE` / `DROP` statements so unexpected schema drift fails loudly. Use `IF NOT EXISTS` / `IF EXISTS` only for an explicitly reviewed repair or idempotent migration.

### `test/`
Integration/e2e tests and larger test scenarios (separate from unit tests in `internal/...`).  
Why: fast unit tests stay close to code, while heavier scenarios run separately with the `integration` tag.

### `build/`
Build and delivery assets: Dockerfile, CI notes, related build files.  
Why: separates runtime code from build/deploy concerns and keeps `internal/` focused.

### `scripts/`
Developer and CI helper scripts.  
Why: standard commands for local work and CI without repeating long command lines.

Key scripts:
- `scripts/dev/setup.sh`: onboarding bootstrap (native or docker mode), `.env` creation, skills sync, module auto-initialization from `git remote origin`, CODEOWNER inference from origin, and optional strict native coverage sanity (`--strict`).
- `scripts/dev/doctor.sh`: readiness checks for native/docker prerequisites and template placeholders.
- `scripts/init-module.sh`: manual fallback for module path and CODEOWNERS initialization after clone.
- `scripts/dev/docker-tooling.sh`: zero-setup wrappers for test/lint/OpenAPI/security/CI flows without host Go/Node toolchain; tooling image references are read from `build/docker/tooling-images.Dockerfile`.

### `docs/`
Engineering documentation (including LLM instructions and this document).  
Why: development rules and structure should be explicit and versioned, not scattered in code comments.

### `.agents/skills`
Canonical source of runnable `SKILL.md` definitions.  
Why: this is the repository-native authoring surface used as the source for skill mirror sync.

### `.claude/skills`, `.gemini/skills`, `.github/skills`, `.cursor/skills`, `.opencode/skills`
Provider runtime skill directories (`SKILL.md` files are stored here).  
Why: these are the locations where agent tools actually load and execute mirrored skills.

### `.github/`
CI workflow and dependency update automation (Dependabot).  
Why: quality and security checks are codified, reviewable, and reproducible on every PR.

## 3) Code Ownership Boundaries

`cmd/service/main.go` should only perform composition and delegate lifecycle orchestration to `cmd/service/internal/bootstrap`:
- read config;
- wire dependencies;
- call bootstrap runner with CLI flags/context;
- return process exit status based on bootstrap result.

`internal/app/*` should not import `internal/infra/http` or concrete database drivers.

`internal/infra/*` can import external libraries (`pgx`, Prometheus, and similar), because these packages are adapters.

`internal/domain/*` should remain small and stable: only shared contracts/types. Prefer a consumer-owned interface beside the `internal/app/<feature>` package first; for readiness, implement `internal/app/health.Probe`.

`/metrics` is operational telemetry, not a normal public business endpoint. If the service HTTP listener is internet-facing, expose `/metrics` only through a private scrape path/network or add a real auth/internal-listener design first. Browser CORS remains fail-closed until a dedicated security decision covers origins, credentials, headers, and protected endpoints.

## 4) Where to Put New Code

New HTTP endpoint:
1. Add or update contract in `api/openapi/service.yaml`.
2. Record the endpoint security decision: public by design, protected by real auth, or blocked pending security design. For protected endpoints, define OpenAPI `security`, 401/403 Problem responses, identity middleware, tenant/object authorization rules, and negative tests.
3. Generate or refresh API artifacts in `internal/api`.
4. Add use-case logic in `internal/app/<feature>`.
5. Add handler mapping in `internal/infra/http`; do not bypass generated routing with a manual `/api/...` chi route.
6. Add tests near changed code and add integration tests in `test/` when needed.
7. Validate with `make openapi-check`, plus targeted handler/app tests for the changed behavior.

New Postgres persistence:
1. Add a deterministic migration under `env/migrations`.
2. Add SQLC query sources under `internal/infra/postgres/queries/*.sql`.
3. Regenerate `internal/infra/postgres/sqlcgen` with `make sqlc-generate`; do not hand-edit generated files.
4. Add a hand-written repository under `internal/infra/postgres` that maps generated rows/types into app-facing records.
5. Add an app-owned port beside the consumer in `internal/app/<feature>` when the app layer needs inversion over the adapter; use `internal/domain` only for a genuinely shared stable contract.
6. Wire the concrete repository in `cmd/service/internal/bootstrap`.
7. Validate with `make sqlc-check`, repository unit tests, and `make test-integration` when migration-backed behavior changed.

New integration (Redis, Kafka, S3, external API):
1. Add adapter in `internal/infra/<integration>`.
2. Add an app-owned or domain interface only if `app` needs inversion over the concrete adapter.
3. Wire the concrete adapter in `cmd/service/internal/bootstrap`; keep `cmd/service/main.go` thin.
4. For outbound calls, declare the target source, timeout, redirect policy, DNS/IP-class behavior, and egress allowlist policy before wiring. Fixed outbound targets are validated by bootstrap policy. Dynamic or user-controlled URLs require a separate security design and review before implementation.

New binary:
1. Create `cmd/<binary>/main.go`.
2. Reuse existing packages from `internal/*`.
3. Do not duplicate business logic in `cmd`.

Changes to startup/lifecycle flow of the `service` binary:
1. Keep `cmd/service/main.go` thin.
2. Add/modify logic in `cmd/service/internal/bootstrap/*`.
3. Add tests near modified bootstrap files (`*_test.go` in the same folder).

Test placement by layer:
- Unit tests for app/domain behavior stay beside the package under `internal/app` or `internal/domain`.
- HTTP mapping, generated-route ownership, CORS, and Problem response tests stay under `internal/infra/http`.
- Bootstrap lifecycle, config wiring, dependency admission, and shutdown tests stay under `cmd/service/internal/bootstrap`.
- Postgres pool and repository unit tests stay under `internal/infra/postgres`; migration-backed read/write tests stay under `test/` with the `integration` build tag.
- Cross-package or end-to-end scenarios belong under `test/` or a focused subdirectory below it, using an external package such as `integration_test` when possible.

## 5) Why This Structure Scales

- Improves code review: contract, use-case, and infrastructure concerns are easy to locate.
- Reduces coupling: `app` layer can be tested without booting HTTP server or database.
- Supports contract-first API evolution through OpenAPI and CI quality gates.
- Speeds onboarding: each top-level folder has one clear responsibility.
