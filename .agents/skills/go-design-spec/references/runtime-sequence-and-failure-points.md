# Runtime Sequence And Failure Points

## Behavior Change Thesis
When loaded for a vague `design/sequence.md`, this file makes the model choose scenario-level runtime flow with failure ownership and side-effect boundaries instead of likely mistakes like happy-path arrows, hidden dual writes, or async work that still promises synchronous finality.

## When To Load
Load this when the symptom is runtime ambiguity: `design/sequence.md` needs request order, startup/shutdown flow, async or background work, retries, timeouts, idempotency, fallback, degradation, side effects, persistence, event emission, external calls, parallelism, or failure points that planning must preserve.

Do not load this to tune low-level retry constants or write implementation steps. Route those details to reliability planning or implementation after the design is approved.

## Decision Rubric
- Write named scenarios, not generic arrows.
- Include where validation stops, where policy runs, where persistence commits, and where the response is mapped.
- For each side effect, name atomicity, retry owner, idempotency requirement, and recovery path or explicitly say none.
- If async exists, disclose freshness, job or message identity, lifecycle owner, and reconciliation path.
- Keep retries and fallback out of HTTP handlers unless the design names why that owner is correct.
- Parallel branches must not write the same source of truth without an explicit coordination rule.

## Imitate

Sequence with failure points:

```markdown
## Create Order Request

1. `internal/infra/http` receives `POST /orders` through generated strict OpenAPI routing.
2. The HTTP adapter validates transport shape and maps the request to `internal/app/orders.Create`.
3. `internal/app/orders` applies use-case policy and calls the order repository through the approved app-facing contract.
4. `internal/infra/postgres` persists the order in one local transaction.
5. The HTTP adapter maps the app result to the approved REST response.

Failure points:
- Invalid request: fail at HTTP adapter with problem response; app is not called.
- Policy rejection: app returns a domain error; HTTP maps it to the approved problem shape.
- Database timeout: app receives repository failure; HTTP returns the approved transient error response; no fallback writes.
- Commit succeeds and response write fails: persisted state remains authoritative; no compensating DB write from HTTP.

Side effects:
- No async emission in this phase.
- No retries inside HTTP; retry policy is owned by clients or a later idempotency design.
```

Async sequence with lifecycle owner and reconciliation:

```markdown
## Export Job

1. HTTP adapter validates request and creates a durable export job through `internal/app/exports`.
2. The app records job state before returning `202`.
3. A worker binary owns job execution, retries, cancellation, and shutdown.
4. Worker updates job state monotonically: `queued -> running -> completed` or `queued/running -> failed`.
5. Reconciliation scans stuck `running` jobs after worker restart.

Planning must preserve the worker lifecycle owner and durable job identity. Do not hide this loop inside the HTTP handler.
```

## Reject

Happy-path-only sequence:

```markdown
HTTP -> app -> DB -> response.
```

Why it is bad: planning cannot see where validation, side effects, retries, and partial failures belong.

Sync/async ambiguity:

```markdown
The request sends a message and immediately returns the final state once the consumer updates the database.
```

Why it is bad: the design mixes async convergence with synchronous finality without a freshness or completion contract.

Hidden side effect:

```markdown
After writing the row, the repository publishes a webhook directly.
```

Why it is bad: dual writes and post-commit side effects need an explicit atomicity, retry, and ownership model.

## Agent Traps
- Sequence says async, API/design says read-after-write finality.
- Sequence adds retries but ownership map has no retry owner or idempotency key.
- Sequence includes an external call after a non-compensable local commit without compensation or forward recovery.
- Sequence moves background work into an HTTP handler while repo architecture says distinct lifecycle work should use an explicit runtime.
- Sequence says fallback is safe, but security or data notes say fallback bypasses authorization, tenant isolation, or freshness rules.
- Sequence shows parallel branches that both write the same source of truth.
- Sequence depends on exact TTL timing for correctness.

## Validation Shape
Before handoff, each critical scenario should include success order, at least the dominant failure modes, side effects, retry or no-retry ownership, and the planning constraints that must not be revisited during task breakdown.

## Escalation Rules
- Escalate to `go-reliability-spec` when timeout, retry, fallback, overload, degradation, startup, or shutdown policy is the primary open issue.
- Escalate to `go-distributed-architect-spec` when the sequence crosses service boundaries, emits events, needs outbox/inbox, or has saga/process-state concerns.
- Escalate to `api-contract-designer-spec` when sequence behavior changes HTTP status, idempotency, long-running operation semantics, webhooks, or read-after-write disclosure.
- Escalate to `go-data-architect-spec` when persistence, projection, cache, or migration sequence owns correctness.
- Reopen specification when the sequence exposes a missing user-visible outcome or acceptance rule.

## Repo Pointers
- `docs/repo-architecture.md`: request/response path, startup/shutdown path, and async extension path.
- `docs/spec-first-workflow.md`: purpose of `design/sequence.md` and artifact handoff rules.
- `.agents/skills/technical-design-session/SKILL.md`: design-session stop rule and planning handoff shape.
