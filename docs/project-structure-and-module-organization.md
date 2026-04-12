# Project Structure & Module Organization

This document explains the `go-service-template-rest` repository layout: what is stored where, why it is placed there, and where to add new code.

## 1) Project Tree

```text
.
├── .agents/                    # canonical project skill sources
├── .codex/, .claude/, ...       # agent/runtime mirrors and agent configs
├── .github/                     # CI, release, skills mirror, repository policy
├── .artifacts/test/             # generated local test reports, not source
├── api/
│   ├── openapi/service.yaml     # REST contract source of truth
│   └── proto/                   # reserved protobuf contract surface
├── build/                       # Docker and CI build assets
├── cmd/service/                 # service binary and bootstrap lifecycle
├── docs/                        # repository and delivery documentation
├── env/
│   ├── config/                  # non-secret config file examples
│   ├── docker-compose.yml
│   └── migrations/              # SQL migration source of truth
├── internal/
│   ├── api/                     # generated OpenAPI bindings plus generation config
│   ├── app/                     # use-case behavior
│   ├── config/                  # config loading, defaults, validation, snapshot
│   ├── domain/                  # small shared contracts only when stable
│   ├── observability/           # narrow shared observability vocabulary, not SDK setup
│   └── infra/
│       ├── http/                # HTTP adapter, middleware, route policy
│       ├── postgres/            # Postgres pool, repositories, sqlc sources/output
│       └── telemetry/           # shared metrics and tracing setup
├── scripts/                     # developer and CI helper scripts
├── specs/                       # spec-first task records and implementation history
├── test/                        # integration and broad scenario tests
├── Makefile
├── README.md
├── go.mod
└── go.sum
```

Generated outputs such as `internal/api/openapi.gen.go`, `internal/infra/postgres/sqlcgen/*`, mock/stringer artifacts, coverage files, and `.artifacts/test/*` reports are derived from their owning sources and commands. Do not edit generated code by hand, and do not treat local report files as design or source artifacts.

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

Domain type decision rule: keep feature-local request/result/value types in `internal/app/<feature>` until two app packages need the same abstraction or the type represents a stable cross-adapter contract. Do not promote a type into `internal/domain` just because a future repository, handler, or worker might use it later.

### `internal/infra/`
Infrastructure adapters: HTTP, Postgres, telemetry.  
Why: framework and integration details are isolated from business code; replacing an adapter affects minimal code.

HTTP route ownership: normal API endpoints are added through `api/openapi/service.yaml` and generated bindings first. Manual root-router routes are only for documented operational exceptions such as `/metrics`; do not add manual `/api/...` routes.

Postgres/sqlc ownership: `env/migrations/*.sql` owns schema shape, `internal/infra/postgres/queries/*.sql` owns query sources, and `internal/infra/postgres/sqlcgen` is generated output. Hand-written repositories under `internal/infra/postgres` translate generated rows into app-facing types instead of leaking `sqlcgen` into `internal/app`.

`ping_history` is retained as a replaceable SQLC fixture because the current generator setup requires at least one query to prove drift checks. It is not production business state and must not be wired into `ping` as a side effect. New services should replace the fixture with real feature-owned migrations, queries, repositories, and app ports.

Feature telemetry placement: HTTP request metrics, route labels, access logs, and request spans belong at the HTTP edge in `internal/infra/http`, using shared instruments from `internal/infra/telemetry` where appropriate. Feature-specific counters, spans, or logs should live beside the feature or adapter that owns the event, use low-cardinality labels, and move into shared telemetry code only after the instrument is genuinely reused.

### `internal/observability/`
Shared observability vocabulary that is not an adapter.
Why: OTel config strings used by both `internal/config` and `internal/infra/telemetry` need one neutral owner without reversing the config-to-infra dependency direction.

`internal/observability/otelconfig` owns OTel sampler/protocol names, defaults, and pure validation helpers only. It must not load config, construct OTel SDK resources/exporters, emit metrics, or become a generic observability helper package.

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

Online production safety is a separate question from deterministic local rehearsal. Prefer additive expand-first migrations for online systems. Destructive changes, type rewrites, large backfills, new constraints, and new indexes need an explicit lock/backfill/mixed-version rollout plan sized to the table and traffic shape. `make migration-validate` and `make docker-migration-validate` rehearse migration mechanics; they do not prove production lock safety, backfill safety, or mixed-version compatibility. Escalate schema ownership, retention, backfill, and rollout questions to data-architecture design before coding.

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
Engineering documentation, including this document and the stable repository architecture baseline.
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

`internal/app/*` and `internal/domain/*` should not import `internal/infra/*`, `internal/infra/postgres/sqlcgen`, or concrete database drivers.

`internal/infra/*` can import external libraries (`pgx`, Prometheus, and similar), because these packages are adapters.

`internal/domain/*` should remain small and stable: only shared contracts/types. Prefer a consumer-owned interface beside the `internal/app/<feature>` package first; for readiness, implement `internal/app/health.Probe`.

`/metrics` is operational telemetry, not a normal public business endpoint. If the service HTTP listener is internet-facing, expose `/metrics` only through a private scrape path/network or add a real auth/internal-listener design first. Browser CORS remains fail-closed until a dedicated security decision covers origins, credentials, headers, and protected endpoints. Fail-closed CORS is not CSRF protection.

Public ingress declaration rule: in non-local environments, wildcard HTTP binds such as `:8080`, `0.0.0.0:8080`, and `[::]:8080` require `NETWORK_PUBLIC_INGRESS_ENABLED` to be explicitly set. `false` is a private-ingress assertion: the operator is saying a platform load balancer, private network, firewall, or equivalent control keeps the listener away from the public internet. `true` is a public-ingress exception and must include the reviewed `NETWORK_INGRESS_EXCEPTION_*` metadata with owner, reason, scope, expiry, and rollback plan.

Health endpoint rule: `/health/live` stays process-only and must not call external dependencies. External dependency checks, startup admission, drain state, and runtime ingress policy belong in readiness, not liveness.

Browser-callable endpoint checklist:
- Record whether browsers may call the endpoint and whether credentials are allowed.
- Define the CORS origin, method, and header allowlist before enabling CORS.
- For cookie-backed flows, define `Secure`, `HttpOnly`, `SameSite`, Path, and Domain attributes.
- Define CSRF controls such as origin checks, token policy, or `http.CrossOriginProtection`.
- Add negative tests for disallowed origins, disallowed methods or headers, missing CSRF controls when credentials are used, and unauthenticated protected calls.

## 4) Where to Put New Code

### Worked Feature Paths

Use these paths as starting points before choosing a narrower recipe below:

| Feature shape | Put the code here | Prove it with |
| --- | --- | --- |
| App-only behavior | Put use-case rules, feature-local request/result types, and consumer-owned ports in `internal/app/<feature>` without adding transport or adapter packages. | App package tests beside `internal/app/<feature>`, plus the caller package tests if an existing adapter starts using it. |
| Simple read-only endpoint | Add use-case behavior in `internal/app/<feature>`, update `api/openapi/service.yaml`, regenerate `internal/api`, implement `strictHandlers.<Operation>` in `internal/infra/http`, and keep manual `/api/...` routes out of the root router. | Contract and handler tests near `internal/infra/http`, app tests near `internal/app/<feature>`, and `make openapi-check`. |
| Postgres-backed endpoint | Keep app behavior and app-owned ports in `internal/app/<feature>`, evolve `env/migrations`, add sqlc queries under `internal/infra/postgres/queries`, regenerate `sqlcgen`, map rows in a hand-written Postgres repository, then inject the concrete adapter in bootstrap. | Repository tests under `internal/infra/postgres`, integration tests under `test/` when container-backed behavior matters, `make sqlc-check`, and migration rehearsal. |
| Background job or worker | Keep business behavior in `internal/app/<feature>`, put queue/scheduler/database/external-system mechanics in `internal/infra/<integration>`, and create a `cmd/<binary>` composition root when lifecycle or scaling differs from the HTTP service. | App tests near the feature, adapter tests near the integration, bootstrap/lifecycle tests for the new binary, and shutdown/cancellation proof for worker loops. |

### First Production Feature Checklist

Before coding the first real business feature, write down the feature owner and keep this path intact:

1. Start in `internal/app/<feature>` with use-case behavior, feature-local request/result/value types, and app-owned ports only when the app must invert a concrete adapter.
2. For HTTP behavior, update `api/openapi/service.yaml`, regenerate `internal/api`, and map generated request/response and Problem shapes in `internal/infra/http`; never add manual `/api/...` routes.
3. For Postgres behavior, replace the `ping_history` SQLC fixture with feature-owned migrations and queries, regenerate `sqlcgen`, map rows in `internal/infra/postgres`, and keep generated types out of `internal/app`.
4. Wire concrete adapters in `cmd/service/internal/bootstrap` after config and dependency admission are defined; prove disabled, ready, and partial-initialization cleanup paths in bootstrap tests.
5. Keep feature-specific telemetry beside the feature or adapter that owns the event; move an instrument into shared telemetry only after it is genuinely reused and low-cardinality.
6. Put tests at the owning layer first: app tests beside `internal/app/<feature>`, HTTP mapping tests beside `internal/infra/http`, repository tests beside `internal/infra/postgres`, config tests beside `internal/config`, bootstrap wiring tests beside `cmd/service/internal/bootstrap`, and integration tests under `test/` only when a real dependency or cross-package scenario is part of the claim.

Existing examples to inspect before adding new surfaces:
- `internal/app/ping` for small app-owned behavior.
- `internal/infra/http` for strict-server handler mapping, generated-route policy, Problem responses, and route labels.
- `internal/infra/postgres/ping_history_repository.go` for the temporary replaceable SQLC fixture shape, not production business ownership.
- `cmd/service/internal/bootstrap` for dependency admission, disabled/ready/cleanup paths, and runtime wiring.

Keep feature-local types in `internal/app/<feature>` until there is a real shared contract. Keep feature-specific telemetry local unless the same low-cardinality instrument is shared across features.

New HTTP endpoint:
1. Add or update contract in `api/openapi/service.yaml`.
2. Record the endpoint security decision: public by design, protected by real auth, or blocked pending security design. For protected endpoints, define OpenAPI `security`, 401/403 Problem responses using the canonical `Problem` schema, scoped generated/strict middleware or an explicitly designed equivalent, identity middleware, tenant/object authorization rules, unauthenticated-call tests, and public-route non-regression tests.
3. Generate or refresh API artifacts in `internal/api`.
4. Add use-case logic in `internal/app/<feature>`.
5. Add handler mapping in `internal/infra/http`; do not bypass generated routing with a manual `/api/...` chi route.
6. Add tests near changed code and add integration tests in `test/` when needed.
7. For parameterized routes, prove logs, metrics, and spans use route templates such as `/users/{id}` rather than concrete IDs.
8. Validate with `make openapi-check`, plus targeted handler/app tests for the changed behavior.

New Postgres persistence:
1. Replace the template `ping_history` sample with real feature-owned schema and queries instead of wiring the sample into app behavior.
2. Add a deterministic migration under `env/migrations`.
3. Add SQLC query sources under `internal/infra/postgres/queries/*.sql`.
4. Regenerate `internal/infra/postgres/sqlcgen` with `make sqlc-generate`; do not hand-edit generated files.
5. Add a hand-written repository under `internal/infra/postgres` that maps generated rows/types into app-facing records.
6. Add an app-owned port beside the consumer in `internal/app/<feature>` when the app layer needs inversion over the adapter; use `internal/domain` only for a genuinely shared stable contract.
7. Wire the concrete repository in `cmd/service/internal/bootstrap`.
8. Clamp bounded list limits before values reach SQL `LIMIT`; the API/app contract or repository must define the upper bound instead of trusting caller input.
9. Validate with `make sqlc-check`, repository unit tests, `make test-integration`, and `make migration-validate` when migration-backed behavior changed. Use `make docker-migration-validate` when the native migration toolchain is unavailable.

Transaction recipe:
1. Start the transaction with the caller context.
2. Bind sqlc queries to the transaction for the duration of the operation.
3. Defer rollback next to transaction creation and use a bounded cleanup context when the driver requires context for cleanup.
4. Commit once, with the caller context, after all side effects that belong inside the transaction have succeeded.
5. Do not add a generic transaction helper until repeated production code shows one stable local policy.

DB-required feature bootstrap:
1. Validate required Postgres config before constructing feature repositories.
2. Construct repositories only after an initialized pool exists and the dependency is enabled for that feature.
3. Inject the concrete repository through an app-owned port beside `internal/app/<feature>`.
4. Test disabled, ready, and partial-initialization cleanup paths in bootstrap.

App-facing persistence port sketch:

```go
// internal/app/orders/store.go
package orders

import "context"

type Store interface {
	Save(ctx context.Context, order Order) error
}
```

The port belongs beside the app feature that consumes it. The Postgres implementation belongs under `internal/infra/postgres`, and bootstrap wires the concrete adapter into the app service. Do not add a generic runtime port package just to prepare for future repositories.

Redis and Mongo extension note: current Redis and Mongo config/probe keys are guard-only extension stubs unless a real feature owns adapter behavior. The enabled flags, probe addresses, readiness flags, probe timeouts, and Redis store-mode admission guard are active bootstrap controls. Redis cache/store knobs and Mongo database/pool/server-selection knobs are reserved future adapter API: they may remain in the config contract for compatibility, but they are not full cache, store, or database runtime behavior in the baseline template. Add `internal/infra/redis` or `internal/infra/mongo` only when an app feature needs real runtime behavior, and do not grow cache/store semantics in `internal/config` or bootstrap alone.

New integration (Redis, Kafka, S3, external API):
1. Add adapter in `internal/infra/<integration>`.
2. Add an app-owned or domain interface only if `app` needs inversion over the concrete adapter.
3. Wire the concrete adapter in `cmd/service/internal/bootstrap`; keep `cmd/service/main.go` thin.
4. For outbound calls, declare the target source, timeout, redirect policy, DNS/IP-class behavior, and egress allowlist policy before wiring. Fixed outbound targets are validated by bootstrap policy. Dynamic or user-controlled URLs require a separate security design and review before implementation.
5. Add the runtime dependency admission checklist before enabling it in startup:
   - config keys, defaults, secret-source policy, and validation;
   - network policy egress target and public/private exposure assumptions;
   - criticality mode (`critical_fail_closed`, `optional_fail_open`, degraded, or feature-off) and degraded-mode contract;
   - retry class, timeout, startup budget, and readiness participation;
   - cleanup registration for partially initialized resources;
   - low-cardinality metrics/log labels such as `startup_dependency_status` `dep` and `mode`;
   - bootstrap tests for disabled, ready, policy-denied, degraded, and cleanup paths.

New binary:
1. Create `cmd/<binary>/main.go`.
2. Reuse existing packages from `internal/*`.
3. Do not duplicate business logic in `cmd`.

Changes to startup/lifecycle flow of the `service` binary:
1. Keep `cmd/service/main.go` thin.
2. Add/modify logic in `cmd/service/internal/bootstrap/*`.
3. Add tests near modified bootstrap files (`*_test.go` in the same folder).

Test placement matrix:

| Behavior under test | Owning test location |
| --- | --- |
| App/domain use-case rules, feature-local types, and app-owned ports | Beside the package under `internal/app/<feature>` or `internal/domain` when a stable shared contract exists. |
| HTTP mapping, generated-route ownership, manual route exceptions, CORS, Problem responses, route labels, metrics, and span naming | `internal/infra/http`. |
| Bootstrap lifecycle, config wiring, dependency admission, disabled/ready/cleanup paths, and shutdown | `cmd/service/internal/bootstrap`. |
| Runtime config key defaults, snapshot construction, validation, and secret-source policy | `internal/config`. |
| Feature bootstrap wiring that introduces a real dependency adapter | `cmd/service/internal/bootstrap`, proving disabled, ready, policy-denied, and partial-initialization cleanup paths before adding broad integration coverage. |
| Postgres pool and hand-written repository mapping | `internal/infra/postgres`. |
| Migration-backed read/write behavior and container-backed database scenarios | `test/` with the `integration` build tag. |
| Endpoint plus real persistence plus bootstrap composition in one scenario | Targeted owner tests first, then `test/` with the `integration` tag when a real database-backed flow is required to prove the combined contract. |
| Broad cross-package or end-to-end scenarios | `test/` or a focused subdirectory below it, using an external package such as `integration_test` when possible. |

See [test/README.md](../test/README.md) for integration-test ownership, build-tag rules, Docker behavior, and migration-backed helper guidance.

## 5) Why This Structure Scales

- Improves code review: contract, use-case, and infrastructure concerns are easy to locate.
- Reduces coupling: `app` layer can be tested without booting HTTP server or database.
- Supports contract-first API evolution through OpenAPI and CI quality gates.
- Speeds onboarding: each top-level folder has one clear responsibility.
