---
name: go-chi-spec
description: "Design go-chi-routing-first specifications for Go services in a spec-first workflow. Use when planning or revising HTTP router topology, middleware ordering, 404/405/OPTIONS/CORS policy, OpenAPI chi integration, and route-label observability semantics before coding. Skip when the task is a local code fix, endpoint payload contract design, DB/schema-cache design, CI/container setup, or implementation-only coding/review."
---

# Go Chi Spec

## Purpose
Create a clear, reviewable transport-routing specification for Go services using `github.com/go-chi/chi/v5` before implementation. Success means routing and middleware behavior is explicit, testable, and free of "decide later in code" ambiguity.

## Scope And Boundaries
In scope:
- define `chi` router topology (`Route`/`Group`/`Mount`, root vs subrouter composition)
- define middleware layering and ordering invariants (`global` vs route-local)
- define OpenAPI + `oapi-codegen` integration shape for `chi` server generation
- define `NotFound`/`MethodNotAllowed` behavior and `OPTIONS`/CORS policy boundaries
- define route-conflict and route-shadowing prevention controls
- define route-template observability semantics for logs/metrics/traces with bounded cardinality
- define transport-boundary fallback/degradation behavior for unmatched or method-disallowed routes
- produce routing deliverables that are directly translatable into implementation and tests

Out of scope:
- endpoint payload/status schema ownership (`api-contract-designer-spec` domain)
- DB/schema/migration/cache strategy ownership
- deep authn/authz threat modeling ownership
- detailed SLI/SLO/alert policy ownership
- low-level code implementation and test writing
- domain-scoped code-review responsibilities

## Hard Skills
### Chi Routing Specification Core Instructions

#### Mission
- Convert ambiguous HTTP routing expectations into explicit, reviewable transport contracts before coding.
- Preserve contract and observability stability during router migrations (`ServeMux` -> `chi`) and router refactors.
- Prevent hidden runtime drift from route conflicts, middleware-order changes, or fallback behavior changes.

#### Default Posture
- Keep `net/http` compatibility and explicit composition.
- Prefer deterministic routing behavior over framework defaults when defaults are contract-ambiguous.
- Keep middleware stack explicit and ordered; no hidden global router state.
- Keep observability labels route-template-based and low-cardinality.
- Keep OpenAPI source of truth unchanged; adapt transport wiring, not contract ownership.

#### Go-chi Framing And Philosophy Competency
- Treat `chi` as a stdlib-first transport router, not as a full-stack framework.
- Keep core architecture unchanged: business logic and domain rules stay outside router concerns.
- Use `chi` for composability under API growth:
  - route grouping and modular mounting,
  - local middleware scoping,
  - `net/http` ecosystem compatibility.
- Preserve `http.Handler` interoperability for testing and integration (`httptest`, standard middleware, `http.Server` lifecycle).
- Choose `chi` by operational need, not trend:
  - choose when routing composition and middleware ergonomics are the bottleneck,
  - avoid unnecessary migration when current router complexity is low and stable.

#### Chi Behavioral Semantics Competency
- Treat routing behavior as explicit contract, including framework-specific nuances:
  - global `Use(...)` middleware runs before final route match context is fully resolved,
  - route-template extraction via `RoutePattern()` is reliable only after downstream handling,
  - default `405` and `OPTIONS` behavior must be evaluated and pinned to explicit policy,
  - duplicate route registrations can silently override by registration order unless prevented.
- Require every semantics-sensitive behavior to be documented and test-covered:
  - route conflict/shadowing prevention,
  - `404/405/OPTIONS` response policy,
  - CORS preflight handling path,
  - route label/span naming consistency.

#### Router Topology Competency
- Require explicit topology decision per affected path set:
  - single router registration policy,
  - mounted subrouter policy,
  - direct handler coexistence policy.
- For paths that may collide (for example generated + direct endpoints), require one deterministic ownership point.
- Reject route plans that allow silent override behavior without guardrails and tests.

#### Middleware Order And Scope Competency
- Define exact middleware order and explain why order matters for:
  - correlation/request ID,
  - security headers,
  - framing/body limits,
  - access logging and route-label extraction,
  - panic recovery.
- Require explicit `global` vs route-local scope boundaries.
- Reject reorder proposals without behavior impact analysis.

#### OpenAPI And Codegen Integration Competency
- Keep OpenAPI source of truth in `api/openapi/service.yaml`.
- Require explicit `oapi-codegen` mode decision for routing (`chi-server`) and strict wrapper behavior.
- Ensure generated and manual routes can coexist without collision ambiguity.
- Prohibit manual edits of generated files.

#### 404/405/OPTIONS/CORS Policy Competency
- Define explicit policy for:
  - `NotFound` response shape,
  - `MethodNotAllowed` response shape and `Allow` header behavior,
  - `OPTIONS` behavior and preflight handling boundaries,
  - CORS middleware placement constraints.
- Treat CORS placement as architecture decision:
  - top-level middleware by default,
  - if scoped placement is used, require explicit `OPTIONS` route strategy.
- Reject "leave framework defaults" when API-facing behavior must be stable and reviewable.

#### Observability Route Semantics Competency
- Require unified route-template extraction policy:
  - prefer `chi.RouteContext(...).RoutePattern()`,
  - define safe fallback behavior,
  - define timing rules for extraction in middleware lifecycle.
- Enforce bounded cardinality:
  - no raw request path/user/request IDs in route labels.
- Ensure one consistent route semantic across logs, metrics, and span naming.

#### Resilience And Lifecycle Interface Competency
- Preserve graceful startup/shutdown behavior of `http.Server` integration.
- Define transport fallback semantics for unmatched and method-disallowed requests.
- Require explicit behavior when router-level degradation or policy mismatch appears.

#### Evidence Threshold And Assumption Discipline
- Every major `CHI-###` decision must include:
  - at least two options,
  - one explicit rejected option with reason,
  - measurable acceptance boundaries,
  - reopen conditions.
- Every major decision must include framework-semantics evidence from:
  - repository code/tests and
  - approved references (for this repo: `docs/deep-research-report (64).md` or official docs linked from it).
- Mark missing critical facts as bounded `[assumption]`.
- Resolve assumptions in-pass when possible; otherwise move them to `80-open-questions.md` with owner and unblock condition.

#### Review Blockers For This Skill
- Router topology allows unresolved route-collision or shadowing ambiguity.
- Middleware order/scope is changed without explicit impact analysis.
- `404/405/OPTIONS` policy is left implicit for API-facing behavior.
- Observability route semantics are undefined or high-cardinality-unsafe.
- OpenAPI/codegen integration path is inconsistent with generated/runtime ownership.
- Critical routing uncertainty is deferred to coding instead of tracked as blocker.

## Working Rules
1. Determine current phase from `docs/spec-first-workflow.md` and target gate before proposing decisions.
2. Load minimal context using this skill's dynamic loading rules.
3. Normalize routing problem scope: affected paths, policies, middleware chain, and observability impact.
4. For each nontrivial routing decision, evaluate at least two options and select one explicitly.
5. Assign decision IDs (`CHI-###`) and owners for major routing decisions.
6. Record trade-offs and cross-domain impact (API/security/operability/reliability).
7. Define explicit implementation obligations in `60-implementation-plan.md`.
8. Define explicit test obligations in `70-test-plan.md`.
9. Move unresolved routing blockers to `80-open-questions.md`.
10. Verify no router/middleware policy decisions are deferred to coding.
11. Produce output mapped to required artifacts and decision IDs.

## Output Expectations
- Produce/update specification artifacts, minimum:
  - `20-architecture.md`
  - `60-implementation-plan.md`
  - `80-open-questions.md`
  - `90-signoff.md`
- Update conditional artifacts when impacted:
  - `30-api-contract.md`
  - `50-security-observability-devops.md`
  - `55-reliability-and-resilience.md`
  - `70-test-plan.md`
- For each impacted conditional artifact, include explicit status:
  - `Status: updated` or
  - `Status: no changes required` (+ one-line rationale linked to decision IDs)
- Keep language aligned with user language when practical.
- Keep output architecture-level, not implementation-code-level.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when routing topology, middleware policy, HTTP policy, and observability semantics are all source-backed.

Always load:
- `docs/spec-first-workflow.md`:
  - `Core Principles`, relevant phase section, and target gate criteria
- `docs/project-structure-and-module-organization.md`
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/llm/architecture/10-service-boundaries-and-decomposition.md`
- `docs/llm/api/30-api-cross-cutting-concerns.md`

Load by trigger:
- Routing philosophy/trade-off rationale or framework behavior disputes:
  - `docs/deep-research-report (64).md`
- Sync HTTP behavior, error mapping, deadline implications:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
- Degradation and fallback semantics:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Route labeling and telemetry behavior:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- Boundary security controls in routing path:
  - `docs/llm/security/10-secure-coding.md`
- OpenAPI/codegen routing integration:
  - `api/openapi/service.yaml`
  - `internal/api/oapi-codegen.yaml`
  - `internal/api/README.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents.
- If conflict remains with frozen spec intent, do not decide locally; raise `Spec Clarification Request`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Promote unresolved semantic assumptions to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Routing topology and middleware policy are explicit and internally consistent.
- `404/405/OPTIONS` behavior is explicitly defined for affected surfaces.
- Route-template observability semantics are explicit and bounded-cardinality-safe.
- Major decisions include IDs, options, rejected option rationale, and reopen conditions.
- No hidden routing decisions are deferred to coding.
- Required artifacts are updated with explicit status and decision links.

## Anti-Patterns
- "Replace router and decide behavior in code later"
- implicit middleware ordering without invariants
- implicit `OPTIONS/CORS` behavior on public API paths
- raw-path observability labels that create unbounded cardinality
- mixing this role with API payload design or security architecture ownership
- broad document loading when section-level loading is sufficient
