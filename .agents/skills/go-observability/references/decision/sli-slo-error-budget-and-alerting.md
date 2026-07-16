# SLI/SLO, Error Budget, And Alerting

## Behavior Change Thesis
When loaded for SLI, SLO, error budget, or alerting symptoms, this file makes the model define good/total events, exclusions, event floors, owner, runbook, and proportional response instead of likely mistake raw threshold pages such as "any 5xx > 0" or dashboards without operator action.

## When To Load
Load this when the spec needs user-impacting SLIs, SLO windows, budget policy, burn-rate alerts, alert severity, low-traffic event floors, runbook/dashboard ownership, or release/degradation policy tied to budget state.

## Decision Rubric
- Start from the user or workflow promise, then define `good_events`, `total_events`, exclusions, measurement source, and window.
- Separate admission, durable async handoff, final completion, stream continuity, and freshness when they are different promises.
- Do not let fast failures improve latency health. Define failed-request handling for latency SLIs.
- Page only when budget burn is meaningful, ownership is clear, the runbook names a first action, and sparse traffic has an explicit low-traffic policy: event floor, synthetic traffic, aggregation, longer window, renegotiated SLO, or ticket-only response.
- Use ticket-only, dashboard-only, or release/degrade policy when the response is not immediate human wake-up.
- Keep dependency labels out of user-facing SLO math unless the operator response truly differs by bounded dependency.

## Imitate
- Availability SLI: `good_events = valid payout create requests completed synchronously or durably accepted for async retry when that is the product promise`; `total_events = valid payout create requests`; exclude caller validation failures.
  Copy the explicit promise boundary.
- Async completion SLI: `good_events = invoice messages processed and completion event emitted before freshness target`; `total_events = valid invoice messages delivered to the consumer`.
  Copy the final-completion/freshness split.
- Latency SLI: `good_events = successful GET /v1/payouts/{id} responses under 200 ms`; failed requests tracked separately so fast 500s do not look healthy.
  Copy the failed-request guard.
- Alert: page on multi-window burn only after the chosen low-traffic policy is satisfied; create ticket for slow budget consumption; link SLO panel, dependency panel, and trace/log queries from the runbook.
  Copy the proportional response.

## Reject
- "Pager on any 5xx > 0 in 5 minutes" for a low-QPS service.
- Average latency SLO with no percentile, threshold, or good/total event definition.
- Counting `202 Accepted` as final success when the product promise is downstream completion.
- One SLO across admin, polling, streaming, and create flows with different user promises.
- Page with no owner, no runbook, no dashboard or query entry point, and no defined operator action.

## Agent Traps
- Treating transport success as product success for async workflows.
- Adding per-tenant, per-user, per-account, per-message, or raw-error labels to SLO metrics by default.
- Splitting SLOs by every dependency and accidentally alerting on implementation details rather than user impact.
- Choosing a 28-day window by reflex when traffic, release cadence, or product promise makes it misleading.
- Writing budget policy the team cannot enforce.

## Validation Shape
- For each SLI, verify `good_events`, `total_events`, exclusions, source metric, aggregation labels, and window.
- For each paging alert, verify burn condition, event floor or low-traffic policy, owner, runbook, dashboard/query entry point, and first operator action.
- Verify async workflows do not count admission as final success unless that is explicitly the product promise.

## Canonical Verification Pointer
Use the Google SRE Workbook chapters on implementing SLOs and alerting on SLOs when burn-rate or error-budget policy details affect the spec.
