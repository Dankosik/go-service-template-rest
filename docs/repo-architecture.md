# Repository Architecture Baseline

This document is the stable repository-wide architecture baseline for `go-service-template-rest`.
Use it to recover the standing component boundaries, ownership rules, and runtime flow before writing task-local design in `specs/<feature-id>/design/`.

This file is intentionally narrower than:
- [Project Structure & Module Organization](./project-structure-and-module-organization.md)
- [Configuration Source Policy](./configuration-source-policy.md)
- [Build, Test, and Development Commands](./build-test-and-development-commands.md)
- [CI/CD Production-Ready Checklist](./ci-cd-production-ready.md)

It does not restate the full tree, every command, or task-local design choices.

## Stable Component Boundaries

| Area | Owns | Does not own |
| --- | --- | --- |
| `cmd/service/main.go` | Thin process entrypoint. Delegates immediately to bootstrap. | Business logic, request handling, dependency details. |
| `cmd/service/internal/bootstrap/` | Service composition root, startup/shutdown flow, config/bootstrap lifecycle, dependency admission, runtime policy. | Use-case semantics, transport contract definition, persistence logic. |
| `internal/config/` | Building one validated, immutable runtime config snapshot from defaults, config files, env, and flags. | Feature behavior, dependency wiring, request handling. |
| `api/openapi/service.yaml` | Source of truth for the REST contract. | Hand-written runtime logic or transport implementation. |
| `internal/api/` | Generated Go bindings derived from the OpenAPI contract. | Manual business logic; hand-editing should not become the source of truth. |
| `internal/app/` | Use-case behavior and service-level orchestration that should stay transport-agnostic. | HTTP details, driver details, process lifecycle. |
| `internal/domain/` | Small stable contracts/types used to decouple app behavior from adapters when needed. | Framework code, transport code, concrete integration code. |
| `internal/infra/http/` | HTTP server, middleware, request/response mapping, route policy, and observability at the transport edge. | Core business rules or config loading. |
| `internal/infra/postgres/` | Postgres connection/pool lifecycle and repository code. | Process lifecycle, HTTP behavior, config precedence rules. |
| `internal/infra/telemetry/` | Prometheus metrics and OpenTelemetry tracing setup/adapters. | Feature semantics or request routing decisions. |
| `env/migrations/` | SQL schema migration source of truth. | Runtime repository logic or generated Go bindings. |

## Source-Of-Truth Ownership

Keep these ownership rules stable across tasks:

| Source of truth | Derived or consuming surfaces |
| --- | --- |
| `api/openapi/service.yaml` | `internal/api/` generated bindings and `internal/infra/http/` transport wiring |
| `internal/config/` snapshot build + validation rules | Runtime config consumed by bootstrap and adapters |
| `env/config/*.yaml`, `APP__...`, and runtime flags | Inputs to `internal/config/`; precedence and secret rules live in [Configuration Source Policy](./configuration-source-policy.md) |
| `env/migrations/*.sql` | Database shape used by Postgres runtime code and any generated SQL access layer |
| `internal/app/*` behavior | Consumed by HTTP handlers now; reusable by future binaries or async workers |
| `cmd/service/internal/bootstrap/*` lifecycle logic | Consumed by the `service` binary only; future binaries should own their own bootstrap flow |

Two repository-wide rules matter most:
1. Generated code is derived code. Edit the contract or generation inputs first, then regenerate.
2. Concrete adapter wiring belongs in the composition root, not inside `internal/app`.

## Stable Dependency Direction

The default dependency direction is inward toward business behavior and outward only at the composition root:

```text
cmd/service/main.go
  -> cmd/service/internal/bootstrap
     -> internal/config
     -> internal/app/*
     -> internal/infra/*

internal/infra/http
  -> internal/api
  -> internal/app/*

internal/app/*
  -> internal/domain/*   (when an abstraction is needed)

internal/infra/postgres, internal/infra/telemetry
  -> external libraries
  -> internal/domain/*   (only if an app-facing contract exists)
```

Stable direction rules:
- `internal/app` must not depend on `internal/infra/http` or other concrete transport packages.
- Concrete integration packages belong under `internal/infra/*` and may depend on external libraries.
- `internal/domain` should stay small and stable; add contracts there only when the app layer needs an abstraction.
- `cmd/service/internal/bootstrap` is allowed to know concrete adapters because it is the composition root.

## Primary Runtime Flows

### Request/Response Path

1. `cmd/service/internal/bootstrap.Run` builds the config snapshot, lifecycle logging, telemetry, dependency probes, app services, router, and HTTP server.
2. `internal/infra/http.NewRouter` wraps the root router with request correlation, security headers, framing/body guards, panic recovery, access logging, route labeling, metrics, and tracing middleware.
3. `/metrics` is the documented operational root-router exception, served directly to avoid strict-handler buffering while still being guarded against accidental generated/manual route overlap. API routes are handled through the generated strict OpenAPI server.
4. `internal/infra/http` maps the request into the generated OpenAPI handler interface and calls the app service (`internal/app/*`).
5. The app service returns domain/use-case results; the HTTP adapter turns them into contract-shaped responses or problem responses.
6. Transport observability is emitted at the edge: request logs, Prometheus request metrics, and OpenTelemetry spans use route labels from the HTTP layer.

Current runtime note: the shipped endpoints are intentionally small (`ping`, liveness, readiness, metrics), and they are public system/sample endpoints. New business endpoints must make a security decision before implementation: public by design, protected by real OpenAPI security plus auth middleware and 401/403 Problem responses, or blocked pending a security spec. Browser CORS remains fail-closed by default.

Operational exposure note: `/metrics` is not an ordinary public business API. Production deployments should expose it only on a private scrape path/network or add a real auth/internal-listener design before internet exposure.

Public ingress note: non-local wildcard binds require an explicit `NETWORK_PUBLIC_INGRESS_ENABLED` declaration. `false` is a private-ingress assertion by the operator; `true` is a public-ingress exception and requires the reviewed ingress-exception metadata.

### Startup/Shutdown Path

1. `cmd/service/main.go` delegates to `bootstrap.Run`.
2. Bootstrap parses config flags, creates the signal-aware root context, initializes baseline metrics, and loads the immutable config snapshot through `internal/config`.
3. Bootstrap reconfigures structured logging from the validated config, initializes tracing in fail-open mode, applies startup network policy checks, and probes enabled dependencies.
4. The HTTP runtime may begin serving while startup admission is still running, but external `/health/ready` stays not ready until startup admission marks the process ready.
5. Readiness is guarded by startup admission, runtime ingress policy, and `internal/app/health.Service`, which runs enabled dependency probes sequentially under one readiness timeout.
6. `/health/live` remains process-only; external dependency checks, startup admission, drain, and ingress-policy checks belong in readiness.
7. On shutdown, bootstrap marks the service as draining, flips readiness off, waits the configured propagation delay, gracefully shuts down the HTTP server, and flushes telemetry inside the process-grace budget.

The lifecycle baseline is: config and dependency validation happen before accepting traffic, and shutdown is coordinated from the bootstrap layer rather than from handlers or app services.

### Background / Async Extension Path

The baseline repository does not ship an always-on background worker runtime.

When a task introduces async work, keep the extension path stable:
1. Put business behavior in `internal/app/<feature>`.
2. Put queue, scheduler, database, or external-system mechanics in `internal/infra/<integration>`.
3. Own lifecycle, config loading, telemetry, and graceful shutdown in a composition root under `cmd/<binary>/` or another explicit bootstrap entrypoint.

Preferred rule: if the workload has a distinct lifecycle or scaling model, add a new binary instead of hiding durable background loops inside HTTP handlers.

## Extension Seams

Use these seams when extending the repository:

- New HTTP capability: update `api/openapi/service.yaml`, regenerate `internal/api`, add use-case logic in `internal/app`, then wire handlers/routes in `internal/infra/http`.
- New persistence flow: add a deterministic migration under `env/migrations`, add SQLC query sources under `internal/infra/postgres/queries`, regenerate `internal/infra/postgres/sqlcgen`, add a hand-written Postgres repository that maps generated rows into app-facing types, add an app-owned port only if needed, then wire the concrete adapter in `cmd/service/internal/bootstrap`.
- New integration adapter: add it under `internal/infra/<integration>`; add an app-owned or domain contract only if `internal/app` needs inversion over the concrete adapter; wire concrete dependencies in `cmd/service/internal/bootstrap`. Before enabling a runtime dependency, define config keys and secret-source policy, network egress admission, criticality/degraded-mode behavior, retry and timeout budget, readiness participation, cleanup on partial initialization, low-cardinality metrics labels, and bootstrap tests.
- New outbound target: fixed targets must declare source, timeout, redirect policy, DNS/IP-class behavior, and egress allowlist policy before bootstrap wiring; dynamic or user-controlled URLs require a separate security design.
- New durable schema behavior: evolve `env/migrations/` first, then keep adapter or generated access code derived from that schema.
- New executable surface: add `cmd/<binary>/main.go` with its own bootstrap path and reuse shared app/infra packages instead of duplicating logic.
- New non-HTTP contract surface: `api/proto/` is the reserved source-of-truth location for protobuf contracts when that runtime is introduced.

## Related Repository Docs

Use these docs instead of duplicating their detail here:

- Structure and placement rules: [Project Structure & Module Organization](./project-structure-and-module-organization.md)
- Config sources, precedence, and secret policy: [Configuration Source Policy](./configuration-source-policy.md)
- Local commands, validation commands, and generation flows: [Build, Test, and Development Commands](./build-test-and-development-commands.md)
- CI gates and production-readiness expectations: [CI/CD Production-Ready Checklist](./ci-cd-production-ready.md)
- Task-local workflow and artifact sequencing: [Spec-First Workflow](./spec-first-workflow.md)
