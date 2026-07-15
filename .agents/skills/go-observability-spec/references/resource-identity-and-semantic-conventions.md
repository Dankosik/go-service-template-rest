# Resource Identity And Semantic Conventions

## Behavior Change Thesis
When loaded for service identity, semantic conventions, metric/span names, instrumentation scope, or cross-signal naming drift, this file makes the model reuse stable resource and OpenTelemetry conventions or mark unstable exceptions instead of likely mistake bespoke telemetry names and inconsistent labels across logs, metrics, and traces.

## When To Load
Load this when the spec mentions `service.name`, `service.version`, deployment environment, resource attributes, instrumentation scope, semantic conventions, custom metric names, span naming, units, or a draft that names the same thing differently across signals.

## Decision Rubric
- Start from existing repo/platform conventions. If none exist, prefer stable OpenTelemetry semantic conventions for common HTTP, RPC, DB, messaging, and runtime surfaces.
- Treat resource identity as cross-signal metadata: service name, version, deployment environment, region, and instance identity belong at the resource/scope layer where the backend supports it, not copied into every metric label by reflex.
- Use low-cardinality span names: route template, RPC service/method, worker operation, job type, or reconciler name. Never include entity IDs in span names.
- Use custom domain metrics only when no standard metric answers the operator question, and give them bounded units, label taxonomy, owner, and retirement/review rule.
- Mark experimental or vendor-specific semantic conventions as a proof obligation or reopen condition when the name/status could change the implementation.
- Keep names and attributes consistent across logs, metrics, and traces so operators can pivot without translating invented vocabularies.

## Imitate
- API path: metric follows HTTP server duration convention with route template; span name is `POST /v1/payouts`; logs carry the same route template and bounded `outcome`.
  Copy cross-signal naming alignment.
- Resource identity: `service.name="payments-api"`, `service.version` from deployment version, and deployment environment attached as resource metadata; request ID stays in logs/traces only.
  Copy resource identity placement.
- Domain metric: `payout_reconciliation_unresolved_items` is acceptable when no standard metric represents drift, and labels are bounded to `partner` and `reason`.
  Copy the "custom because domain decision" rule.

## Reject
- `payoutServiceLatency`, `payments_latency`, and `http_request_seconds` all measuring the same route with different labels and no standard convention rationale.
- Span names like `GET /accounts/123/payouts/po_456`.
- Copying `service.version`, `service.instance.id`, trace ID, request ID, and tenant ID into every metric label for joinability.
- Treating experimental semantic conventions as stable without a standards-status note.
- Creating a custom metric because it is shorter than the standard name while losing backend correlation.

## Agent Traps
- Assuming "OpenTelemetry" means any name is fine if it is exported through OTel.
- Forgetting that resource attributes can still increase backend cardinality when copied into metric labels.
- Mixing service, component, dependency, and tenant identity in one `service` label.
- Renaming standard attributes to local synonyms and making dashboards/queries harder to share.
- Using vendor dashboards as the source of truth when repo or semantic-convention names should drive the contract.

## Validation Shape
- Check each common surface for an existing stable semantic convention or a recorded custom-metric rationale.
- Check resource identity is present across signals without reflexively multiplying metric labels.
- Check logs, metrics, and traces use the same bounded vocabulary for route, dependency, operation, outcome, and error type.

## Canonical Verification Pointer
Use current OpenTelemetry semantic-convention and resource docs when convention stability, attribute names, or units affect the spec.
