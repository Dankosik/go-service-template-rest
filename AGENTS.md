# Repository Guidelines

### Go core instructions

#### Mission
- Write idiomatic, production-grade Go that looks natural to experienced Go developers and works well with the Go toolchain.
- Prefer clarity, predictability, explicit control flow, and small stable APIs over clever abstractions.

#### Default posture
- Prefer the standard library unless a third-party dependency is clearly justified or explicitly requested.
- Produce code that is naturally compatible with `gofmt` and `goimports`.
- Prefer simple, readable code over framework-heavy or over-engineered solutions.
- Use real package names, functions, and APIs. Do not invent libraries or nonexistent standard library features.
- When several solutions are possible, choose the one that a typical Go code reviewer would call straightforward and idiomatic.
- Prefer compatibility-first changes: keep behavior/API backward compatible by default, prefer additive evolution, and document migration/rollout for unavoidable breaking changes.

#### Formatting and structure
- Assume `gofmt` is non-negotiable. Do not fight its output.
- Keep files, packages, and functions focused.
- Avoid dumping unrelated helpers into `util`, `utils`, `common`, or `misc`.
- Prefer small packages with a clear responsibility.
- Keep the exported surface area as small as possible.
- Use `internal/` for implementation details that should not become public API.
- Keep dependency wiring explicit in the composition root (`cmd/<service>/main.go`) and pass dependencies via constructors/functions.
- Avoid hidden runtime magic: no dependency wiring through global mutable singletons or side-effect-heavy `init()` flows.

#### Naming
- Use short, clear, lowercase package names.
- Avoid package names with underscores, mixed caps, or vague catch-all meanings.
- Avoid stutter in client code. Names should read well as `pkg.Identifier`.
- Use Go-style initialisms consistently: `ID`, `URL`, `HTTP`, `JSON`, `API`.
- Use short, consistent receiver names, usually one or two letters.
- Name booleans so they read like facts or questions, such as `isReady`, `hasNext`, or `enabled`.

#### API design
- Prefer concrete types unless an interface is clearly needed.
- Define small interfaces on the consumer side, not preemptively on the implementation side for mocking.
- Pass values by value unless mutation, shared state, or large-copy cost makes a pointer the better semantic choice.
- Do not use pointers to basic values or to interfaces just to avoid copying.
- Keep zero values useful when practical.
- Prefer composition over inheritance-style designs; do not simulate deep inheritance hierarchies through embedding.
- Introduce extension points only for proven needs. Do not pre-build abstractions for hypothetical scenarios.
- Type selection defaults:
  - Use concrete types by default for internal code and single-implementation flows.
  - Use interfaces when the consumer needs runtime-substitutable behavior or multiple implementations behind a boundary.
  - Use function types for narrow behavior injection (strategy/callback) when one operation is enough and a full interface adds noise.
  - Use generics for reusable algorithms/data structures where logic is identical across types; avoid generics for DI/runtime polymorphism and introduce them only after repeated concrete use cases.

#### SOLID/KISS/DRY/YAGNI in Go
- Apply SOLID as pragmatic heuristics, not as mandatory OOP layering.
- SRP: split by axis of change and responsibility (usually package/component level), not by class-like ceremony.
- OCP: use strategic extension points only where change is likely; a clear `switch` is often better than plugin architecture.
- LSP: interface implementations must preserve behavioral contracts (especially error semantics), not only method signatures.
- ISP: keep interfaces narrow and consumer-owned; avoid fat shared interfaces.
- DIP: high-level code depends on consumer-side abstractions; wiring stays explicit in composition root.
- KISS: choose the simplest explicit design that satisfies current requirements and is easy to read in incidents.
- DRY: remove duplicated knowledge, not merely similar-looking code; if intent may diverge, keep code separate.
- YAGNI: do not add extension points, frameworks, or generic layers before real usage proves need.
- Avoid Java-style overengineering: no interface-per-struct, no `service/manager/factory` chains without distinct responsibilities, no abstract-factory/service-locator scaffolding by default.

#### Classical patterns in Go
- Prefer these patterns when they reduce complexity:
  - Adapter via small wrappers or function adapters (for example, `HandlerFunc`-style adaptation).
  - Decorator via explicit composition/middleware chains for cross-cutting concerns.
  - Strategy via function types or tiny consumer-owned interfaces when behavior must be swapped.
  - Factory as simple `New(...)` constructors with explicit dependency wiring.
  - State via explicit state enum + `switch` for finite workflows.
- Use with caution:
  - Functional options for APIs that evolve over time; avoid if they hide lifecycle/validation or reduce clarity.
  - Builder-style flows only for truly complex initialization; prefer config structs/constructors otherwise.
- Usually harmful in Go service code:
  - Singleton/global service objects as dependency management.
  - Abstract Factory / Service Locator scaffolding without proven multi-implementation need.
  - Visitor-style hierarchies and inheritance-heavy pattern ports from Java/C#.
  - Over-embedded pseudo-inheritance that obscures behavior and ownership.

#### Control flow
- Use early returns to keep the happy path minimally indented.
- Avoid unnecessary `else` blocks after `return`.
- Keep functions short enough that the main path is easy to follow.
- Prefer explicit code over clever chaining or hidden control flow.

#### Errors
- Treat errors as values and handle them explicitly.
- Return errors for ordinary failures; reserve `panic` for programmer mistakes or truly impossible states.
- Add context to errors so the caller can understand what operation failed.
- When the caller may need the original cause, wrap with `%w`.
- Use `errors.Is` and `errors.As` instead of string comparisons.
- Write error strings in lowercase and without trailing punctuation.

#### Context
- When a function needs cancellation, deadlines, or request scope, accept `ctx context.Context` as the first parameter.
- Do not store `context.Context` inside structs.
- Do not pass a nil context. Use `context.TODO()` if no better context exists.
- If you create a derived context with cancel, timeout, or deadline, ensure the cancel function is called.
- In request flows, propagate the incoming request context instead of replacing it with `context.Background()`.

#### Slices, maps, and data handling
- Prefer nil slices over empty slices when both mean "no values" and no API contract requires otherwise.
- Do not make nil versus empty slice semantics meaningful unless an external contract requires it.
- Protect shared mutable maps with synchronization or ownership confinement.
- Be clear about ownership and mutation of passed-in data.

#### Concurrency baseline
- Never start a goroutine without a clear shutdown or completion path.
- Use channels for communication and coordination when they express ownership transfer naturally.
- Use mutexes to protect truly shared mutable state when channels would be awkward.
- Propagate cancellation through context.

#### Documentation
- Add doc comments for exported packages, types, functions, methods, constants, and variables.
- Write doc comments as complete sentences.
- Start the comment with the identifier name.
- Make examples minimal, accurate, and executable when possible.

#### Testing and quality baseline
- Write tests for nontrivial logic.
- Prefer table-driven tests when many cases share one structure.
- Use the standard `testing` package by default.
- Before finalizing nontrivial code, expect the equivalent of:
  - `gofmt` or `goimports`
  - `go test`
  - `go vet`
- Add `-race`, `staticcheck`, `golangci-lint`, and `govulncheck` when the task or environment calls for stronger validation.

#### Production readiness gates
- For behavior-changing work, production readiness requires all of: tests, observability, security controls, and graceful failure behavior.
- Observability baseline: structured logs for critical paths, metrics with bounded cardinality, and trace-context propagation across service boundaries when relevant.
- Security baseline: explicit input validation, authorization at object/action boundaries, parameterized data access, and no secret leakage in logs/errors/traces.
- Graceful failure baseline: explicit timeout/cancel/shutdown behavior, bounded and idempotency-aware retries, and no partial side effects without recovery strategy.
- Risky or incompatible changes require a rollback-safe rollout plan.
- If any mandatory gate above is unmet, the change is not ready to merge.

#### Performance baseline
- Do not optimize blindly.
- Measure first, then optimize the proven hot path.
- Prefer algorithmic and structural improvements over micro-optimizations.
- Reach for profiling and tracing before rewriting code for performance.

## Project Structure & Module Organization
- Detailed project structure, module boundaries, and folder rationale are documented in:
  - `docs/project-structure-and-module-organization.md`

## Build, Test, and Development Commands
- Detailed command reference for build, test, OpenAPI workflow, and local development is documented in:
  - `docs/build-test-and-development-commands.md`

## Coding Style & Naming Conventions
- Follow standard Go style: tabs (via `gofmt`), lower-case package names, short receiver names.
- Keep module boundaries: business logic in `internal/app`, transport details in `internal/infra/http`.
- Name tests and files by behavior: `service_test.go`, `router_test.go`, `TestServiceReadyFail`.
- Prefer explicit constructor functions (`New`) and small interfaces in `internal/domain`.

## Testing Guidelines
- Framework: Go `testing` package with table/subtests where useful.
- Run `make test` before every push; add tests for any changed behavior and edge cases.
- HTTP behavior should be tested at router/handler level (`httptest`); keep integration scenarios in `test/`.
- No fixed coverage gate is configured; maintain or improve coverage in touched packages.

## Security & Configuration Tips
- Never commit real secrets; use `.env` derived from `env/.env.example`.
- Validate config-driven behavior locally (especially `POSTGRES_DSN`, HTTP timeouts, and log level).
- Ensure security checks remain green in CI (`govulncheck`, `gosec`, Trivy).

## API, Data, and Platform Guardrails
- Keep API contract and implementation in sync; avoid silent contract drift.
- Avoid undocumented breaking API behavior changes.
- Bound request payload sizes and define idempotency expectations for retryable operations.
- Keep transaction boundaries explicit in application/use-case orchestration.
- Avoid implicit retries that can duplicate side effects.
- Outbound HTTP/RPC clients must use explicit timeouts; avoid default infinite-timeout clients in production flows.
- Do not expose debug/profiling endpoints publicly by default.

## Go Dynamic Instructions

### Source and scope
- Optional instruction files are stored in `docs/llm/go-instructions/` and must be loaded dynamically per task.
- API instruction files are stored in `docs/llm/api/` and must be loaded dynamically for REST API contract and endpoint design tasks.
- Architecture instruction files are stored in `docs/llm/architecture/` and must be loaded dynamically when architecture decomposition is in scope.
- Data modeling instruction files are stored in `docs/llm/data/` and must be loaded dynamically for SQL/data modeling, datastore decision tasks (NoSQL/columnar), schema evolution/migrations/data reliability tasks, and caching strategy/consistency/observability tasks.
- Security instruction files are stored in `docs/llm/security/` and must be loaded dynamically for secure coding standards, threat-class controls, and security review criteria in Go services.
- Operability instruction files are stored in `docs/llm/operability/` and must be loaded dynamically for observability baselines, telemetry contracts (logs/metrics/traces), correlation IDs, and instrumentation review criteria.

### Dynamic loading policy
- Always apply the core section in this file for any Go task.
- Load optional files only when the task clearly requires them.
- If files overlap, the more specific file is the decisive rule for that topic.
- Load the smallest set of optional files that fully covers the task.
- Do not load all optional files by default.

### Optional files and when to load
- `docs/llm/go-instructions/10-go-errors-and-context.md`
  - Load when: designing or revising error contracts, adding wrap/unwrap behavior, handling failures across HTTP/RPC/DB/files, or implementing context deadlines/cancellation.
  - Strong signals: `%w`, `errors.Is`, `errors.As`, sentinel or typed errors, timeout/cancel semantics, request-scoped context flow.
  - Skip when: task is pure local logic with trivial error handling and no meaningful context semantics.
- `docs/llm/go-instructions/20-go-concurrency.md`
  - Load when: code introduces or changes goroutines, channels, mutexes, wait groups, `errgroup`, worker pools, pipelines, fan-out/fan-in, or shutdown coordination.
  - Strong signals: race risk, goroutine leak risk, blocked channel operations, deadlock symptoms, bounded concurrency requirements.
  - Skip when: code remains single-threaded and has no synchronization or lifecycle concerns.
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
  - Load when: creating/scaffolding services, moving packages, changing `cmd`/`internal` boundaries, adjusting `go.mod`, or resolving import/package dependency direction.
  - Strong signals: new module decisions, import cycles, package ownership ambiguity, generated-code placement.
  - Skip when: change is a small local edit inside an already stable structure.
- `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - Load when: behavior changes require test coverage, CI quality checks are being updated, or task includes race/fuzz/bench/lint validation.
  - Strong signals: flaky tests, missing edge-case tests, deterministic test strategy, `go test`/`go vet`/lint workflow updates.
  - Skip when: task is documentation-only with no runtime behavior impact.
- `docs/llm/go-instructions/50-go-public-api-and-docs.md`
  - Load when: touching exported identifiers, library/public packages, compatibility guarantees, doc comments, or examples for external consumers.
  - Strong signals: API contract stability questions, public naming decisions, semantic versioning impact, package-level docs updates.
  - Skip when: all touched code is strictly internal and no exported surface changes.
- `docs/llm/go-instructions/60-go-performance-and-profiling.md`
  - Load when: task targets latency/throughput/CPU/memory/allocations/lock contention improvements or requires benchmark/profile-based investigation.
  - Strong signals: `pprof`, traces, benchmark comparisons, hot path identification, allocation pressure analysis.
  - Skip when: no explicit performance goal or measured bottleneck exists.
- `docs/llm/go-instructions/70-go-review-checklist.md`
  - Load when: task is code review, audit, idiomaticity cleanup, or bug/risk/regression analysis of existing Go code.
  - Strong signals: PR review framing, "find issues" requests, maintainability/correctness checklist pass.
  - Skip when: task is pure greenfield implementation without review/audit intent.
- `docs/llm/security/10-secure-coding.md`
  - Load when: implementing or reviewing security-sensitive flows including input validation, output encoding, injection prevention, SSRF controls, path traversal defense, deserialization boundaries, file upload handling, command execution policy, and `unsafe` usage.
  - Strong signals: untrusted input handling, outbound URL fetches, filesystem access with user-influenced paths, template rendering, OS command invocation, or requests for secure-by-default patterns/review gates.
  - Skip when: task is documentation-only or a pure internal refactor with no trust-boundary or runtime security behavior impact.
- `docs/llm/operability/10-observability-baseline.md`
  - Load when: defining or reviewing observability baseline requirements including structured logs, RED metrics, trace propagation, correlation IDs, and OpenTelemetry instrumentation defaults across API, clients, DB, workers, and jobs.
  - Strong signals: missing request/message correlation, unclear telemetry naming conventions, high-cardinality metrics concerns, instrumentation consistency gaps, or requests to standardize observability review gates.
  - Skip when: task is local code refactoring with no observability behavior, telemetry contract, or instrumentation impact.
- `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
  - Load when: defining or reviewing default SLI/SLO targets for API/workers/async consumers, error budget policy, burn-rate alerting, paging vs ticket routing, dashboard hierarchy, runbook standards, or release/degradation gates tied to budget consumption.
  - Strong signals: requests to set SLI/SLO defaults, tune burn-rate alerts, formalize paging policy, connect release decisions to budget state, define service readiness/incident triage signals, or remove noisy non-actionable alerts.
  - Skip when: task only changes local telemetry instrumentation and does not change SLO/alerting policy, error budget decisions, or runbook/operational readiness rules.
- `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
  - Load when: defining or reviewing production diagnostics (`/livez`/`/readyz`/`/startupz`, admin/debug endpoints, pprof, crash diagnostics), telemetry cost controls (sampling, histogram strategy, log-volume/cardinality limits, retention, privacy/redaction), or async observability contracts across queues/retries/DLQ/lag/batches/reconciliation jobs.
  - Strong signals: requests for safe debug instrumentation, probe/shutdown diagnostics policy, incident-mode telemetry escalation with TTL, preventing telemetry explosion, broker-specific lag/DLQ visibility, or trace/log/metric correlation across async workflows.
  - Skip when: task is a local change with no impact on diagnostics endpoints, telemetry cost policy, sampling/retention strategy, or async observability behavior.
- `docs/llm/api/10-rest-api-design.md`
  - Load when: designing or reviewing REST/JSON resource modeling, URI conventions, status codes, pagination/filtering, PATCH/PUT semantics, bulk operations, idempotency, ETags, long-running operations, async acknowledgement, or API error/consistency semantics.
  - Strong signals: endpoint naming debates, ambiguous HTTP method semantics, retry-safety and `Idempotency-Key` decisions, `202 Accepted` + operation resource patterns, and standardization of `application/problem+json`.
  - Skip when: task changes only internal implementation details and does not alter REST API contract behavior.
- `docs/llm/api/30-api-cross-cutting-concerns.md`
  - Load when: designing or reviewing API cross-cutting behavior including request validation/normalization, input size limits, auth/tenant context propagation, idempotency and retry semantics, rate limiting, file uploads, webhooks/callbacks, and async operation semantics.
  - Strong signals: requirements for middleware/interceptor responsibilities vs contract-level requirements, `Idempotency-Key`/`request_id` rules, `X-Request-ID`/trace propagation, `429`/`Retry-After`, upload constraints, webhook signature/replay/dedup semantics, and `202` + operation status patterns.
  - Skip when: task is purely internal refactoring with no API boundary behavior or contract-level cross-cutting impact.
- `docs/llm/architecture/10-service-boundaries-and-decomposition.md`
  - Load when: defining or changing service boundaries, decomposing by bounded contexts, deciding new service vs module in an existing service, or reviewing ownership/transaction boundaries.
  - Strong signals: microservice split/merge discussions, distributed monolith symptoms, shared database proposals, shared domain logic proposals, unclear team/data ownership.
  - Skip when: task is a local implementation detail that does not change service/module boundaries.
- `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - Load when: defining or reviewing synchronous service communication defaults, choosing REST vs gRPC/Connect, setting timeout/deadline/retry/idempotency/error/pagination rules, or designing gateway/BFF/client ownership boundaries.
  - Strong signals: request-reply design, internal vs external API contract decisions, sync-hop policy, anti-pattern analysis (chatty services, unknown retry semantics, no deadlines, cascading synchronous chains).
  - Skip when: task does not change communication style, API interaction contracts, or synchronous reliability rules.
- `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - Load when: defining or reviewing event-driven interactions, broker-based processing, producer/consumer contracts, outbox/inbox, delivery semantics, retries/DLQ/deduplication, or background workflows.
  - Strong signals: events vs commands, pub/sub vs queues, schema evolution in messages, ordering/replay assumptions, async idempotency requirements, and async observability requirements.
  - Skip when: task is fully synchronous and does not introduce or modify messaging or async workflow behavior.
- `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
  - Load when: defining or reviewing cross-service consistency in multi-step business flows, saga orchestration/choreography, compensation, idempotency key policies, reconciliation jobs, or read-model consistency.
  - Strong signals: eventual consistency contracts, cross-service invariants, 2PC/dual-write discussions, saga step/state design, race-condition controls, and projection staleness budgets.
  - Skip when: task stays within one local transaction boundary and does not change inter-service consistency semantics.
- `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
  - Load when: defining or reviewing resilience baselines (timeouts, retries, jitter, backpressure, load shedding, circuit breakers, bulkheads), graceful startup/shutdown, fallback/degradation behavior, or safe system evolution and rollouts.
  - Strong signals: cascading-failure risks, dependency-failure policies, overload behavior, graceful degradation requirements, canary/blue-green/strangler planning, feature-flag rollout controls, and rollback-safety gates tied to SLO/error-budget.
  - Skip when: task is a local code change that does not alter resilience controls, degradation behavior, or rollout strategy.
- `docs/llm/data/10-sql-modeling-and-oltp.md`
  - Load when: designing or reviewing SQL schema for CRUD/business OLTP workloads in microservices, including constraints, indexes, transaction boundaries, and data lifecycle policy.
  - Strong signals: service-owned schema, normalization vs denormalization, keys/indexes/constraints, soft delete, audit fields, optimistic locking, pagination, temporal data, multi-tenant modeling.
  - Skip when: task is non-SQL, OLAP/reporting-only, or does not change data modeling decisions.
- `docs/llm/data/20-sql-access-from-go.md`
  - Load when: implementing or reviewing SQL access code in Go, choosing `pgx`/`database/sql`/`sqlc`, or defining transaction/pooling/query-discipline rules.
  - Strong signals: repository/DAL changes, context and DB timeouts, connection pool settings, batching/bulk writes, null/scanning handling, SQL injection/N+1/chatty query risks, observability of queries.
  - Skip when: task is schema-only modeling without runtime Go SQL access changes.
- `docs/llm/data/30-nosql-and-columnar-decision-guide.md`
  - Load when: choosing datastore class for a service/workload (SQL OLTP vs NoSQL vs columnar/analytical), or reviewing access-pattern fit, partitioning, consistency trade-offs, hot partitions, and retention strategy.
  - Strong signals: document/key-value/wide-column/time-series selection, OLTP vs OLAP decision, read-model/analytical-store introduction, async ingestion, retention/downsampling, operational readiness for new engine.
  - Skip when: datastore class is fixed and task is a local implementation detail without data-architecture decision impact.
- `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - Load when: designing or reviewing zero-downtime migrations, expand-contract rollout sequence, backfills/reindexing/data verification, rollback limitations, or data reliability controls (backup/restore drills, retention, archival, PII deletion, DR basics).
  - Strong signals: mixed-version rollout risks, destructive migration concerns, schema/code compatibility windows, outbox vs dual-write decisions during migration, PITR/restore drill requirements, and migration review checklists.
  - Skip when: task has no schema evolution, migration execution, or data lifecycle/reliability impact.
- `docs/llm/data/50-caching-strategy.md`
  - Load when: deciding whether to add cache, choosing local vs distributed vs hybrid cache, or designing/reviewing cache patterns (`cache-aside`, `read-through`, `write-through`, `stale-while-revalidate`), TTL/jitter, stampede protection, and fallback behavior.
  - Strong signals: read bottleneck/hot-key analysis, staleness and consistency trade-offs, key design/tenant isolation concerns, cache serialization/memory guardrails, cache observability metrics, and cache correctness/reliability testing requirements.
  - Skip when: task is a local refactor with no cache behavior change, no read-path bottleneck, and no cache boundary/topology decision.

If one task spans multiple domains, load all matching optional files, but keep the set minimal.
