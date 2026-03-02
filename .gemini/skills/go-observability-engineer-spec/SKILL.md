---
name: go-observability-engineer-spec
description: "Design observability-first specifications for Go services in a spec-first workflow. Use when planning or revising telemetry behavior before coding and you need explicit logs/metrics/traces correlation rules, SLI/SLO and error-budget policy, debuggability contracts, async observability requirements, and telemetry cost guardrails. Skip when the task is a local code fix, endpoint-level API payload design, physical SQL schema/migration scripting, CI/container setup, or low-level implementation tuning."
---

# Go Observability Engineer Spec

## Purpose
Create a clear, reviewable observability specification package before implementation. Success means telemetry contracts, SLO policy, incident diagnostics, and telemetry-cost controls are explicit, defensible, and directly translatable into implementation and test obligations.

## Scope And Boundaries
In scope:
- define observability contracts for logs/metrics/traces/correlation across API, clients, DB, workers, and jobs
- define SLI/SLO formulas, error-budget policy, burn-rate alerting, and paging vs ticket routing
- define runtime diagnostics contracts (`/livez`, `/readyz`, `/startupz`, admin/debug endpoints, crash diagnostics)
- define telemetry cost controls (cardinality, histogram strategy, sampling, retention, redaction)
- define async observability requirements (retry/DLQ/lag/backlog/reconciliation correlation)
- define observability acceptance obligations for `70-test-plan.md` and runtime verification
- synchronize observability implications across impacted spec artifacts
- produce observability deliverables that remove hidden "decide later" gaps

Out of scope:
- primary ownership of service decomposition and architecture boundaries
- endpoint/resource payload design and HTTP semantics as primary domain
- primary ownership of SQL schema design, DDL, and migration scripting mechanics
- primary ownership of authn/authz model design and threat mitigation architecture
- primary ownership of CI/container platform implementation details
- implementation-level coding of instrumentation, dashboards, or alert-rule syntax in specific vendors
- primary ownership of reliability policy design (timeouts/retries/backpressure/degradation)

## Hard Skills
### Observability Spec Core Instructions

#### Mission
- Protect production diagnosability and rollout safety by making observability behavior explicit before coding.
- Turn telemetry from advisory guidance into enforceable contracts with owners, thresholds, and acceptance criteria.
- Prevent incident blindness and telemetry cost explosion by default.

#### Default Posture
- Treat observability coverage for changed runtime paths as blocking by default.
- Use evidence-first design: no observability decision is accepted without concrete signal contract and operational consumer.
- Prefer bounded, low-cardinality telemetry dimensions and stable semantics over ad-hoc detail collection.
- Prefer the cheapest signal that answers the operational question: metrics for trend/alerts, traces for causality, logs for high-cardinality diagnostics.
- Fail closed on missing critical observability facts by recording bounded `[assumption]` plus explicit blocker/owner.

#### Spec-First Workflow Competency
- Enforce `docs/spec-first-workflow.md` Phase 2 obligations for observability ownership.
- Keep `50-security-observability-devops.md` as primary observability artifact.
- Synchronize observability implications in `55-reliability-and-resilience.md`, `70-test-plan.md`, `80-open-questions.md`, and `90-signoff.md`.
- Update `30-api-contract.md` and `40-data-consistency-cache.md` when observability behavior depends on API/data semantics.
- Never defer critical observability contracts to coding phase.

#### Telemetry Contract Competency (Logs, Metrics, Traces)
- Require OTel bootstrap contract in composition root (`cmd/<service>/main.go`): resource identity, tracer provider, meter provider, and propagators.
- Enforce mandatory service identity fields across signals: `service.name`, `service.version`, `deployment.environment.name`.
- Enforce component-level mandatory signal coverage for:
  - API handlers
  - outbound clients
  - DB access
  - workers/consumers/producers
  - scheduled jobs/reconcilers
- Require RED metrics plus saturation/backlog signals per component class.
- Require structured logs with stable common keys and bounded `error.type` taxonomy.
- Require trace error recording and low-cardinality span names.

#### Correlation And Propagation Competency
- Require W3C Trace Context + Baggage propagation defaults.
- Require `X-Request-ID` generation/propagation for sync flows.
- Require stable `correlation_id` + `message_id` + `attempt` for async flows.
- Require correlation continuity across retries and DLQ transitions.
- For batch/fan-in async processing, require span links instead of forced single-parent lineage.

#### SLI, SLO, Error Budget, And Alerting Competency
- Require each SLI to be explicitly defined as ratio with `good_events` and `total_events` plus exclusions.
- Require default 28-day SLO window unless justified deviation is approved.
- Require service-class and criticality-aware SLO targets (API, worker, async consumer profiles).
- Require budget state model (`green/yellow/orange/red`) and budget-linked release/degradation policy.
- Require multi-window burn-rate rules with event-floor guards for low-traffic services.
- Require explicit paging vs ticket routing and owner-runbook-dashboard linkage for every alert.

#### Debuggability And Runtime Diagnostics Competency
- Require split probe semantics:
  - `/livez` for restart decision only
  - `/readyz` for traffic admission only
  - `/startupz` for startup completion only
- Require graceful shutdown observability contract: readiness fail, drain, telemetry flush, bounded exit.
- Require admin/debug endpoint isolation on separate listener; no public exposure by default.
- Require debug/pprof/expvar kill-switches and TTL-governed incident activation policy.
- Require crash diagnostics policy (`GOTRACEBACK`, crash metadata, retention/privacy constraints).

#### Telemetry Cost And Cardinality Competency
- Enforce cardinality discipline as review blocker:
  - prohibit unbounded IDs in metric labels (`request_id`, `trace_id`, `message_id`, `correlation_id`, `user_id`, raw path)
  - require bounded label vocabularies and documented justification for new dimensions
- Require histogram strategy with fixed buckets and SLO cut-points; prohibit uncontrolled bucket growth.
- Require explicit trace/log sampling defaults by environment and incident burst-mode with auto-expire TTL.
- Require explicit retention policy for metrics/logs/traces and cost impact notes for retention changes.
- Require telemetry attribute limits and monitoring of dropped/truncated attributes.

#### Async Observability Competency
- Require async trace model coverage for producer send, consumer process, retry, DLQ transition, lag growth, and reconciliation runs.
- Require mandatory async metrics for outcomes, retries, DLQ, lag/backlog/oldest-age, idempotency decisions.
- Require observable retry classification and bounded reason taxonomy.
- Require DLQ depth/age visibility and redrive observability.
- Require reconciliation run telemetry with drift/repair signals.

#### Privacy And Security Telemetry Competency
- Require pipeline-level redaction/sanitization policy for logs, traces, URL/query data, and DB statements.
- Prohibit secret/token/credential/PII leakage in telemetry by default.
- Require baggage allowlist at trust boundaries.
- Require explicit tenant-safe telemetry dimensions and prohibition of tenant/user IDs as default metric labels.
- Treat correlation metadata as observability-only; never auth/authz input.

#### Cross-Domain Observability Contract Competency
- API cross-cutting alignment:
  - idempotency and retry classification must be observable
  - request limits and overload outcomes (`429/503`) must have diagnostic coverage
  - async `202` + operation-resource flows must include operation telemetry
- Distributed/reliability alignment:
  - outbox/inbox, retries, compensation, and degradation mode transitions must emit observable state changes
  - rollback/canary gates must consume SLO/burn/saturation telemetry
- Data/cache/migration alignment:
  - DB query and pool metrics, cache outcomes/miss reasons/fallback, migration/backfill verification telemetry must be explicit
- Delivery alignment:
  - observability-impacting contract changes must include drift/gate awareness in release readiness artifacts

#### Evidence Threshold Competency
- Every major observability decision must be recorded as `OBS-###` with:
  1. owner role and phase
  2. operational question answered
  3. at least two options
  4. selected option and rejection rationale for at least one alternative
  5. signal contract deltas (logs/metrics/traces/correlation)
  6. cardinality/cost impact and controls
  7. SLI/SLO/burn/alert/runbook impact
  8. cross-domain impact on API/data/security/reliability/delivery
  9. verification obligations (`70-test-plan.md` + runtime checks)
  10. reopen conditions and linked blockers

#### Assumption And Uncertainty Discipline
- Mark missing critical facts as bounded `[assumption]`.
- Resolve assumptions in current pass when possible; otherwise track in `80-open-questions.md` with owner and unblock condition.
- Never hide uncertainty in generic wording.

#### Review Blockers For This Skill
- Missing telemetry contract for any changed critical runtime path.
- High-cardinality or unbounded metric dimensions without approved exception.
- Missing SLI/SLO semantics (`good/total`, exclusions) for declared critical objectives.
- Burn-rate paging without event floors or without actionable runbook/dashboard linkage.
- Async flows with retries/DLQ but no correlation continuity or lag/depth observability.
- Public exposure of debug endpoints or absent shutdown telemetry-flush contract.
- Telemetry changes that risk secret/PII leakage without sanitization controls.
- Critical observability decisions deferred to coding without blocker tracking.

## Working Rules
1. Determine current phase and target gate from `docs/spec-first-workflow.md`.
2. Set pass scope by affected runtime paths and service class (`api`, `worker`, `async_consumer`, `job_runner`, or mixed).
3. Load context using this skill's dynamic-loading policy.
4. Build/update `OBS-###` decision register for each nontrivial observability decision.
5. Keep `50-security-observability-devops.md` as primary artifact and update mandatory sections.
6. Synchronize required implications into `55/70/80/90` and conditional `20/30/40/60` artifacts.
7. Validate internal consistency across signals, SLO policy, alert routing, and rollout gates.
8. Convert unresolved critical uncertainty into `80-open-questions.md` with owner and unblock condition.
9. Do not replace domain-specific ownership of neighboring skills; record cross-domain impact and handoff explicitly.
10. Close pass only when blockers are resolved or explicitly tracked with accountability.

## Output Expectations
- Response format:
  - `Decision Register`: accepted `OBS-###` decisions with rationale and trade-offs
  - `Signal Contract Matrix`: logs/metrics/traces/correlation requirements by component
  - `SLO And Alert Policy`: SLI formulas, budget states, burn-rate windows, routing policy
  - `Artifact Update Matrix`: required updates for `50/55/70/80/90` and status for impacted `20/30/40/60`
  - `Assumptions`: active `[assumption]` items and resolution path
  - `Open Blockers`: unresolved observability blockers with owner and unblock condition
  - `Sign-Off Delta`: what must be appended to `90-signoff.md`
- Primary artifact:
  - `50-security-observability-devops.md` with mandatory observability sections:
    - `Telemetry Signal Contract`
    - `SLI/SLO, Error Budget, And Burn-Rate Policy`
    - `Alert Routing, Dashboards, And Runbooks`
    - `Diagnostics And Debuggability Contract`
    - `Telemetry Cost, Cardinality, And Retention Controls`
    - `Async Correlation, Retry/DLQ, And Reconciliation Observability`
- Required core artifacts per pass:
  - `80-open-questions.md`
  - `90-signoff.md`
- Required alignment artifacts per pass:
  - `55-reliability-and-resilience.md`
  - `70-test-plan.md`
- Conditional alignment artifacts (update when impacted):
  - `20-architecture.md`
  - `30-api-contract.md`
  - `40-data-consistency-cache.md`
  - `60-implementation-plan.md`
- Conditional artifact status format for `20/30/40/60`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `OBS-###`
  - for `updated`, list changed sections and linked `OBS-###`
- Language: match user language when possible.
- Detail level: concrete and operationally testable; avoid generic advice.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when six observability axes are source-backed: telemetry contract, correlation/propagation, SLI/SLO+budget policy, diagnostics contract, cost/cardinality controls, and async observability.

Always load:
- `docs/spec-first-workflow.md`:
  - read `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only when unresolved observability decisions require them
- `docs/llm/operability/10-observability-baseline.md`
- `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
- `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`

Load by trigger:
- API boundary/cross-cutting implications (`request_id`, idempotency, limits, async acknowledgements):
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Sync/async/distributed workflow implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Data/cache/migration observability implications:
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/50-caching-strategy.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
- Telemetry privacy and identity-boundary implications:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Release-gate and CI enforcement implications:
  - `docs/llm/delivery/10-ci-quality-gates.md`
  - `docs/ci-cd-production-ready.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in the current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- `50-security-observability-devops.md` contains all mandatory observability sections from this skill.
- Every major observability decision includes `OBS-###`, owner, selected option, and at least one rejected option with reason.
- Signal contract covers all changed runtime paths (sync and async).
- SLI/SLO formulas, budget states, burn-rate windows, and routing policy are explicit and testable.
- Diagnostics contract (`livez/readyz/startupz`, shutdown, debug endpoint isolation) is explicit and consistent.
- Telemetry cost/cardinality/sampling/retention controls are explicit and bounded.
- Async retries/DLQ/lag/reconciliation observability obligations are explicit.
- Every `[assumption]` is source-validated or tracked in `80-open-questions.md` with owner and unblock condition.
- `55/70/80/90` are synchronized and impacted `20/30/40/60` artifacts have explicit status with decision links.
- No critical observability decision is deferred to coding.

## Anti-Patterns
- generic observability statements without concrete signal contract by component
- collecting telemetry without owner, consumer, or actionable incident question
- adding unbounded IDs to metric labels or expanding histogram/cardinality without cost rationale
- defining SLO targets without explicit `good/total` semantics and excluded traffic
- burn-rate alerts without event-floor guards or without runbook/dashboard linkage
- async retries or DLQ configured but not observable via correlation and lag/depth metrics
- exposing debug endpoints publicly or enabling permanent high-detail incident mode
- treating telemetry as "collect everything forever" instead of bounded cost-aware design
- leaking tokens/secrets/PII into logs, traces, baggage, or metrics
- deferring observability-critical decisions to coding phase without explicit blocker tracking
