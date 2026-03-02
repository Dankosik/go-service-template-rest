---
name: go-reliability-review
description: "Review Go code changes for reliability and resilience correctness in a spec-first workflow. Use when auditing diffs or pull requests for timeout/deadline propagation, retry budget and jitter policy, backpressure and overload behavior, graceful startup/shutdown, degradation modes, and rollout/rollback safety against approved reliability contracts. Skip when designing specifications, implementing features, or performing primary architecture/security/performance/concurrency/DB-correctness reviews."
---

# Go Reliability Review

## Purpose
Deliver domain-scoped code review findings for reliability and resilience during Phase 4 review. Success means failure-path behavior remains aligned with approved reliability contracts, critical outage risks are surfaced before `Gate G4`, and spec mismatches are escalated explicitly.
Use `Hard Skills` as the normative reliability baseline for decision quality and merge-blocking thresholds; use workflow sections below for execution sequence and output protocol.

## Scope And Boundaries
In scope:
- review changed code against approved reliability contracts in `specs/<feature-id>/55-reliability-and-resilience.md`
- review timeout/deadline propagation and fail-fast behavior in changed critical paths
- review retry eligibility, bounded retry budget, and jitter/backoff behavior
- review overload containment and backpressure controls (bounded queues/concurrency, shedding semantics)
- review startup/readiness/liveness/shutdown reliability behavior
- review degradation and fallback transitions for safety and predictability
- review rollout/rollback reliability safety implications in changed code paths
- review reliability fail-path test traceability against approved `70-test-plan.md`
- produce actionable findings with exact `file:line`, impact, and minimal safe fix
- escalate spec-level conflicts through `Spec Reopen`

Out of scope:
- redesigning architecture during code review without explicit `Spec Reopen`
- editing spec artifacts in Phase 4
- performing primary-domain idiomatic/style, architecture integrity, performance evidence, concurrency mechanics, DB/cache correctness, security, QA strategy, or domain-invariant review
- blocking PRs with preference-only comments without concrete reliability impact

## Hard Skills
### Reliability Review Core Instructions

#### Mission
- Protect merge safety by finding reliability defects in changed code paths before `Gate G4`.
- Translate resilience risks into concrete, minimal, rollback-safe corrections.
- Keep review conclusions enforceable against approved reliability contracts, not reviewer preference.

#### Default Posture
- Review failure paths first (`timeout`, `retry`, `overload`, `degradation`, `shutdown`, `rollback`) and only then happy paths.
- Treat implicit or unbounded behavior as unsafe until proven bounded by contract.
- Use source-backed defaults from loaded docs unless approved spec decisions explicitly override them.
- Keep domain ownership strict: hand off deep cross-domain root causes to the corresponding review skill.
- Prefer smallest safe fix that restores contract conformance over broad redesign proposals.

#### Spec-First Review Competency
- Enforce `docs/spec-first-workflow.md` Phase 4 constraints:
  - domain-scoped findings only;
  - exact `file:line`;
  - practical fix path;
  - explicit `Spec Reopen` for spec-intent conflict.
- Treat unresolved `critical/high` reliability defects as merge blockers for `Gate G4`.
- Never modify approved spec intent implicitly through review comments.
- Never edit spec files in code-review phase.

#### Dependency Failure-Contract Competency
- For each changed critical dependency path, require explicit contract fields:
  - timeout budget;
  - retry eligibility and retry budget;
  - bulkhead/concurrency bound and queue bound;
  - fallback mode (`fail_closed`, `stale`, `defer_async`, `feature_off`, `fail_fast`);
  - circuit mode (`none`, `soft_retry_breaker`, `state_machine`);
  - observable degradation signal with reason and timing.
- Verify dependency criticality semantics remain consistent:
  - `critical_fail_closed`;
  - `critical_fail_degraded`;
  - `optional_fail_open`.
- Flag ownerless critical dependency behavior or missing rollback authority as reliability governance risk.

#### Timeout And Deadline Competency
- Every outbound call in changed paths must have explicit deadline; infinite/implicit timeout is a blocker.
- Inbound deadline propagation to downstream calls must be preserved; replacing request context with `context.Background()` in request path is a blocker.
- Default timeout controls (unless approved spec overrides):
  - interactive end-to-end budget: `2500ms`;
  - reserve: `100ms` for response write/cleanup;
  - per-hop read: `300ms`;
  - per-hop write: `1000ms`;
  - absolute per-hop cap: `2000ms`;
  - fail-fast threshold: skip downstream call if remaining budget `<150ms`.
- Preserve cancellation semantics:
  - `context.Canceled` and `context.DeadlineExceeded` must remain recognizable via `errors.Is`;
  - derived contexts must call `cancel`.

#### Retry Budget, Idempotency, And Jitter Competency
- Default retry posture is no retry; retries are allowed only for retry-safe operations and transient failures.
- Interactive default retry envelope (unless overridden):
  - max 1 retry (2 attempts total);
  - exponential backoff with full jitter;
  - base `50ms`, cap `250ms`.
- Enforce retry budget per dependency (default: retries <= `20%` of primary attempts over rolling 1 minute); when budget is exhausted, retries must fail fast.
- Never-retry classes must stay excluded:
  - validation/contract errors;
  - authn/authz failures;
  - not-found/business conflicts;
  - caller cancellation.
- Retry-unsafe operations that may be retried must enforce idempotency contract:
  - `Idempotency-Key` (or equivalent field) required;
  - same key + same payload => equivalent result;
  - same key + different payload => conflict (`409`/equivalent).
- For async handlers, require bounded retries and deterministic error classification (`retryable_transient`, `non_retryable`, `poison_payload`) with DLQ routing.

#### Backpressure, Overload, Bulkhead, And Circuit Competency
- Every queue/channel/worker lane in changed paths must be bounded; unbounded buffers or unbounded fan-out are blockers.
- Overload handling must prefer fast rejection over unbounded waiting:
  - `429` for policy/quota throttling;
  - `503` for dependency/system capacity exhaustion;
  - `Retry-After` when recovery horizon is predictable.
- Dependency isolation must be explicit:
  - bounded per-dependency concurrency lanes;
  - no shared global unbounded pool across dependencies.
- Backpressure trigger sanity:
  - when queue depth crosses high-water mark (default `80%`), optional work shedding/degraded mode should activate.
- Circuit behavior:
  - default is `soft_retry_breaker` (retry budget + bounds);
  - state-machine breaker requires explicit threshold contract and observable state transitions.

#### Startup, Readiness, Liveness, And Shutdown Competency
- Probe split must remain correct:
  - startup probe for initialization;
  - readiness for traffic admission;
  - liveness independent of external dependencies.
- Shutdown path must remain deterministic:
  - on `SIGTERM`, mark not-ready first, then drain;
  - stop accepting new traffic before dependency teardown;
  - bounded drain timeout (default `20s`);
  - Kubernetes grace period must exceed drain + preStop budget (default minimum `30s`).
- Probe anti-flap rules must be preserved (for example hysteresis instead of single-failure readiness flip).
- Shutdown outcome must be observable (logs/metrics for completion and timeout path).

#### Degradation And Fallback Competency
- Degradation model must be explicit and unchanged unless approved:
  - `normal`;
  - `degraded_optional_off`;
  - `degraded_read_only_or_stale`;
  - `emergency_fail_fast`.
- Fallback policy must match dependency criticality:
  - `critical_fail_closed` => fail closed;
  - `critical_fail_degraded` => bounded stale/deferred fallback only with invariant preservation;
  - `optional_fail_open` => disable capability and preserve core flow.
- Stale fallback defaults must remain bounded (default max staleness `5m` unless stricter contract exists).
- Deferred fallback should use explicit async acknowledgment pattern (`202` + tracking resource/id).
- Every fallback activation/deactivation must emit structured telemetry with mode and reason.
- Hidden fallback that silently changes correctness semantics is a blocker.

#### Rollout, Rollback, And Error-Budget Gate Competency
- Risky changes must preserve progressive-delivery and rollback-safety rules:
  - canary stages and soak windows must remain explicit;
  - promotion blocked while page-level burn alert is active;
  - rollback trigger and authority must be explicit.
- Budget-state release rules must remain enforceable:
  - `green`/`yellow`/`orange`/`red` states and corresponding release restrictions.
- Feature-flagged reliability controls must preserve owner, expiry, and rollback behavior.
- Reliability review must flag any change that requires manual heroics for rollback.

#### Async And Distributed Reliability Competency
- For state-change-driven event publication, outbox-equivalent atomic linkage is mandatory; cross-system dual write is a blocker.
- Consumer idempotency and dedup must remain durable:
  - unique dedup boundary (for example `(consumer_group, dedup_key)`);
  - dedup retention >= replay/redrive window.
- Ack/offset commit must happen only after durable side effects and dedup state persist.
- DLQ policy must remain explicit:
  - exhausted/non-retryable/poison cases routed to DLQ with failure context;
  - redrive only after root-cause fix, rate-limited and observable.
- For saga-style flows, require explicit pivot semantics, compensation/forward-recovery rules, and reconciliation ownership.
- Writes must not rely on stale read models to enforce hard invariants.

#### Data, Migration, And Cache Reliability Competency
- DB reliability checks:
  - all DB calls context-bounded with explicit deadlines;
  - transaction retries only for approved transient classes and retried as full block;
  - retried writes must be idempotent;
  - pool bounds and connection budget must remain explicit.
- Migration reliability checks:
  - rollout pattern stays `expand -> migrate/backfill -> contract`;
  - no destructive-first changes in active production path;
  - backfills are idempotent, resumable, throttled, and checkpointed;
  - contract phase requires verification evidence and explicit rollback limitations.
- Cache reliability checks:
  - read-cache default fail-open with timeout shorter than origin;
  - cache outage must not trigger unbounded origin storm (coalescing + bounded fallback concurrency);
  - TTL + jitter + stampede protection remain enforced for hot keys;
  - bypass switch exists for degraded cache behavior.

#### Security And Observability Interaction Competency
- Reliability controls must not weaken security boundaries:
  - no fail-open on authn/authz or hard correctness/security invariants;
  - no hidden retries of non-idempotent/security-sensitive operations.
- Enforce bounded resource controls on reliability paths (request size/time/concurrency/quotas).
- Reliability state changes must be observable with low-cardinality telemetry:
  - logs include correlation fields and bounded `error.type`;
  - metrics cover retries, fallback/degradation activation, queue/lag saturation;
  - traces propagate across sync and async boundaries.
- Correlation identifiers are observability metadata only, never auth/authz input.
- Paging-level reliability alerts must map to runbook and dashboard links.

#### Reliability Test And CI Evidence Competency
- Reliability findings must map to explicit fail-path test obligations in `70-test-plan.md`:
  - timeout/deadline propagation;
  - retry eligibility/budget/jitter;
  - overload/backpressure shedding;
  - degradation/fallback activation and recovery;
  - graceful shutdown and rollback path behavior.
- When changed code touches concurrency/failure paths, require race-evidence command path (`make test-race` or `go test -race ./...`).
- When changes affect integration boundaries, require integration evidence (`make test-integration` when applicable).
- For API-visible reliability semantics, require contract validation path (`make openapi-check` and breaking check when applicable).
- For migration-impacting reliability changes, require migration validation evidence in CI.

#### Evidence Threshold And Severity Calibration Competency
- Every finding must include:
  - exact `file:line`;
  - violated reliability contract/default;
  - concrete failure mode and blast radius;
  - smallest safe fix path;
  - explicit `Spec reference`;
  - verification command suggestion.
- Severity is assigned by outage/availability risk, not by style preference:
  - `critical/high`: outage, cascading-failure, or merge-unsafe contract breach;
  - `medium`: meaningful reliability weakness with bounded blast radius;
  - `low`: local hardening with non-blocking impact.
- Generic advice without failure impact is invalid review output.

#### Assumption And Uncertainty Discipline
- Mark missing critical facts as bounded `[assumption]` immediately.
- If required artifacts are missing, annotate `[assumption: missing-spec-artifacts]` and reduce certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated via `Spec Reopen`.
- Never hide uncertainty in vague wording.

#### Review Blockers For This Skill
- Outbound call without explicit deadline in critical path.
- Unbounded retry/queue/concurrency behavior or missing retry budget/jitter controls.
- Retry on never-retry classes (validation/auth/conflict/not-found/cancellation).
- Missing idempotency protection on retry-unsafe retried operation.
- Overload behavior without deterministic rejection/degradation semantics (`429`/`503` policy mismatch).
- Shutdown/readiness behavior that can cause data loss, request loss, restart storm, or probe flapping.
- Hidden fallback/degradation behavior that changes correctness without explicit contract.
- Rollout/rollback path with no explicit trigger, authority, or rollback-safe mechanism.
- Async flow without outbox/dedup/durable-ack ordering or without DLQ/reconciliation controls.
- Reliability-critical change with missing fail-path tests or missing validation command path.
- Spec-intent conflict left implicit instead of explicit `Spec Reopen`.

## Working Rules
1. Confirm the task is code review and identify changed reliability-sensitive scope.
2. Determine `feature-id` from review context, changed paths, or task metadata. If it cannot be identified, continue with bounded `[assumption]` and reduced certainty.
3. Load context using this skill's dynamic loading rules.
4. Apply `Hard Skills` defaults from this file; any deviation must be explicit in findings or residual risks.
5. Evaluate changed code in this order:
   - `Timeout And Deadline Conformance`
   - `Retry Budget, Eligibility, And Jitter`
   - `Overload And Backpressure Safety`
   - `Startup/Readiness/Liveness/Shutdown Correctness`
   - `Degradation And Fallback Correctness`
   - `Rollout/Rollback Reliability Safety`
   - `Reliability Test Traceability`
6. Record only evidence-backed findings and map each finding to explicit approved obligations (prefer `REL-*` decisions or explicit clauses in `55/60/70/90`).
7. Classify severity by merge safety impact (`critical/high/medium/low`) and provide the smallest safe corrective action.
8. Keep comments strictly in reliability-review domain; hand off deep cross-domain risks to the corresponding reviewer role.
9. If a safe fix requires changing approved spec intent, create `Spec Reopen` in `reviews/<feature-id>/code-review-log.md`.
10. Do not edit spec files during code review.
11. If no findings exist, state this explicitly and include residual reliability risks.
12. Run final blocker check against `Hard Skills -> Review Blockers For This Skill` before closing the pass.

## Output Expectations
- Findings-first output ordered by severity.
- Match output language to the user language when practical.
- Use this exact finding format:

```text
[severity] [go-reliability-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

- After findings, include:
  - `Handoffs`: cross-domain risks and owner skill.
  - `Spec Reopen`: `required` or `not required` with reason.
  - `Residual Risks`: non-blocking reliability risks, assumptions, or verification gaps.
- `Validation commands`: minimal command set to verify proposed fixes.
- Keep section order stable: `Findings`, `Handoffs`, `Spec Reopen`, `Residual Risks`, `Validation commands`.
- Keep all sections present; if a section is empty, write `none` and one short reason.
- If there are no findings, output `No reliability findings.` and still include `Residual Risks` and `Validation commands`.

Severity guide:
- `critical`: proven outage/cascading-failure risk, unbounded retry/queue/timeout in critical path, or unsafe shutdown/degradation/rollback behavior that makes merge unsafe.
- `high`: strong evidence of significant reliability contract mismatch likely to impact availability/SLO under expected failure conditions.
- `medium`: bounded but meaningful reliability weakness with limited blast radius.
- `low`: local reliability hardening improvement with non-blocking impact.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once all reliability review axes are assessable with code evidence and approved spec references.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4` criteria first
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- review artifacts:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `specs/<feature-id>/60-implementation-plan.md`
  - `specs/<feature-id>/70-test-plan.md`
  - `specs/<feature-id>/90-signoff.md`
  - `reviews/<feature-id>/code-review-log.md` (if present)

Load by trigger:
- Context cancellation, timeout propagation, and error-contract semantics:
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Goroutine lifecycle, channels, bounded queues, worker pools, shutdown coordination:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- API-visible reliability semantics (`429/503`, `Retry-After`, idempotency/retry behavior, async `202` patterns):
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Distributed/async workflow reliability implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Data/cache consistency implications for retries, fallback, or reconciliation:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Security and observability interaction for fail-open/fail-closed and reliability signals:
  - `specs/<feature-id>/50-security-observability-devops.md`
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
- CI/release gate context when rollout safety is impacted:
  - `docs/llm/delivery/10-ci-quality-gates.md`
- Test evidence and command baseline:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`

Conflict resolution:
- Approved decisions in `specs/<feature-id>/90-signoff.md` override generic guidance unless `Spec Reopen` is raised.
- The more specific document is the decisive rule for that topic.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- If required spec artifacts are unavailable, mark `[assumption: missing-spec-artifacts]` and reduce certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated as `Spec Reopen`.

## Definition Of Done
- Review output stays within reliability-review domain boundaries.
- Findings are evidence-backed and use exact `file:line` references.
- Every finding has impact, minimal fix path, spec reference, and verification command guidance.
- All `critical/high` reliability findings are either resolved or clearly escalated.
- No spec-level mismatch remains implicit.
- Cross-domain root causes are handed off explicitly instead of being absorbed by this skill.
- No active item from `Hard Skills -> Review Blockers For This Skill` remains unresolved.
- If no findings, output explicitly states `No reliability findings.` and includes residual risk and validation-command notes.

## Anti-Patterns
Use these preferred patterns to avoid anti-pattern drift:
- define reliability issues with explicit failure impact, not generic advice
- require bounded timeout/retry/queue behavior in critical paths
- keep reliability-domain ownership explicit and hand off deep cross-domain issues
- prefer the smallest safe correction before broader redesign proposals
- escalate spec-intent conflicts via `Spec Reopen` instead of implicit requirement changes
- omit verification evidence path for proposed reliability fixes
- hide uncertainty instead of explicit `[assumption]` and residual-risk annotation
