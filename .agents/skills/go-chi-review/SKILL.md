---
name: go-chi-review
description: "Review Go code changes for go-chi transport-routing correctness in a spec-first workflow. Use when auditing pull requests or diffs for chi router topology, middleware ordering/scope, 404/405/OPTIONS/CORS behavior, route conflict and shadowing risk, OpenAPI chi integration, and route-label observability semantics. Skip when designing specifications, implementing features, or performing primary security/reliability/performance/concurrency/DB/QA reviews."
---

# Go Chi Review

## Purpose
Deliver domain-scoped code review findings for `github.com/go-chi/chi/v5` transport-routing behavior during Phase 4 review. Success means router and middleware behavior remains aligned with approved `chi` decisions, merge-unsafe routing regressions are surfaced before `Gate G4`, and spec-intent conflicts are escalated explicitly.
Use `Hard Skills` as the normative `go-chi` baseline for decision quality and merge-blocking thresholds; use workflow sections below for execution order and output protocol.

## Scope And Boundaries
In scope:
- review changed transport code against approved routing intent in `specs/<feature-id>/20-architecture.md` and `specs/<feature-id>/60-implementation-plan.md`
- review `chi` router topology and route ownership (`Route`/`Group`/`Mount`, root vs subrouter boundaries)
- review route conflict/shadowing/override risks caused by registration order or mixed route ownership
- review middleware ordering invariants and global vs local scope boundaries
- review `NotFound`, `MethodNotAllowed`, `Allow`, `OPTIONS`, and CORS behavior for affected API surfaces
- review observability route semantics (`RoutePattern` extraction timing, low-cardinality labels, trace/log/metric consistency)
- review OpenAPI/codegen integration boundaries for `chi`-generated and manually wired routes
- review transport lifecycle safety impact of routing-layer changes (startup/readiness/shutdown/fallback behavior)
- produce actionable findings with exact `file:line`, impact, and minimal safe fix
- escalate spec-level conflicts through `Spec Reopen`

Out of scope:
- redesigning architecture during code review without explicit `Spec Reopen`
- editing spec artifacts in Phase 4
- performing primary-domain API payload modeling, business invariants, DB/cache correctness, security architecture, reliability architecture, performance proof, or QA strategy ownership
- blocking PRs with preference-only comments without concrete routing/runtime impact

## Hard Skills
### Go-chi Review Core Instructions

#### Mission
- Protect merge safety by finding `chi` routing and middleware correctness defects before `Gate G4`.
- Preserve deterministic HTTP behavior and route observability semantics across router refactors and feature additions.
- Keep findings enforceable against approved spec decisions, not reviewer preference.

#### Default Posture
- Treat `chi` as stdlib-compatible transport composition, not as a business-logic layer.
- Treat implicit framework defaults as risky when API-visible behavior is not explicitly pinned by spec.
- Prefer explicit route ownership, explicit middleware order, and explicit fallback behavior over implicit runtime behavior.
- Keep domain ownership strict and hand off cross-domain root causes to the correct reviewer skill.
- Prefer the smallest safe fix that restores contract conformance over broad redesign proposals.

#### Spec-First Review Competency
- Enforce `docs/spec-first-workflow.md` Phase 4 constraints:
  - domain-scoped findings only;
  - exact `file:line`;
  - practical fix path;
  - explicit `Spec Reopen` for spec-intent conflict.
- Treat unresolved `critical/high` `go-chi` findings as merge blockers for `Gate G4`.
- Never modify approved spec intent implicitly through review comments.
- Never edit spec files in code-review phase.

#### Router Topology And Match Semantics Competency
- Verify changed route registrations preserve deterministic ownership:
  - root router vs mounted subrouter boundaries remain explicit;
  - generated and manual routes do not silently overlap.
- Flag duplicate or ambiguous `method+pattern` registration as high-risk behavior drift.
- Flag registration-order-dependent behavior where ownership is not explicitly guarded.
- Verify pattern/param routes do not unintentionally shadow static routes.
- Require route ownership rules to remain testable and observable.

#### Middleware Order And Scope Competency
- Validate middleware order invariants for:
  - request/correlation ID propagation;
  - security headers and boundary checks;
  - body/framing limits;
  - logging/observability enrichment;
  - panic recovery.
- Validate scope boundaries:
  - global middleware should only include truly global behavior;
  - route-local middleware should not unintentionally widen or narrow coverage.
- Flag middleware reorder without explicit behavior impact analysis.
- Flag middleware that depends on route pattern before route is resolved.

#### 404/405/OPTIONS/CORS Policy Competency
- Verify explicit policy for unmatched routes (`NotFound`) and method mismatch (`MethodNotAllowed`).
- Verify `Allow` header behavior remains contract-consistent when `405` is used.
- Verify `OPTIONS` behavior and CORS preflight handling match approved API policy.
- Flag implicit reliance on framework defaults for API-critical behavior.
- Verify scoped CORS setup does not leave uncovered preflight paths.

#### Observability Route Semantics Competency
- Verify route-template extraction uses `chi.RouteContext(...).RoutePattern()` at the correct lifecycle point.
- Flag labels/spans based on raw request path or high-cardinality values.
- Verify route semantics remain consistent across logs, metrics, and tracing.
- Verify fallback behavior when route pattern is unavailable is explicit and bounded-cardinality-safe.
- Ensure observability behavior remains stable across mounted subrouters.

#### OpenAPI And Codegen Chi Integration Competency
- Verify route wiring remains aligned with approved `oapi-codegen` `chi` mode usage.
- Verify generated handlers and manual handlers coexist without collision ambiguity.
- Verify generated ownership boundaries are preserved (no manual edits in generated artifacts).
- Flag runtime contract drift between OpenAPI-intended behavior and router-level implementation behavior.

#### Transport Lifecycle And Runtime Safety Competency
- Verify router-layer changes do not break `http.Server` lifecycle expectations.
- Verify unmatched/method-disallowed fallback behavior remains deterministic under load and shutdown.
- Verify middleware/panic-recovery changes preserve safe boundary behavior.
- Flag route-layer changes that can cause startup/readiness inconsistency or shutdown surprises.

#### Cross-Domain Handoff Competency
- Hand off to `go-security-review` when root issue is trust-boundary/authz/tenant isolation.
- Hand off to `go-reliability-review` when root issue is timeout/retry/degradation policy.
- Hand off to `go-performance-review` when root issue needs benchmark/profile evidence.
- Hand off to `go-concurrency-review` when root issue is goroutine/channel/lock lifecycle.
- Hand off to `go-qa-review` when primary gap is test strategy completeness.
- Hand off to `go-design-review` when safe correction requires architecture change outside approved intent.

#### Evidence Threshold And Severity Calibration Competency
- Every finding must include:
  - exact `file:line`;
  - violated routing/middleware contract or approved `chi` decision;
  - concrete runtime impact;
  - smallest safe corrective action;
  - explicit `Spec reference`;
  - verification command suggestion.
- Severity is merge-risk based, never preference based:
  - `critical`: confirmed merge-unsafe routing behavior defect with high production impact;
  - `high`: strong evidence of major contract drift (`404/405/OPTIONS`, collisions, label semantics);
  - `medium`: bounded but meaningful routing correctness risk;
  - `low`: local hardening with non-blocking impact.
- Generic "clean up routing" advice without failure impact is invalid output.

#### Assumption And Uncertainty Discipline
- Mark missing critical facts as bounded `[assumption]` immediately.
- If required artifacts are missing, annotate `[assumption: missing-spec-artifacts]` and reduce certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated via `Spec Reopen`.
- Do not hide uncertainty behind vague wording.

#### Review Blockers For This Skill
- Ambiguous route ownership that can produce collision/shadowing/override drift.
- Middleware ordering/scope change that can alter boundary behavior without explicit contract rationale.
- `404/405/OPTIONS/CORS` behavior left implicit on affected API surfaces.
- High-cardinality or inconsistent route observability semantics.
- OpenAPI/codegen and runtime router integration conflict with unresolved ownership.
- Spec-intent conflict left implicit instead of explicit `Spec Reopen`.

## Working Rules
1. Confirm the task is code review and identify changed `chi` routing-sensitive scope.
2. Determine `feature-id` from review context, changed paths, or task metadata. If it cannot be identified, continue with bounded `[assumption]` and reduced certainty.
3. Load context using this skill's dynamic loading rules.
4. Apply `Hard Skills` defaults from this file; any deviation must be explicit in findings or residual risks.
5. Evaluate changed code in this order:
   - `Router Topology And Ownership`
   - `Middleware Order And Scope`
   - `404/405/OPTIONS/CORS Behavior`
   - `Route Observability Semantics`
   - `OpenAPI/Codegen Chi Integration`
   - `Transport Lifecycle Safety`
6. Record only evidence-backed findings and map each finding to explicit approved obligations (prefer `CHI-*` decisions or clauses in `20/30/50/55/60/70/90`).
7. Classify severity by merge safety impact (`critical/high/medium/low`) and provide the smallest safe corrective action.
8. Keep comments strictly in `go-chi-review` domain and hand off deep cross-domain root causes to the corresponding reviewer role.
9. If safe fix requires changing approved spec intent, create `Spec Reopen` in `reviews/<feature-id>/code-review-log.md`.
10. Do not edit spec files during code review.
11. If no findings exist, state this explicitly and include residual routing risks.
12. Run final blocker check against `Hard Skills -> Review Blockers For This Skill` before closing the pass.

## Output Expectations
- Findings-first output ordered by severity.
- Match output language to the user language when practical.
- Use this exact finding format:

```text
[severity] [go-chi-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

- After findings, include:
  - `Handoffs`: cross-domain risks and owner skill.
  - `Spec Reopen`: `required` or `not required` with reason.
  - `Residual Risks`: non-blocking routing risks, assumptions, or verification gaps.
  - `Validation commands`: minimal command set to verify proposed fixes.
- Keep section order stable: `Findings`, `Handoffs`, `Spec Reopen`, `Residual Risks`, `Validation commands`.
- Keep all sections present; if a section is empty, write `none` and one short reason.
- If there are no findings, output `No go-chi findings.` and still include `Residual Risks` and `Validation commands`.

Severity guide:
- `critical`: confirmed transport behavior defect that makes merge unsafe.
- `high`: strong evidence of significant routing contract mismatch likely to break common production flows.
- `medium`: bounded but meaningful routing correctness weakness.
- `low`: local robustness hardening with non-blocking impact.

Suggested validation command pool:
- `go test ./...`
- `go test ./... -run <TargetedTest> -count=1`
- `make test`
- `make openapi-check` (when API-visible behavior or generated integration is affected)
- `make lint`

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once routing topology, middleware policy, HTTP policy, observability semantics, and codegen ownership for changed scope are assessable with code evidence and approved spec references.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4` criteria first
- `docs/deep-research-report (64).md`
- review artifacts:
  - `specs/<feature-id>/20-architecture.md`
  - `specs/<feature-id>/60-implementation-plan.md`
  - `specs/<feature-id>/90-signoff.md`
  - `reviews/<feature-id>/code-review-log.md` (if present)

Load by trigger:
- API cross-cutting behavior impact:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
- Reliability/fallback behavior impact:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Observability route-label/span impact:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- Security boundary implications in middleware chain:
  - `docs/llm/security/10-secure-coding.md`
- OpenAPI/codegen integration implications:
  - `api/openapi/service.yaml`
  - `internal/api/oapi-codegen.yaml`
  - `internal/api/README.md`
- Test and command baseline validation:
  - `specs/<feature-id>/70-test-plan.md`
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
- Review output stays within `go-chi-review` domain boundaries.
- Every finding is evidence-backed and references exact `file:line`.
- Every finding includes impact, fix path, and spec reference.
- All `critical/high` findings are either resolved or clearly escalated.
- No unresolved `Review Blockers For This Skill` remain implicit.
- No spec-level mismatch remains implicit.
- If no findings, output explicitly states `No go-chi findings.` and includes residual risk note.

## Anti-Patterns
The following are review anti-patterns and should be treated as quality drift:
- broad non-routing comments without concrete transport behavior impact
- relying on implicit `chi` defaults for API-critical behavior
- ignoring route collision/shadowing potential from registration order
- extracting/labeling route semantics using raw request path (high cardinality)
- reviewing payload/domain logic as primary scope instead of routing domain
- hiding spec-impacting routing conflicts instead of opening `Spec Reopen`
