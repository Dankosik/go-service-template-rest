# Signal Contract Matrix

## Behavior Change Thesis
When loaded for a broad observability section or telemetry contract, this file makes the model write an operator-decision matrix per changed runtime path instead of likely mistake "add logs, metrics, and traces" lists with no owner, proof, or rejection logic.

## When To Load
Load this when the symptom is broad cross-signal coverage: "observability section", "signal matrix", "telemetry contract", "spec.md-ready guidance", or a change touching several runtime paths.

Do not load this as the default for a narrow label, SLO, baggage, DLQ, log privacy, or probe question. Use the narrower reference instead.

## Decision Rubric
- Use one row per changed runtime path, not one section per signal type.
- Name the operator decision first: page, rollback, degrade, retry, redrive, isolate dependency, prove recovery, or investigate a specific entity.
- Pick the cheapest primary signal for that decision; add secondary logs/traces only when they answer a distinct follow-up question.
- Keep metrics bounded and aggregateable. Put request IDs, trace IDs, message IDs, user IDs, account IDs, and sample entity IDs in logs/traces only when privacy policy allows them.
- Include owner, dashboard/runbook or investigation entry point, and validation proof for every critical row.
- State one plausible rejected option when the wrong choice is tempting, such as log-scrape alerting, raw-path labels, or a generic dashboard.

## Imitate
| Runtime path | Operator decision | Selected signal contract | Reject |
| --- | --- | --- | --- |
| `POST /v1/payouts` | Detect user-impacting create failures and decide rollback or async degrade. | Route-template request duration/error metric, server span named from route template, one completion log with bounded `outcome` and `error.type`, trace/request IDs in logs only, SLO panel and runbook owner. | Raw path, tenant ID, request body, request ID, or trace ID as metric labels. |
| Fraud gRPC call | Decide whether dependency timeouts need fallback or provider escalation. | Client latency/outcome by bounded dependency and method, client span with deadline status, fallback handoff log, dependency dashboard linked from payout alert. | Merging dependency attempts with final user-visible request failure. |
| Reconciler run | Decide whether drift is growing and whether manual repair/redrive is needed. | Run duration, drift found/repaired/unresolved counters, oldest unresolved drift age, run span with partner fan-out, run summary log with privacy-safe sample IDs. | A run log with IDs but no age or unresolved-count metric. |

Copy the pattern: changed path -> operator decision -> bounded primary signal -> forensic pivot -> owner/proof -> rejected tempting shortcut.

## Reject
- A "full coverage" plan that asks for logs, metrics, traces, dashboards, and alerts for every path without saying which operator decision each signal supports.
- A dashboard containing every exported metric with no alert, runbook, or first-response path.
- A single `success=false` field for a flow where `202 Accepted`, retry admission, and downstream completion are separate promises.
- Treating DB timeout attempts and final request failure as the same metric.

## Agent Traps
- Filling the matrix with implementation mechanics instead of operator decisions.
- Letting the matrix duplicate every narrow reference. Route deep cardinality, SLO, correlation, privacy, async, or diagnostics questions to their specific file.
- Adding a signal because it is available from the library, not because someone will use it during an incident or support investigation.
- Omitting validation proof after naming a signal contract.

## Validation Shape
- Verify each critical path has at least one bounded metric for alert/SLO math or a recorded reason why it cannot.
- Verify each alert path has an owner, dashboard or query entry point, runbook action, and event-floor or noise control where needed.
- Verify each high-cardinality forensic field stays out of metric labels and has a privacy/retention rule when logged.
