---
name: go-observability-engineer-spec
description: "Design observability-first specifications for Go services in a spec-first workflow. Use when planning or revising telemetry behavior before coding and you need explicit logs/metrics/traces correlation rules, SLI/SLO and error-budget policy, debuggability contracts, async observability requirements, and telemetry cost guardrails. Skip when the task is a local code fix, endpoint-level API payload design, physical SQL schema/migration scripting, CI/container setup, or low-level implementation tuning."
---

# Go Observability Engineer Spec

## Purpose
Create a clear, reviewable observability specification package before implementation. Success means telemetry contracts and operational readiness requirements are explicit, defensible, and directly translatable into implementation and tests.

## Scope And Boundaries
In scope:
- define telemetry signal contract for API/client/DB/worker/job paths (logs, metrics, traces)
- define correlation and propagation rules across sync and async boundaries
- define SLI/SLO profile, error-budget policy, and burn-rate alerting expectations
- define paging vs ticket routing and runbook-readiness requirements
- define debuggability and diagnostics contract (`/livez`, `/readyz`, `/startupz`, admin/debug exposure, crash diagnostics, telemetry flush on shutdown)
- define telemetry cost and safety controls (cardinality budgets, sampling, retention, redaction/sanitization)
- define async observability obligations for retries, DLQ, lag/backlog, and reconciliation
- produce observability deliverables that remove hidden "decide later" gaps

Out of scope:
- baseline service/module decomposition and ownership topology decisions
- endpoint-level API request/response payload and error-body schema design
- physical SQL schema design, DDL details, and migration script authoring
- full secure-coding and authorization control catalog outside telemetry/privacy surface
- CI/CD pipeline design and container/runtime hardening setup
- detailed resilience implementation tuning (retry/backpressure/circuit knobs) as primary domain
- implementation-level instrumentation code and vendor-specific dashboard/alert syntax
- benchmark/profile-driven performance optimization plans

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 0: seed observability assumptions/blockers in `80`; define baseline observability concerns for `50`
   - Phase 1: define observability constraints that must shape `20` and `60`
   - Phase 2 and later: maintain `50/80/90` and update impacted `30/40/55/70`
3. Load context using this skill's dynamic loading rules and stop when four observability axes are source-backed: signal contract, SLI/SLO+alert policy, diagnostics/debugability, telemetry-cost+async visibility.
4. Normalize operational questions first: what must be diagnosable in first response and which signals answer those questions.
5. Select signal types by operational question and cost: use metrics for trend/alert conditions, traces for causal path and latency breakdown, and logs for high-cardinality diagnostics.
6. For each nontrivial observability decision, compare at least two options and select one explicitly.
7. Assign decision ID (`OBS-###`) and owner for each major observability decision.
8. Record trade-offs and cross-domain impact (architecture, API, data, security, reliability, delivery).
9. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate them in the current pass or move them to `80-open-questions.md` with owner and unblock condition.
10. If uncertainty blocks a safe or measurable decision, record it in `80-open-questions.md` with concrete next step.
11. Keep `50-security-observability-devops.md` as primary artifact and synchronize observability implications in impacted artifacts.
12. Verify internal consistency: no contradictory signal definitions, no unbounded telemetry assumptions, and no critical observability decisions deferred to coding.

## Observability Decision Protocol
For every major observability decision, document:
1. decision ID (`OBS-###`) and current phase
2. owner role
3. context and operational question
4. options (minimum two)
5. selected option with rationale
6. at least one rejected option with explicit rejection reason
7. signal contract impact (logs/metrics/traces/correlation)
8. SLI/SLO, alerting, and runbook impact
9. telemetry cost/safety impact (cardinality, sampling, retention, privacy)
10. async visibility impact (retry, DLQ, lag, reconciliation)
11. reopen conditions, affected artifacts, and linked open-question IDs (if any)

## Output Expectations
- Primary artifact:
  - `50-security-observability-devops.md` with mandatory observability sections:
    - `Signal Contract And Correlation Rules`
    - `SLI/SLO And Error-Budget Policy`
    - `Alert Routing And Runbook Requirements`
    - `Debuggability And Diagnostics Contract`
    - `Telemetry Cost And Safety Guardrails`
    - `Async Observability Requirements`
- Required core artifacts per pass:
  - `80-open-questions.md` with observability blockers/uncertainties
  - `90-signoff.md` with accepted observability decisions and reopen criteria
- Conditional alignment artifacts (update when impacted):
  - `30-api-contract.md`
  - `40-data-consistency-cache.md`
  - `55-reliability-and-resilience.md`
  - `70-test-plan.md`
- Conditional artifact status format for `30/40/55/70`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `OBS-###`
  - for `updated`, list changed sections and linked `OBS-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit signal and operations semantics.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when four observability axes are covered with source-backed inputs: signal contract, SLI/SLO+alert policy, diagnostics/debugability, telemetry-cost+async visibility.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only when unresolved decisions require them
- `docs/llm/operability/10-observability-baseline.md`
- `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
- `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`

Load by trigger:
- API boundary telemetry semantics (`X-Request-ID`, propagation headers, idempotency/retry visibility):
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Sync/async architecture and distributed consistency implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Data/cache observability implications:
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Security/privacy constraints affecting telemetry:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Delivery and release-gate implications:
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
- SLI definitions are explicit (`good`, `total`, exclusions, window) and linked to alerting/budget policy.
- Cardinality, sampling, retention, and redaction controls are explicit and bounded.
- Sync and async correlation requirements are explicit and testable.
- Every `[assumption]` is either source-validated in the current pass or tracked in `80-open-questions.md` with owner and unblock condition.
- Observability blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- Impacted `30/40/55/70` artifacts have explicit status with decision links and no contradictions.
- No hidden observability decisions are deferred to coding.

## Anti-Patterns
- replacing signal contracts with generic observability statements
- defining SLO/alerts without explicit SLI numerator and denominator semantics
- allowing unbounded labels or high-cardinality identifiers in metric dimensions
- leaving retries/DLQ/lag behavior without correlation and measurable diagnostics
- treating telemetry as "collect everything" without cost controls
- duplicating security/devops/reliability decisions without observability rationale
- deferring critical observability uncertainties to coding without open-question tracking
