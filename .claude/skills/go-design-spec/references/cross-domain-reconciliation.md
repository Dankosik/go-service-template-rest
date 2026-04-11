# Cross-Domain Reconciliation

## Behavior Change Thesis
When loaded for conflicting specialist outputs or design artifacts, this file makes the model choose an explicit integrated resolution with rejected alternatives and proof obligations instead of likely mistakes like vague compromise, specialist override, or scope changes hidden inside design.

## When To Load
Load this when the symptom is cross-domain tension: architecture versus API contract, data ownership versus reliability behavior, security fail-closed rules versus fallback/degradation, observability versus new runtime paths, delivery versus migration shape, or QA proof obligations versus accepted design risks.

Use this to reconcile already-surfaced specialist outputs. Do not use it to replace specialist decisions or to make final `spec.md` decisions inside design.

## Decision Rubric
- Identify the conflict as two or more claims that cannot all remain true.
- Preserve final `spec.md` decisions unless the conflict proves they must be reopened.
- Select one integrated option and state what each affected domain must preserve.
- Record rejected options with concrete rejection reasons, not taste.
- Convert accepted risk into validation and reopen conditions.
- Escalate when a domain claim is unevidenced or the only coherent option changes user-visible behavior, scope, or accepted risk.

## Imitate

Reconciliation note with conflict, resolution, rejected options, and planning consequence:

```markdown
## Reconciliation: async export completion

Conflict:
- API candidate says `POST /exports` returns final export metadata.
- Reliability/data notes require durable background execution because export duration can exceed request budgets.

Resolution:
- Keep create request synchronous only through durable job creation.
- Return operation/job state from the approved REST contract.
- Worker owns execution, retry, and stuck-job reconciliation.
- Read endpoint discloses job state and freshness.

Rejected:
- Keep request open until export completes; rejected because the runtime sequence would exceed deadline and restart safety.
- Fire-and-forget worker without durable job state; rejected because clients need recovery and support needs reconciliation.

Planning consequence:
- Planning may sequence contract update, app job state, worker runtime, and validation, but must not revisit the sync/async decision unless the spec is reopened.
```

Compact cross-domain matrix:

```markdown
| Domain | Integrated consequence |
| --- | --- |
| Architecture | New worker runtime with explicit bootstrap owner. |
| API | `202` plus status read; no finality claim at submit time. |
| Data | Durable job table is authoritative for job state. |
| Security | Status read uses existing auth and tenant filter before state exposure. |
| Reliability | Worker owns retry, stuck detection, and shutdown. |
| Observability | Correlate submit request, worker job ID, retries, and final state. |
| QA | Cover submit, status read, worker retry, stuck-job reconciliation, and tenant isolation. |
```

## Reject

Generic compromise:

```markdown
Use async internally but make the API feel synchronous.
```

Why it is bad: it hides the consistency contract and pushes correctness ambiguity to implementation.

Specialist override:

```markdown
Ignore the data concern; the design bundle says a cache is enough as the source of truth.
```

Why it is bad: the integrator reconciles specialist decisions; it does not replace a data source-of-truth decision.

Scope rewrite:

```markdown
Because security is complex, remove tenant isolation from MVP.
```

Why it is bad: scope and acceptance changes belong back in `spec.md` and the orchestrator.

## Agent Traps
- API promises immediate final state while data/reliability design is eventually consistent.
- Security requires fail-closed behavior while reliability proposes fallback to a weaker authorization path.
- Observability mentions a new async path but has no correlation, retry, or DLQ visibility.
- Delivery requires mixed-version compatibility while data design uses a one-shot contract migration.
- QA proof expectations do not cover the accepted cross-domain risk.
- Architecture delegates process ownership to "the broker" or "the database" with no app/runtime owner.
- Rollback notes claim full reversibility after a non-compensable state or contract change.

## Validation Shape
Before handoff, each reconciled tension should name the selected option, rejected alternatives, affected domains, proof obligations, and reopen conditions. If the resolution changes scope or user-visible behavior, it is not design-ready until specification is reopened.

## Escalation Rules
- Escalate to the orchestrator when reconciliation would change final scope, non-goals, accepted risk, or user-visible behavior.
- Escalate to the relevant specialist when a domain claim lacks evidence or conflicts with another domain in a planning-critical way.
- Reopen specification when no selected option can satisfy the approved scope and constraints.
- Keep design blocked when a contradiction changes correctness, ownership, security posture, rollout safety, or validation proof.
- Hand off to planning only after the design records the selected option, rejected options, trade-offs, proof obligations, and reopen conditions.

## Repo Pointers
- `docs/spec-first-workflow.md`: `spec.md` owns decisions, `design/` owns task-local technical context, and `plan.md` consumes approved `spec.md + design/`.
- `docs/repo-architecture.md`: stable ownership and runtime path constraints to preserve during reconciliation.
- Adjacent specialist skills such as `go-architect-spec`, `api-contract-designer-spec`, `go-data-architect-spec`, `go-security-spec`, `go-reliability-spec`, `go-observability-engineer-spec`, and `go-qa-tester-spec` for domain-owned decisions.
