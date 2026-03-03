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
│       └── main.go
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
│       └── 000001_init.down.sql
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
│   │   └── readiness.go
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
│       │   └── postgres.go
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
├── skills/
│   ├── api-contract-designer-spec/
│   │   └── SKILL.md
│   ├── go-architect-spec/
│   │   └── SKILL.md
│   └── ... (other skill packs)
├── test/
│   ├── README.md
│   └── postgres_integration_test.go
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

### `internal/`
Private service code that is not part of the module public API.  
Why: Go `internal` enforces import boundaries and keeps the service contract controlled.

### `internal/app/`
Use-case layer: business scenarios and orchestration without transport or storage details.  
Why: this behavior can be reused by HTTP handlers, background jobs, CLI commands, and tests.

### `internal/domain/`
Minimal domain contracts and interfaces (for example, `ReadinessProbe`).  
Why: the `app` layer depends on abstractions, while concrete implementations live in `infra`.

### `internal/infra/`
Infrastructure adapters: HTTP, Postgres, telemetry.  
Why: framework and integration details are isolated from business code; replacing an adapter affects minimal code.

### `internal/config/`
Environment-based config loading and validation, including defaults.  
Why: one config source reduces local/CI/prod drift and keeps startup behavior predictable.

### `internal/api/`
Generated Go bindings from OpenAPI (`go generate`, `oapi-codegen`).  
Why: contract is maintained separately (`api/openapi/service.yaml`) and code is synchronized from a single source of truth.

### `api/openapi/`
REST API contract (source of truth).  
Why: contract-first workflow gives predictable API evolution, lint/validate/breaking checks in CI, and clear visibility for API consumers.

### `api/proto/`
Optional protobuf contract for gRPC/inter-service communication.  
Why: REST and protobuf contracts are explicitly separated instead of mixed into runtime code.

### `env/`
Local environment assets: env template, docker-compose, SQL migrations.  
Why: everything needed for local reproducible runs is versioned and kept together.

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

### `skills/`
Canonical source of runnable `SKILL.md` definitions.  
Why: this directory is the single source for skill content, which is mirrored to provider runtime directories via `make skills-sync`.

### `.agents/skills`, `.claude/skills`, `.gemini/skills`, `.github/skills`, `.cursor/skills`, `.opencode/skills`
Provider runtime skill directories (`SKILL.md` files are stored here).  
Why: these are the locations where agent tools actually load and execute skills.

### `.github/`
CI workflow and dependency update automation (Dependabot).  
Why: quality and security checks are codified, reviewable, and reproducible on every PR.

## 3) Code Ownership Boundaries

`cmd/service/main.go` should only perform composition:
- read config;
- wire dependencies;
- start the HTTP server;
- perform graceful shutdown.

`internal/app/*` should not import `internal/infra/http` or concrete database drivers.

`internal/infra/*` can import external libraries (`pgx`, Prometheus, and similar), because these packages are adapters.

`internal/domain/*` should remain small and stable: only required contracts/types.

## 4) Where to Put New Code

New HTTP endpoint:
1. Add or update contract in `api/openapi/service.yaml`.
2. Generate or refresh API artifacts in `internal/api`.
3. Add use-case logic in `internal/app/<feature>`.
4. Add handler/routing wiring in `internal/infra/http`.
5. Add tests near changed code and add integration tests in `test/` when needed.

New integration (Redis, Kafka, S3, external API):
1. Add adapter in `internal/infra/<integration>`.
2. Add a domain interface in `internal/domain` only if `app` needs it.
3. Wire it in `cmd/service/main.go`.

New binary:
1. Create `cmd/<binary>/main.go`.
2. Reuse existing packages from `internal/*`.
3. Do not duplicate business logic in `cmd`.

## 5) Why This Structure Scales

- Improves code review: contract, use-case, and infrastructure concerns are easy to locate.
- Reduces coupling: `app` layer can be tested without booting HTTP server or database.
- Supports contract-first API evolution through OpenAPI and CI quality gates.
- Speeds onboarding: each top-level folder has one clear responsibility.
