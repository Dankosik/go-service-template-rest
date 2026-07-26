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
| `internal/openapi/` | Generated Go bindings derived from the OpenAPI contract. | Manual business logic; hand-editing should not become the source of truth. |
| `internal/<feature>/` | Feature-owned use cases, business types, ports, invariants, and domain errors. | HTTP details, driver details, runtime config, process lifecycle. |
| `internal/infra/http/` | HTTP server, middleware, request/response mapping, route policy, and observability at the transport edge. | Core business rules or config loading. |
| `internal/infra/httpclient/` (`OUTBOUND_HTTP=bounded`) | Optional shared outbound target validation, transport bounds, trace propagation, and idle-pool cleanup. | Provider authentication, operation budgets, retries, error mapping, or readiness policy. |
| `internal/infra/postgres/` | Optional Postgres connection/pool lifecycle and repository code. | Process lifecycle, migrations, HTTP behavior, config precedence rules. |
| `internal/infra/postgresmigrate/` | Optional migration execution used by `cmd/migrate`. | Runtime pool ownership or application startup. |
| `internal/infra/telemetry/` | OpenTelemetry tracing/metrics SDK setup and Prometheus export. | Feature semantics, startup logging, or request routing decisions. |
| `internal/observability/otelconfig/` | Narrow shared OTel config vocabulary, defaults, and pure validation helpers used by config and telemetry. | Config loading, OTel SDK construction, exporter setup, or generic observability helpers. |
| `migrations/` | SQL schema migration source of truth. | Runtime repository logic or generated Go bindings. |

## Source-Of-Truth Ownership

Keep these ownership rules stable across tasks:

| Source of truth | Derived or consuming surfaces |
| --- | --- |
| `api/openapi/service.yaml` | `internal/openapi/` generated bindings and `internal/infra/http/` transport wiring |
| `internal/config/` snapshot build + validation rules | Runtime config consumed by bootstrap and adapters |
| `env/config/*.yaml`, `APP__...`, and runtime flags | Inputs to `internal/config/`; precedence and secret rules live in [Configuration Source Policy](./configuration-source-policy.md) |
| `migrations/*.sql` | Database shape used by Postgres runtime code and any generated SQL access layer |
| `internal/<feature>/*` behavior | Consumed by HTTP handlers now; reusable by future binaries or async workers |
| `cmd/service/internal/bootstrap/*` lifecycle logic | Consumed by the `service` binary only; future binaries should own their own bootstrap flow |

Two repository-wide rules matter most:
1. Generated code is derived code. Edit the contract or generation inputs first, then regenerate.
2. Concrete adapter wiring belongs in the composition root, not inside `internal/<feature>`.

## Stable Dependency Direction

The default dependency direction is inward toward business behavior and outward only at the composition root:

```text
cmd/service/main.go
  -> cmd/service/internal/bootstrap
     -> internal/config
     -> internal/<feature>/*
     -> internal/infra/*

internal/infra/http
  -> internal/openapi
  -> internal/<feature>/*

internal/infra/postgres, internal/infra/telemetry
  -> external libraries

internal/config, internal/infra/telemetry
  -> internal/observability/otelconfig
```

Stable direction rules:
- `internal/<feature>` must not depend on `internal/infra/http` or other concrete transport packages.
- Concrete integration packages belong under `internal/infra/*` and may depend on external libraries.
- Shared contracts start beside the consuming `internal/<feature>` package and should move only when real reuse exists.
- `cmd/service/internal/bootstrap` is allowed to know concrete adapters because it is the composition root.
- `internal/observability/otelconfig` is a vocabulary package only; it must not import config, infra adapters, or OpenTelemetry SDK packages.

## Primary Runtime Flows

### Request/Response Path

1. `cmd/service/internal/bootstrap.Run` builds the config snapshot, lifecycle logging, telemetry, dependency probes, feature services, router, and HTTP server.
2. `internal/infra/http.NewRouter` validates or creates `X-Request-ID` at the HTTP boundary, extracts only W3C Trace Context, and wraps the root router with security headers, framing/body guards, panic recovery, access logging, route labeling, and OpenTelemetry HTTP instrumentation.
3. The application router contains only the generated client API. `/metrics`
   is served by a separate bootstrap-owned diagnostics listener and is never
   mounted on the application listener.
4. `internal/infra/http` maps the request into the generated OpenAPI handler interface and calls the feature package (`internal/<feature>`).
5. The feature package returns domain/use-case results; the HTTP adapter turns them into contract-shaped responses or RFC 9457 problem responses whose stable `code`, type, title, and status come from one closed transport catalog.
6. Transport observability is emitted at the edge: request logs, OpenTelemetry HTTP metrics exported through Prometheus, and OpenTelemetry spans use bounded route templates from the HTTP layer. HTTP metric server identity comes from configured service identity, never the caller-controlled `Host`; the OTel SDK cardinality cap remains explicit; native startup/config metrics share the same private Prometheus registry.

Current runtime note: the shipped client API is intentionally health-only.
New business endpoints must make a security decision before implementation:
public by design, protected by real OpenAPI security plus auth middleware and
401/403 Problem responses, or blocked pending a security spec. Browser CORS
remains fail-closed by default.

Operational exposure note: `/metrics` defaults to the loopback diagnostics
address `127.0.0.1:9090`. A non-loopback diagnostics bind requires a private
scrape network or a separate authenticated design.

Public ingress note: the service binds `http.addr` and does not gate its own
reachability. The deployment platform — firewall, security group, network
policy, or service mesh — owns ingress admission, because it is the only layer
that observes every connection attempt. The startup summary records `app.env`
and `http.addr` so the effective exposure is visible on every boot.

### Startup/Shutdown Path

1. `cmd/service/main.go` delegates to `bootstrap.Run`.
2. Bootstrap parses config flags, creates the signal-aware root context, initializes baseline metrics, and loads the immutable config snapshot through `internal/config`.
3. Bootstrap reconfigures structured logging from the validated config, initializes local OTel-to-Prometheus metrics and optional tracing with one service resource in fail-open mode, validates explicit public-ingress admission, and probes enabled dependencies.
4. The HTTP runtime may begin serving while startup admission is still running, but external `/health/ready` stays not ready until startup admission marks the process ready.
5. Readiness is guarded by startup admission and `internal/health.Service`, which runs enabled dependency probes sequentially under one readiness timeout.
6. `/health/live` remains process-only; external dependency checks, startup admission, and drain belong in readiness.
7. On shutdown, bootstrap marks the service as draining, flips readiness off, waits the configured propagation delay, gracefully shuts down the HTTP server, and flushes telemetry inside the process-grace budget.

The lifecycle baseline is: config and dependency validation happen before accepting traffic, and shutdown is coordinated from the bootstrap layer rather than from handlers or feature services.

### Background / Async Extension Path

The baseline repository does not ship an always-on background worker runtime.

When a task introduces async work, keep the extension path stable:
1. Put business behavior in `internal/<feature>`.
2. Put queue, scheduler, database, or external-system mechanics in `internal/infra/<integration>`.
3. Own lifecycle, config loading, telemetry, and graceful shutdown in a composition root under `cmd/<binary>/` or another explicit bootstrap entrypoint.

Preferred rule: if the workload has a distinct lifecycle or scaling model, add a new binary instead of hiding durable background loops inside HTTP handlers.

## System Neighbors

The runtime flows above stop at this service's edges. This section records what is on the other side of those edges, and it is the inventory that boundary-crossing diagnosis and design consume before reading any code. This record is repository-owned: [Template Sync](./template-sync.md) never mirrors it, so each derived service maintains its own.

The template itself ships no neighbor. A derived service replaces the placeholder row with one row per system it exchanges work with — inbound callers and clients, outbound providers, brokers, jobs, and managed dependencies.

| Neighbor | Role on the failing path | Canonical contract source | Local checkout or clone | Runtime evidence | Owner |
| --- | --- | --- | --- | --- | --- |
| _(none in the template)_ | inbound caller, outbound provider, broker, job, or managed dependency | its repository, generated contract, published spec, or live contract endpoint | path or clone URL | the log/trace query, dashboard, or command that reads it | accountable team or person |

Rules that stay stable:

- A neighbor belongs here as soon as this service calls it, is called by it, or shares durable state with it — not only when a task changes it.
- Record where its current contract actually lives, not a local copy or generated client standing in for it.
- Record the concrete way to read its runtime evidence, plus the correlation field that joins it to this service's `X-Request-ID` and W3C Trace Context. Without that join, cross-service evidence cannot be gathered for one unit of work.
- A neighbor discovered during diagnosis is added here as part of closing that diagnosis.
- Record access paths and query shapes only. Credentials, tokens, and customer data never belong in this file.

## Extension Seams

Use these seams when extending the repository:

- New HTTP capability: first consume the approved `spec.md` behavior/contract delta plus any needed system/integration contract decisions; then update `api/openapi/service.yaml`, regenerate `internal/openapi`, add use-case logic in `internal/<feature>`, and wire handlers/routes in `internal/infra/http`. Do not use OpenAPI edits, generated code, handlers, or tests to invent resource, status, error, retry, async, freshness, or compatibility semantics.
- New persistence flow: add a deterministic migration under `migrations`, add SQLC query sources under `internal/infra/postgres/queries`, regenerate `internal/infra/postgres/sqlcgen`, add a hand-written Postgres repository that maps generated rows into feature-facing types, add a feature-owned port only if needed, then wire the concrete adapter in `cmd/service/internal/bootstrap`.
- New integration adapter: add it under `internal/infra/<integration>`; add a feature-owned contract only if `internal/<feature>` needs inversion over the concrete adapter; wire concrete dependencies in `cmd/service/internal/bootstrap`. For ordinary provider-specific clients, start with `net/http`. When the repository was initialized with `OUTBOUND_HTTP=bounded`, reuse `internal/infra/httpclient` for fixed-authority transport safety while keeping authentication, operation budgets, retry eligibility, provider errors, and generated clients in the provider adapter. Credentials belong in headers; query-string authentication requires a separate telemetry-disclosure design. When the adapter calls another microservice, first verify the provider's current contract from its repository, generated contract, published spec, or live contract endpoint, then record the source used in the owning spec/design/tasks proof. Before enabling a runtime dependency, define config keys and secret-source policy, platform egress policy, criticality, retry and timeout budget, readiness participation, cleanup on partial initialization, low-cardinality metrics labels, and bootstrap tests.
- New outbound target: fixed targets must declare source, timeout, redirect policy, and DNS/IP-class behavior before bootstrap wiring; the deployment owns network-level egress enforcement. Dynamic or user-controlled URLs require a separate security design.
- New durable schema behavior: evolve `migrations/` first, then keep adapter or generated access code derived from that schema.
- New executable surface: add `cmd/<binary>/main.go` with its own bootstrap path and reuse feature/infra packages instead of duplicating logic.
- New non-HTTP contract surface: `api/proto/` is the reserved source-of-truth location for protobuf contracts when that runtime is introduced.

## Related Repository Docs

Use these docs instead of duplicating their detail here:

- Structure and placement rules: [Project Structure & Module Organization](./project-structure-and-module-organization.md)
- Config sources, precedence, and secret policy: [Configuration Source Policy](./configuration-source-policy.md)
- Local commands, validation commands, and generation flows: [Build, Test, and Development Commands](./build-test-and-development-commands.md)
- CI gates and production-readiness expectations: [CI/CD Production-Ready Checklist](./ci-cd-production-ready.md)
- Task-local workflow and artifact sequencing: [Spec-First Workflow](./spec-first-workflow.md)
