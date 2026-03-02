# SLI, SLO, alerting, and runbooks instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing or changing SLI/SLO definitions, error budgets, or burn-rate alerts
  - Defining paging vs ticketing rules, escalation policies, or alert routing
  - Setting dashboard structure, runbook requirements, or incident triage workflow
  - Defining service readiness criteria before production rollout
  - Reviewing release/degradation decisions tied to reliability signals
- Do not load when: The task is documentation-only and does not change runtime operability behavior

## Purpose
- This document defines repository defaults for SLI/SLO policy, error budget usage, alerting, and runbook quality.
- Goal: convert reliability targets into explicit operational decisions for releases, degradation, and incident response.
- Defaults here are mandatory unless an ADR explicitly approves an exception.

## Baseline assumptions
- SLO compliance window default is rolling `28d`.
- Every SLI is a ratio with explicit `good` and `total` event definitions.
- Error budget is `1 - SLO`; burn rate is actual bad event rate divided by budgeted bad event rate.
- Alerting is symptom-first (user-impacting reliability), not implementation-noise-first.
- Burn-rate alerting uses multi-window rules to reduce noise and catch both fast and slow budget burn.
- All paging alerts must be actionable, owner-routed, and linked to a runbook.

## Required inputs before changing SLI/SLO or alerting
Resolve first. If unknown, apply defaults and document assumptions.

- Service class: `api`, `worker`, `async_consumer`, or mixed
- Criticality tier: `tier_0` (user-facing critical), `tier_1` (internal/important), `tier_2` (best-effort)
- Unit of work semantics for async paths (message/job key, retry model, terminal failure definition)
- Excluded traffic for SLI denominator (health checks, synthetic probes, non-user control paths)
- Dependency profile (DB, cache, queue, upstream APIs) and known saturation points
- Existing runbooks, dashboards, and alert receivers that may break on metric/label changes

## Default SLI/SLO profiles

### Default policy for all service classes
- The SLI formula must be documented as:
  - `good_events`
  - `total_events`
  - `sli_ratio = good_events / total_events`
- Denominator must exclude internal probe endpoints (`/livez`, `/readyz`, `/startupz`, `/metrics`) unless explicitly required.
- SLO targets must include one availability-like objective and one latency/freshness objective.
- For low-traffic services, include event floors to avoid noisy burn-rate paging.

### API services (synchronous request/response)

Default SLI set:
- Availability SLI:
  - `total`: all routed requests except excluded control endpoints
  - `bad`: `5xx`, handler timeouts, gateway timeouts attributable to this service path
  - `good`: `total - bad`
- Latency SLI:
  - measured from request start to response write completion
  - computed on successful service handling (`2xx-4xx`) to avoid fast-failure distortion
  - objective format is threshold ratio (`X% <= T`), not average latency

Default SLO targets:
- `tier_0`: availability `99.9%` over `28d`
- `tier_1`: availability `99.5%` over `28d`
- `tier_2`: availability `99.0%` over `28d`
- Latency default for API:
  - `95% <= 300ms`
  - `99% <= 1s`

Measurement requirements:
- Use histogram metrics with buckets covering SLO thresholds (`0.3s`, `1s`).
- Keep route dimensions low-cardinality (route template, never raw path).

### Workers (background jobs)

Default SLI set:
- Job success SLI (logical unit-of-work based):
  - `total`: unique jobs started (by stable job key)
  - `bad`: terminal failures (`max_retries_exhausted`, `dlq`, `timeout_terminal`)
  - `good`: terminal successes
- End-to-end latency SLI:
  - from job enqueue time to terminal success
  - objective format: threshold ratio (`X% <= T`)

Default SLO targets:
- `tier_0`: success `99.9%` over `28d`
- `tier_1`: success `99.5%` over `28d`
- `tier_2`: success `99.0%` over `28d`
- End-to-end latency defaults:
  - near-real-time workloads: `90% <= 30s`, `99% <= 5m`
  - batch-like workloads: `90% <= 15m`, `99% <= 2h`

Measurement requirements:
- Retry attempts must be tracked separately from terminal outcome.
- DLQ transitions must be counted explicitly in bad events and alerts.

### Async consumers and pipelines (event-driven read models, projections)

Default SLI set:
- Processing success SLI:
  - `total`: consumed messages/events
  - `bad`: terminal processing failures (after retry policy)
- Freshness SLI (read-path facing):
  - `good`: consumed view age is below threshold when data is read
- Completeness SLI (when applicable):
  - `good`: no missing partitions/windows in processing interval

Default SLO targets:
- Processing success:
  - `tier_0`/`tier_1`: `99.9%` and `99.5%` over `28d`
  - `tier_2`: `99.0%` over `28d`
- Freshness default:
  - `90% <= 1m`
  - `99% <= 10m`
- Completeness default:
  - `99%` successful windows/runs over `28d`

Measurement requirements:
- Backlog age and consumer lag are mandatory companion metrics.
- If freshness is user-visible, freshness SLO is release-gating.

## Error budget policy and decision rules

### Budget states
Compute and publish these at least hourly:
- `budget_total = (1 - slo_target) * total_events_28d`
- `budget_used = bad_events_28d`
- `budget_remaining = budget_total - budget_used`
- `budget_remaining_ratio = budget_remaining / budget_total`

Default decision states:
- `green`: budget consumption `<= 25%`
- `yellow`: budget consumption `> 25%` and `<= 50%`
- `orange`: budget consumption `> 50%` and `<= 100%`
- `red`: budget consumed `> 100%` (SLO breach)

### Release decisions linked to error budget
- `green`:
  - normal release cadence allowed
  - canary and progressive rollout policy unchanged
- `yellow`:
  - allow releases, but require canary + automated rollback guard
  - no large-risk migrations without explicit reliability sign-off
- `orange`:
  - freeze non-essential feature releases
  - allow only reliability, incident remediation, and security changes
  - require reliability action plan before next rollout window
- `red`:
  - freeze all feature releases
  - only incident mitigation/security fixes allowed until trend is back within budget policy

### Degradation decisions linked to burn and saturation
- If burn-rate paging alert is active and saturation confirms overload, enable predefined degradation modes.
- Degradation modes must be predefined per service, for example:
  - disable expensive non-critical endpoints
  - switch to stale-read path for non-critical views
  - tighten per-tenant rate limits
  - reduce async fan-out and non-critical enrichment
- Never degrade data integrity semantics (idempotency, correctness, authorization boundaries).
- Every degradation mode must have entry and exit criteria in runbooks.

## Burn-rate alerting defaults

### Multi-window burn-rate rules (default)
Apply for each SLO objective (availability/success/freshness):

- `page_fast` (high urgency):
  - condition: burn rate `>= 14.4`
  - windows: short `5m` and long `1h` must both breach
  - intent: catch incidents that can consume a large budget fraction quickly
- `page_sustained` (high urgency):
  - condition: burn rate `>= 6`
  - windows: short `30m` and long `6h` must both breach
  - intent: catch sustained reliability degradation
- `ticket_slow_burn` (non-paging):
  - condition: burn rate `>= 1`
  - windows: short `6h` and long `3d` must both breach
  - intent: create planned reliability work before breach

### Paging vs ticket policy
- Paging (`sev1`/`sev2`) is only for conditions requiring immediate human intervention.
- Ticket alerts are for slow-burn, capacity trend, and non-immediate remediation.
- A paging alert must include:
  - owner/team route
  - runbook link
  - dashboard link
  - clear symptom statement and immediate action hint

### Low-traffic guardrails (mandatory)
- Burn-rate alerts must include event floor guards.
- Default floor:
  - long window `total_events >= 100`
  - short window `total_events >= 20`
- If event floor is not met:
  - suppress page alerts
  - emit ticket based on absolute bad-event count and manual review

## Dashboard hierarchy defaults

### L0: fleet overview (executive/on-call entry)
Must show:
- SLO status (`green/yellow/orange/red`) per service
- budget remaining ratio
- active paging alerts
- deployment marker timeline

### L1: service SLO dashboard (first responder)
Must show:
- each SLI numerator, denominator, ratio
- burn-rate panels for `5m/1h`, `30m/6h`, `6h/3d`
- impacted routes/operations/consumer groups (low cardinality)
- current release version and recent config changes

### L2: dependency and saturation dashboard (diagnosis)
Must show:
- DB pool saturation and timeout/error rates
- upstream call latency/error/retry
- queue lag/oldest message age
- CPU throttling, memory pressure, in-flight concurrency

### L3: deep diagnostics dashboard (specialist)
Must show:
- trace-based latency breakdown
- error taxonomy distribution
- run-level diagnostics for workers/pipelines
- links to logs/traces/profiles

## Runbook expectations (mandatory)
Every paging and ticket alert must map to a runbook section.

Required runbook sections:
- `Symptom`: what is failing and which SLO is impacted
- `Impact`: user/business scope, affected endpoints/topics/tenants
- `Immediate checks (first 5 minutes)`: exact dashboards and queries
- `Stabilization actions`: safe degradation and traffic controls
- `Rollback/roll-forward`: deployment/config actions and criteria
- `Escalation`: who to page next and when
- `Exit criteria`: conditions to resolve incident and disable degradation
- `Post-incident actions`: mandatory follow-up items and ownership

Runbook quality rules:
- Steps must be executable without tribal knowledge.
- Commands/queries must be copy-ready and environment-specific.
- If an alert has no valid runbook, alert severity must not be paging.

## Mandatory signals for service readiness gate
A service is not production-ready if any required signal is missing.

Required signals:
- SLI numerator and denominator metrics for each declared SLO
- Histogram metrics with SLO threshold buckets for latency/freshness
- Error classification metric/log field with bounded taxonomy
- Saturation signals:
  - API: active/in-flight requests and CPU/memory pressure
  - workers/async: queue lag, oldest message age, consumer concurrency
- Dependency health and timeout/retry metrics
- Correlated structured logs with request/job/message IDs and trace IDs
- Trace propagation across sync and async boundaries
- Deployment/change markers on dashboards (`service.version`, config rollout markers)
- At least one validated paging path and one ticket path per critical SLO

## Mandatory signals for incident triage
These must be available within minutes during incident handling.

- Current burn rates for all windows and current budget remaining ratio
- Failing SLI dimension breakdown (route/operation/consumer group)
- Error type distribution and top failing dependencies
- Saturation state (CPU throttling, memory pressure, pool waits, lag/backlog)
- Recent intended changes (deploys, flags, config)
- Current degradation mode status (enabled/disabled and when)
- Recovery trend panel (is burn rate decreasing after mitigation)

If any triage signal is missing during incident response:
- treat observability gap as an incident action item
- add temporary diagnostics immediately
- create a follow-up task to make the signal permanent

## Decision rules (apply in order)
1. Classify service (`api`, `worker`, `async_consumer`) and criticality tier.
2. Define SLI numerator/denominator with explicit excluded traffic.
3. Set SLO targets from defaults; document any deviations.
4. Compute error budget and configure budget state tracking.
5. Configure multi-window burn-rate alerts with event floor guards.
6. Route alerts to paging or ticket channels by urgency/actionability.
7. Ensure each alert points to a tested runbook and L1 dashboard.
8. Define budget-state release gates and degradation triggers.
9. Validate mandatory readiness and triage signals before rollout.

## Anti-patterns to reject
- SLO defined without explicit `good`/`total` event semantics.
- Using average latency as a primary SLO instead of threshold/percentile objective.
- Burn-rate paging without low-traffic event floors.
- Paging on internal noise metrics that have no immediate user impact.
- Alerts without owner, runbook, or dashboard links.
- Treating retries as successes without measuring terminal failure and latency impact.
- Mixing raw high-cardinality dimensions (`request_id`, raw path, user IDs) into SLI metrics.
- Release policy disconnected from error budget state.
- Ad-hoc degradation actions not documented in runbooks.
- Closing incidents without confirming burn-rate recovery trend.

## Review checklist (merge gate)
- SLI definitions are explicit and testable (`good`, `total`, exclusions).
- Default SLOs are applied or deviation rationale is documented and approved.
- Error budget math and dashboard panels are implemented.
- Burn-rate alerts use multi-window rules and event floor guards.
- Paging vs ticket routing is explicit and actionable.
- Every alert has owner/team labels, severity, runbook URL, and dashboard URL.
- Dashboard hierarchy exists (L0-L3) and links are accessible.
- Runbooks include stabilization and rollback actions with exit criteria.
- Release/degradation gates are documented against budget states.
- Mandatory readiness and incident triage signals are present and validated.

## MUST / SHOULD / NEVER

### MUST
- MUST define SLI as ratio with explicit numerator/denominator semantics.
- MUST set SLO windows to `28d` by default unless explicitly justified otherwise.
- MUST use multi-window burn-rate alerts for each critical SLO objective.
- MUST distinguish paging alerts from ticket alerts by urgency and actionability.
- MUST tie release permissions to error budget consumption states.
- MUST define and test degradation modes before incidents.
- MUST require runbook + dashboard links for every operational alert.
- MUST enforce readiness and triage signal completeness before production rollout.

### SHOULD
- SHOULD keep separate SLOs for availability/success and latency/freshness.
- SHOULD classify services into tiered criticality and align SLO ambition accordingly.
- SHOULD include change markers (version/config/flags) in SLO dashboards.
- SHOULD prefer route templates and bounded labels for dimension breakdowns.
- SHOULD run periodic alert fire-drills to validate runbooks and routing.

### NEVER
- NEVER set `100%` reliability target as default.
- NEVER page on metrics that do not require immediate action.
- NEVER ship new production-critical paths without SLI metrics and alert coverage.
- NEVER rely on one dashboard level for both executive view and deep diagnosis.
- NEVER declare incident resolved without verifying burn-rate normalization.
