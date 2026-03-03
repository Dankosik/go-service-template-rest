---
name: go-db-cache-review
description: "Review Go code changes for DB/cache correctness in a spec-first workflow. Use when auditing pull requests or diffs for SQL query discipline, transaction boundaries, context/deadline and pool hygiene, cache key and invalidation correctness, and stampede/degradation risks. Skip when designing specifications, implementing features, or performing primary architecture/security/performance/concurrency/reliability/QA reviews."
---

# Go DB Cache Review

## Purpose
Deliver domain-scoped code review findings for DB/cache correctness during Phase 4 review. Success means data-access and cache behavior stays aligned with approved contracts, merge-unsafe consistency risks are surfaced before `Gate G4`, and spec-intent conflicts are escalated explicitly.
Use `Hard Skills` as the normative DB/cache baseline for decision quality and severity; use workflow sections below for execution sequence and output protocol.

## Scope And Boundaries
In scope:
- review changed code against approved DB/cache contracts in `specs/<feature-id>/40-data-consistency-cache.md`
- review SQL query discipline (`N+1`, query-in-loop, round-trip amplification, hot-path query misuse)
- review transaction boundary correctness and partial-side-effect risk
- review DB/cache context propagation, timeout/deadline use, and pool/resource safety
- review cache key isolation (`tenant/scope/version`) and serialization safety
- review invalidation/update/write-through behavior, TTL/jitter, and staleness contract conformance
- review stampede/degradation/origin-protection behavior in cache miss and outage paths
- review DB/cache fail-path test traceability against approved `70-test-plan.md`
- produce actionable findings with exact `file:line`, impact, and minimal safe fix
- escalate spec-level conflicts through `Spec Reopen`

Out of scope:
- redesigning architecture during code review without explicit `Spec Reopen`
- editing spec artifacts in Phase 4
- performing primary-domain idiomatic/style, architecture integrity, performance proof, concurrency mechanics, reliability policy, security policy, QA strategy, or domain-invariant review
- blocking PRs with preference-only comments without concrete DB/cache correctness impact

## Hard Skills
### DB/Cache Review Core Instructions

#### Mission
- Protect merge safety by finding DB/cache correctness defects in changed paths before `Gate G4`.
- Preserve consistency and stale-data contracts while keeping fixes minimal and practical.
- Keep conclusions enforceable against approved spec decisions, not reviewer preference.

#### Default Posture
- Review correctness and consistency behavior first, then optimization opportunities.
- Treat unbounded query fan-out, implicit infinite timeout, and unscoped cache keys as defects until proven safe.
- Treat cache as an accelerator, not a source of truth, unless approved contract states otherwise.
- Require explicit invalidation/staleness behavior on write-influenced read paths.
- Keep domain ownership strict: hand off deep non-DB/cache root causes to corresponding review skills.

#### Spec-First Review Competency
- Enforce `docs/spec-first-workflow.md` Phase 4 constraints:
  - domain-scoped findings only;
  - exact `file:line`;
  - practical fix path;
  - explicit `Spec Reopen` for spec-intent conflict.
- Treat unresolved `critical/high` DB/cache findings as merge blockers for `Gate G4`.
- Never modify approved spec intent implicitly through review comments.
- Never edit spec files in code-review phase.

#### Query Discipline Competency
- Flag `N+1`, per-item query loops, and avoidable round-trip amplification in changed paths.
- Flag expensive list/query patterns in hot paths (for example deep `OFFSET` without bounded rationale).
- Require parameterized query values and allowlisted dynamic identifiers when query shape is user-influenced.
- Flag repeated identical dependency reads in one request path when batching/preload is feasible.
- Treat hidden full-scan or high-cardinality query amplification in request path as merge risk.

#### Transaction Boundary And Consistency Competency
- Verify transaction scope groups dependent read/write steps that must commit atomically.
- Flag partial-commit/partial-side-effect risks across related operations.
- Flag long transactions wrapped around network or cross-service calls unless explicitly justified in spec.
- Verify retry-on-transaction policy is bounded and limited to transient classes, with whole-transaction retry semantics.
- Require idempotency safeguards for retried mutating paths.
- Confirm transaction behavior matches approved clauses in `40-data-consistency-cache.md`.

#### Context, Timeout, And Resource Safety Competency
- Require request context propagation into DB/cache calls; `context.Background()` in request path is a blocker unless explicitly justified.
- Require explicit deadlines/timeouts for blocking DB/cache calls in critical paths.
- Validate lifecycle safety for DB resources:
  - `rows.Close` and `rows.Err` checks;
  - `tx` rollback/commit discipline;
  - no leaked cursors or unfinished transactions.
- Flag pool-starvation patterns from long-held transactions or unbounded concurrent DB work.

#### Cache Key Isolation And Serialization Competency
- Require cache keys to include correctness dimensions mandated by contract:
  - tenant/account scope;
  - auth/scope variant when applicable;
  - locale/feature qualifiers when response shape depends on them;
  - key version.
- Flag cross-tenant/cross-scope cache key collision risk as high-severity isolation defect.
- Require deterministic key construction and bounded normalization for key fragments.
- Validate decode/schema-mismatch behavior is safe (treat as miss, evict or overwrite invalid entry, stay observable).
- Reject runtime wildcard key scans (`KEYS`-style patterns) in request paths.

#### Invalidation, TTL, And Staleness Contract Competency
- Verify every cached path has explicit freshness ownership:
  - write-triggered invalidation/update;
  - TTL-based expiration;
  - or approved hybrid policy.
- Verify write paths correctly synchronize cache invalidation/update with durable state transitions.
- Verify staleness behavior matches approved API/domain contract, including stale-window bounds where defined.
- Flag TTL-only strategy when correctness requires stricter invalidation.
- Validate negative-cache behavior uses short TTL and does not cache transient dependency failures as business negatives.

#### Stampede, Degradation, And Origin-Protection Competency
- Require stampede controls on hot/expensive miss paths:
  - request coalescing (`singleflight`-style or equivalent);
  - bounded fallback concurrency;
  - retry/backoff discipline.
- Flag cache outage behavior that can overload origin/DB without containment.
- Verify fallback mode (`fail_open`, `bypass`, bounded stale) matches approved reliability policy.
- Require degraded-mode activation/deactivation to be observable with bounded-cardinality signals.

#### Test Readiness And Evidence Competency
- Map DB/cache findings to explicit obligations in `70-test-plan.md`.
- Require critical path coverage for:
  - hit/miss/error/bypass;
  - invalidation after writes;
  - staleness bounds;
  - stampede suppression behavior.
- For concurrency-sensitive cache wrappers, require race-evidence path (`go test -race ./...` or `make test-race`).
- For integration-sensitive DB/cache behavior, require integration evidence path (`make test-integration` where applicable).
- Require verification command guidance for each nontrivial suggested fix.

#### Cross-Domain Handoff Competency
- Hand off to `go-performance-review` when root issue requires benchmark/profile-level evidence for latency/throughput impact.
- Hand off to `go-concurrency-review` when primary risk is goroutine/channel/lock lifecycle and DB/cache is only symptom.
- Hand off to `go-reliability-review` when primary defect is timeout/retry/degradation/backpressure policy.
- Hand off to `go-security-review` when trust-boundary, authz, tenant-isolation, or sensitive-data controls are primary.
- Hand off to `go-qa-review` when primary gap is test strategy/completeness rather than DB/cache logic correctness.
- Hand off to `go-design-review` when safe correction requires architecture change outside approved intent.

#### Evidence Threshold And Severity Calibration Competency
- Every finding must include:
  - exact `file:line`;
  - concrete DB/cache defect or risk;
  - practical impact on correctness/consistency/availability;
  - smallest safe fix path;
  - explicit `Spec reference`;
  - minimal verification command suggestion.
- Severity is merge-risk based, never preference based:
  - `critical`: confirmed data-correctness, isolation, or stale-contract breach that makes merge unsafe;
  - `high`: strong evidence of significant DB/cache contract mismatch likely to cause incidents;
  - `medium`: bounded but meaningful DB/cache weakness with limited blast radius;
  - `low`: local hardening improvement with non-blocking impact.
- Generic "improve query/cache" comments without concrete risk are invalid output.

#### Assumption And Uncertainty Discipline
- Mark missing critical facts as bounded `[assumption]` immediately.
- If required artifacts are missing, annotate `[assumption: missing-spec-artifacts]` and reduce certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated via `Spec Reopen`.
- Do not hide uncertainty behind vague language.

#### Review Blockers For This Skill
- Confirmed data-consistency or isolation defect in DB/cache behavior for changed path.
- High-risk `N+1`/query amplification in critical request path without safe mitigation.
- Transaction boundary flaw that can produce partial commit or partial side effects.
- Cache key missing mandatory tenant/scope/version dimensions for affected data.
- Missing or incorrect invalidation/staleness behavior that violates approved contract.
- Stampede-prone miss path or cache-outage path that can overload origin without containment.
- Critical DB/cache calls without explicit deadline/context propagation.
- Spec-intent conflict left implicit instead of explicit `Spec Reopen`.

## Working Rules
1. Confirm review unit from context (`single task` or `bounded task scope`), then identify changed DB/cache-sensitive scope.
2. Determine `feature-id` from review context, changed paths, or task metadata. If it cannot be identified, continue with bounded `[assumption]` and reduced certainty.
3. Load context using this skill's dynamic loading rules.
4. Apply `Hard Skills` defaults from this file; any deviation must be explicit in findings or residual risks.
5. Evaluate changed code in this order:
   - `Query Discipline`
   - `Transaction Boundary Correctness`
   - `DB Context/Timeout/Resource Safety`
   - `Cache Key Isolation And Serialization`
   - `Invalidation/TTL/Staleness Correctness`
   - `Stampede/Degradation/Origin Protection`
   - `DB/Cache Test Traceability`
6. Record only evidence-backed findings and map each finding to explicit approved obligations (prefer clauses in `40/60/65/70/90`, and `30/50/55` when relevant).
7. Classify severity by merge safety impact (`critical/high/medium/low`) and provide the smallest safe corrective action.
8. Keep comments strictly in DB/cache-review domain; hand off deep cross-domain root causes to the corresponding reviewer role.
9. If safe fix requires changing approved spec intent, create `Spec Reopen` in `reviews/<feature-id>/code-review-log.md`.
10. Do not edit spec files during code review.
11. If no findings exist, state this explicitly and include residual DB/cache risks.
12. Run final blocker check against `Hard Skills -> Review Blockers For This Skill` before closing the pass.

## Output Expectations
- Findings-first output ordered by severity.
- Match output language to the user language when practical.
- Use this exact finding format:

```text
[severity] [go-db-cache-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

- After findings, include:
  - `Handoffs`: cross-domain risks and owner skill.
  - `Spec Reopen`: `required` or `not required` with reason.
  - `Residual Risks`: non-blocking DB/cache risks, assumptions, or verification gaps.
  - `Validation commands`: minimal command set to verify proposed fixes.
- Keep section order stable: `Findings`, `Handoffs`, `Spec Reopen`, `Residual Risks`, `Validation commands`.
- Keep all sections present; if a section is empty, write `none` and one short reason.
- If there are no findings, output `No DB/cache findings.` and still include `Residual Risks` and `Validation commands`.

Severity guide:
- `critical`: confirmed data-integrity, cache-isolation, or stale-consistency breach in changed path that makes merge unsafe.
- `high`: strong evidence of significant DB/cache correctness mismatch likely to break common production flows.
- `medium`: bounded but meaningful DB/cache correctness weakness with limited blast radius.
- `low`: local hardening improvement with non-blocking impact.

Suggested validation command pool:
- `go test ./...`
- `go test -race ./...`
- `go test ./... -run <TargetedTest> -count=1`
- `make test`
- `make test-race`
- `make test-integration`
- `make openapi-check` (when API-visible consistency/error semantics are affected)
- `make lint`

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once all DB/cache review axes are assessable with code evidence and approved spec references.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4` criteria first
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/data/20-sql-access-from-go.md`
- `docs/llm/data/50-caching-strategy.md`
- review artifacts:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `specs/<feature-id>/65-coder-detailed-plan.md`
  - `specs/<feature-id>/60-implementation-plan.md`
  - `specs/<feature-id>/70-test-plan.md`
  - `specs/<feature-id>/90-signoff.md`
  - `reviews/<feature-id>/code-review-log.md` (if present)

Load by trigger:
- API-visible consistency/staleness/idempotency semantics:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Timeout/retry/degradation policy interaction:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Tenant isolation and sensitive data handling impact:
  - `specs/<feature-id>/50-security-observability-devops.md`
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Schema evolution/backfill/cache-version transition implications:
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
- Concurrency-sensitive cache layers (`singleflight`, worker pools, locks/channels):
  - `docs/llm/go-instructions/20-go-concurrency.md`
- Test evidence and command baseline:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`

Conflict resolution:
- Approved decisions in `specs/<feature-id>/90-signoff.md` override generic guidance unless `Spec Reopen` is raised.
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- If required spec artifacts are unavailable, mark `[assumption: missing-spec-artifacts]` and reduce certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated as `Spec Reopen`.

## Definition Of Done
- Review output stays within DB/cache-review domain boundaries.
- Findings are evidence-backed and use exact `file:line` references.
- Every finding has impact, fix path, spec reference, and verification command guidance.
- All `critical/high` DB/cache findings are either resolved or clearly escalated.
- Cross-domain root causes are handed off explicitly instead of being absorbed by this skill.
- No active item from `Hard Skills -> Review Blockers For This Skill` remains unresolved.
- If no findings, output explicitly states `No DB/cache findings.` and includes residual risk and validation-command notes.

## Anti-Patterns
Use these preferred patterns to avoid anti-pattern drift:
- keep findings anchored to concrete DB/cache correctness risk and exact code location
- report transaction/invalidation/staleness defects with explicit blast radius, not generic warnings
- hand off cross-domain root causes instead of broadening ownership implicitly
- prefer the smallest safe correction path before redesign proposals
- escalate spec-intent conflicts through `Spec Reopen` instead of implicit requirement changes
- omit verification-command guidance for nontrivial fixes
- hide uncertainty instead of explicit `[assumption]` and residual-risk annotation
