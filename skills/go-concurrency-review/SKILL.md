---
name: go-concurrency-review
description: "Review Go code changes for concurrency correctness in a spec-first workflow. Use when auditing pull requests or diffs with goroutines, channels, mutexes, worker pools, or shutdown/cancellation behavior to find race/deadlock/leak and bounded-concurrency risks with spec-reopen escalation. Skip when designing specs, implementing code, or performing primary-domain business, API, DB/cache, security, reliability, or broad style review."
---

# Go Concurrency Review

## Purpose
Deliver domain-scoped code review findings for concurrent behavior during Phase 4 review. Success means changed concurrent paths have safe goroutine lifecycle, cancellation/shutdown behavior, synchronized shared state, and bounded concurrency, with spec mismatches escalated before `Gate G4`.

## Scope And Boundaries
In scope:
- review concurrent code paths that use goroutines, channels, mutexes, wait groups, `errgroup`, worker pools, pipelines, fan-out/fan-in
- verify goroutine lifecycle and termination paths are explicit and safe
- verify cancellation and deadline propagation through concurrent operations
- verify channel ownership, close semantics, and blocking behavior
- verify shared mutable state synchronization and race-risk controls
- verify bounded concurrency and backpressure behavior
- verify shutdown behavior can unblock waits/sends/receives
- verify concurrent error propagation is not lost
- produce actionable findings with exact `file:line`, impact, and fix
- escalate spec mismatches through `Spec Reopen`

Out of scope:
- endpoint business meaning and product acceptance semantics as primary domain
- full idiomatic/style review outside concurrency concerns
- primary performance proof and benchmarking ownership
- primary DB/query/cache correctness ownership
- primary reliability policy ownership outside concurrent control-flow defects
- primary security review ownership
- full test-strategy ownership outside concurrency-specific coverage gaps
- editing spec artifacts in Phase 4

## Hard Skills
### Concurrency Review Core Instructions

#### Mission
- Produce evidence-backed concurrency findings that protect merge safety under real production interleavings, not only happy-path execution.
- Validate that changed concurrent paths satisfy Phase 4 expectations and can pass `Gate G4` without hidden race/deadlock/leak/shutdown risks.
- Convert concurrent risks into minimal, actionable fixes with explicit `file:line`, impact, and spec linkage.

#### Default Posture
- Review changed and directly impacted concurrent paths first; avoid broad style-only scanning.
- Treat scheduling-dependent correctness as a defect until explicit synchronization/lifecycle proof exists.
- Prefer explicit lifecycle, ownership, and cancellation semantics over timing assumptions or implicit goroutine behavior.
- Prefer the smallest safe fix that restores deterministic behavior before proposing structural redesign.
- Keep review comments strictly concurrency-domain; hand off non-concurrency root causes to the owning review skill.

#### Spec-First Review Competency
- Enforce Phase 4 reviewer boundaries from `docs/spec-first-workflow.md`: findings must stay in concurrency scope, be actionable, and not redesign approved architecture without explicit conflict.
- Enforce review output protocol from workflow docs:
  - finding format with `severity`, `skill`, `file:line`, `Issue`, `Impact`, `Suggested fix`, `Spec reference`;
  - severity aligned with merge risk;
  - explicit `Spec Reopen` when approved intent and safe implementation diverge.
- Treat unresolved `high/critical` concurrency findings as blockers for `Gate G4`.

#### Goroutine Lifecycle And Shutdown Competency
- Every started goroutine must have an explicit termination path: work complete, input close, context cancellation, or component shutdown.
- Flag fire-and-forget goroutines in production paths unless process-lifetime ownership and failure irrelevance are explicitly justified.
- Verify shutdown can unblock sends, receives, waits, and worker loops without indefinite blocking.
- Verify pipeline stages stop when downstream exits early or fails; blocked senders on abandoned channels are merge blockers.
- Verify long-running concurrent components stop timers/tickers/resources during shutdown.

#### Cancellation And Deadline Propagation Competency
- Require `context.Context` to drive cancellation across related concurrent work.
- For related goroutines with fail-fast semantics, prefer `errgroup.WithContext` over ad hoc orchestration.
- Require bounded concurrency (`errgroup.SetLimit` or equivalent semaphore/pool limit) where fan-out exists.
- Ensure potentially blocking channel/IO operations have cancel/deadline path (`select` with `ctx.Done()` or explicit timeout).
- Flag request-path `context.Background()` replacements as concurrency/reliability defects.
- Verify outbound/DB/cache operations in concurrent flows honor explicit deadlines.

#### Channel Ownership And Closure Competency
- Channel closure ownership must be explicit, usually producer/sender side.
- Receivers must not close channels they do not own.
- Multiple closers, double-close risk, and send-on-closed risk are `high` by default.
- Blocking send/receive paths must include cancellation or bounded queue semantics where appropriate.
- Buffered channels must represent intentional bounded queues with deliberate capacity, not hidden unbounded backlog.

#### Shared State Synchronization Competency
- Mutable shared state must be protected by synchronization or strict ownership confinement.
- Concurrent map read/write without synchronization is merge-blocking.
- Critical sections should stay minimal and obvious; lock scope must not hide long/blocking external operations.
- Prefer `sync.Mutex` by default; require evidence for `sync.RWMutex` complexity trade-off.
- Verify happens-before guarantees are explicit; correctness must not depend on scheduler luck.

#### Bounded Concurrency, Backpressure, And Bulkhead Competency
- Unbounded goroutine fan-out, unbounded worker pools, and unbounded channels/queues are blockers.
- Concurrency lanes to critical dependencies should be isolated and bounded (bulkhead behavior), not shared global unbounded pools.
- Under overload, code should prefer bounded waiting or fast fail over infinite accumulation.
- When reliability contracts apply, verify queue/concurrency bounds align with documented degradation and shedding behavior.
- For cache-backed concurrent read paths, require stampede controls (`singleflight`/coalescing, TTL jitter, bounded origin fallback concurrency).

#### Deadlock And Shutdown Safety Competency
- Verify lock/channel interaction cannot create cyclic waits across shutdown and steady-state paths.
- Verify wait paths (`Wait`, receive loops, blocking sends) can complete during cancellation/shutdown.
- Flag indefinite waits caused by missing close, missing cancellation branch, or conflicting lock order.
- Reject sleep/polling hacks used to avoid synchronization design issues.

#### Concurrency Error Propagation Competency
- Errors from worker goroutines must be observable by the caller/control plane; swallowed errors are blockers on significant paths.
- Group orchestration must define whether first-error cancels siblings and how aggregated errors are surfaced.
- For async consumers/workers, verify ack/commit happens only after durable local side effects and dedup state.
- Retry behavior in concurrent workers must be bounded, jittered, and class-aware; infinite retry loops are blockers.

#### Concurrent DB/Cache Interaction Competency
- Transactional concurrent code must preserve explicit boundary discipline (`Begin` -> `defer Rollback` -> `Commit`).
- Retry in concurrent DB paths should target full transaction block with scoped transient classes and idempotent writes.
- Long DB transactions around external network calls in concurrent workers are blockers.
- Cache fallback paths in concurrent flows must remain bounded and fail-open by default for read acceleration unless contract says otherwise.
- Verify cache timeout budgets are shorter than origin budgets in fallback paths to avoid amplified saturation.

#### Concurrency Security And Abuse-Resistance Competency
- Concurrency design must enforce resource bounds for untrusted/high-volume paths (size/time/concurrency limits).
- Fan-out or bulk processing triggered by untrusted input must use bounded worker pools/semaphores.
- Critical request-path work must not rely on untracked background goroutines.
- Expensive concurrent endpoints should expose overload behavior (for example `429`/`503`) instead of silent queue growth.

#### Concurrency Verification And Evidence Competency
- Significant concurrent changes require race-evidence (`go test -race ./...` or repo equivalent `make test-race` / `make docker-test-race`).
- Test evidence for concurrent changes should favor deterministic synchronization over timing sleeps.
- Recommend repository-aligned baseline checks where relevant:
  - `make test`
  - `make test-race`
  - `make lint`
  - CI-like `make docker-ci` when local toolchain parity matters
- If race/concurrency evidence is missing, record `[assumption: missing-race-evidence]` in `Residual Risks`.

#### Decision Quality And Assumption Discipline
- Findings must include concrete failure mode and merge impact, not only stylistic preference.
- Mark missing critical facts as bounded `[assumption]`; unresolved safety-impact assumptions must be escalated.
- Prefer spec-linked obligations (`55/70/90`, decision IDs, workflow gates) over generic advice.
- Escalate through `Spec Reopen` when safe fix conflicts with approved spec intent.

#### Review Blockers For This Skill
- Goroutine started without explicit completion/cancellation/shutdown path.
- Blocking operation without cancellation/deadline path in significant concurrent flow.
- Channel close ownership ambiguity or multiple potential closers.
- Unsynchronized shared mutable state or concurrent map access.
- Unbounded concurrency/backlog with no backpressure control.
- Lost worker errors or undefined concurrent failure propagation semantics.
- Missing race/concurrency verification evidence for significant concurrency changes.
- Any concurrency risk requiring spec intent change but left without `Spec Reopen`.

## Working Rules
1. Confirm review unit from context (`single task` or `bounded task scope`) and determine changed scope.
2. Map changed code to one or more concurrency axes. If no axis applies, return `No concurrency findings.` with `Residual Risks: none; no concurrency surface detected in changed scope.`.
3. Determine `feature-id` from review context, changed paths, or task metadata. If it cannot be identified, continue with bounded `[assumption]` and reduced certainty.
4. Load review context using this skill's dynamic loading rules.
5. Apply `Hard Skills` defaults from this file; any deviation must be explicit and justified in findings or residual risks.
6. Review only changed and directly impacted concurrent paths first; avoid broad repository scanning.
7. Evaluate eight concurrency axes for changed scope as the primary reporting taxonomy:
   - `Goroutine Lifecycle Safety`
   - `Cancellation And Deadline Semantics`
   - `Channel Ownership And Closure`
   - `Shared State Synchronization`
   - `Bounded Concurrency And Backpressure`
   - `Deadlock And Shutdown Safety`
   - `Concurrency Error Propagation`
   - `Concurrency Verification Readiness`
8. Activate trigger-driven hard-skill checks (reliability/async/data-cache/security) only when changed scope touches those surfaces.
9. Record only evidence-backed findings with concrete code location and specific obligation reference (prefer `RLY-*`/`TST-*`/decision IDs when present).
10. Classify severity by merge risk (`critical/high/medium/low`).
11. Provide the smallest safe corrective action for each finding.
12. If significant concurrent behavior changed, record verification status (`go test -race ./...` or repository equivalent `make test-race` / `make docker-test-race`) in `Residual Risks` when evidence is missing.
13. If safe resolution requires changing approved spec intent, create `Spec Reopen` in `reviews/<feature-id>/code-review-log.md`.
14. Keep comments strictly in concurrency-review domain and hand off cross-domain risks to the corresponding reviewer role.
15. If no findings exist, state this explicitly and include residual concurrency risks.

## Output Expectations
- Findings-first output ordered by severity.
- Match output language to the user language when practical.
- Use this exact finding format:

```text
[severity] [go-concurrency-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

- After findings, include:
  - `Handoffs`: cross-domain risks and owner skill.
  - `Spec Reopen`: `required` or `not required` with reason.
  - `Residual Risks`: non-blocking concurrency risks or verification gaps.
- Start each `Issue` value with axis context: `Axis: <one of the eight axes>; ...`.
- Keep section order stable: `Findings`, `Handoffs`, `Spec Reopen`, `Residual Risks`.
- Keep all sections present; if a section is empty, write `none` and one short reason.
- If there are no findings, output `No concurrency findings.` and still include `Residual Risks`.

Severity guide:
- `critical`: confirmed risk of deadlock, goroutine leak, data race on critical path, or shutdown hang that blocks safe merge.
- `high`: high-probability race/deadlock/leak or unbounded concurrency in significant path.
- `medium`: localized concurrency risk with bounded blast radius.
- `low`: local robustness/readability improvements that reduce concurrency risk.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once all eight concurrency review axes and any trigger-driven hard-skill checks for touched surfaces can be evaluated with code evidence and approved spec references.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4` criteria first
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/go-instructions/20-go-concurrency.md`
- review artifacts:
  - `specs/<feature-id>/90-signoff.md` (if present)
  - `reviews/<feature-id>/code-review-log.md` (if present)

Load by trigger:
- Concurrency behavior tied to reliability contracts:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Required test obligations for concurrent paths:
  - `specs/<feature-id>/70-test-plan.md`
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`
- Architecture-level ownership or lifecycle boundary ambiguity:
  - `specs/<feature-id>/20-architecture.md`
  - `specs/<feature-id>/65-coder-detailed-plan.md`
  - `specs/<feature-id>/60-implementation-plan.md`
- Async workflow semantics affecting channel/pipeline design:
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
- Data/cache interactions in concurrent flows:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/50-caching-strategy.md`
- Security implications caused by concurrent control flow:
  - `docs/llm/security/10-secure-coding.md`

Conflict resolution:
- Approved decisions in `specs/<feature-id>/90-signoff.md` override generic guidance unless `Spec Reopen` is raised.
- The more specific document is the decisive rule for that topic.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- If required spec artifacts are unavailable, mark `[assumption: missing-spec-artifacts]` and reduce certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated as `Spec Reopen`.
- If concurrency-verification evidence is missing for significant concurrent changes, mark `[assumption: missing-race-evidence]` and surface it in `Residual Risks`.

## Definition Of Done
- Concurrency review output stays within concurrency-review domain boundaries.
- Every finding is mapped to one explicit concurrency axis.
- Findings are evidence-backed and use exact `file:line` references.
- Every finding has impact, fix path, and spec reference.
- All `critical/high` concurrency findings are either resolved or clearly escalated.
- No unresolved `Review Blockers For This Skill` remain implicit.
- No spec-level mismatch remains implicit.
- If no findings, output explicitly states `No concurrency findings.` and includes residual risk note.

## Anti-Patterns
The following are review anti-patterns and should be treated as quality drift:
- providing broad non-concurrency comments without a concrete concurrency failure mode
- relying on timing luck (`sleep`, polling, scheduler behavior) as correctness argument
- accepting unbounded goroutine/queue growth without explicit backpressure controls
- accepting shutdown paths that can hang on waits, sends, receives, or lock cycles
- accepting missing race-evidence for significant concurrent changes without residual-risk annotation
- hiding spec-impacting concurrency conflicts instead of opening `Spec Reopen`
