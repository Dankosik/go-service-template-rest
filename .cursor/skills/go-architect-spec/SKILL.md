---
name: go-architect-spec
description: "Design architecture-first specifications for Go services in a spec-first workflow. Use when planning new features, refactors, or behavior changes before coding and you need boundaries, decomposition, interaction style, consistency model, resilience assumptions, and an implementation-ready architecture plan. Skip when the task is a local code fix, low-level API/DB/security implementation, test-case authoring, or CI/container configuration."
---

# Go Architect Spec

## Purpose
Create a clear, reviewable architecture specification for Go service changes before implementation. Success means architecture decisions are explicit, defensible, and directly translatable into coding tasks. Keep workflow control in `docs/spec-first-workflow.md`; focus this skill on architecture expertise.

## Scope And Boundaries
In scope:
- define service or module boundaries, ownership, and dependency direction
- decide component decomposition and seams
- decide sync or async interaction style and command or event intent
- decide consistency model (local transaction, eventual consistency, outbox or saga frame)
- define resilience shape (failure domains, degradation, rollout safety)
- produce architecture deliverables that remove "decide later" gaps

Out of scope:
- endpoint-level API payload, status, and error details
- physical SQL modeling, DDL details, and migration scripts
- concrete cache key, TTL, and invalidation policies
- detailed security control catalog and hardening checklists
- detailed telemetry schemas, SLI or SLO targets, and alert thresholds
- concrete CI or CD pipeline and container runtime hardening setup
- detailed test matrix design
- benchmark or profile plans and performance tuning details

## Hard Skills
### Architecture Core Instructions

#### Mission
- Produce architecture decisions that remain correct under growth, failure, and mixed-version rollout.
- Convert ambiguous requests into explicit boundaries, consistency contracts, and failure-mode contracts before coding starts.
- Ensure every selected architecture option is reviewable, testable, and rollback-safe.

#### Default Posture
- Prefer modular monolith boundaries until service extraction is justified on all four axes: domain, data ownership, team ownership, and transaction boundary.
- Prefer local ACID within one service-owned datastore; use explicit eventual-consistency patterns across services.
- Prefer explicit sync/async contracts with bounded deadlines and retries over hidden coupling.
- Prefer additive, compatibility-first evolution (`expand -> migrate/backfill -> contract`) over big-bang changes.
- Treat architecture overhead (operations, observability, release coordination) as first-class cost in every decomposition decision.

#### Boundary And Decomposition Competency
- Use the four-axis boundary model for every boundary decision: domain capability, data ownership, team ownership, transaction boundary.
- Require explicit source-of-truth ownership for each critical entity.
- Reject service-per-table, service-per-CRUD, and shared-schema decomposition.
- Reject cross-service direct DB access and cross-service foreign keys by default.
- Approve new service extraction only when independent deployability, ownership, scaling, and consistency tolerance are explicitly proven.
- Detect distributed-monolith signals early: coordinated releases, chatty chains, shared schema coupling, hidden shared business logic.

#### Sync Communication And API Style Competency
- Prove that synchronous hops are required before selecting transport.
- Apply transport defaults: external/public surface via REST/OpenAPI, internal service calls via gRPC/Protobuf unless justified otherwise.
- Define end-to-end deadline budget and per-hop budgets before approving call chains.
- Enforce explicit retry classification per operation and bounded retry budgets.
- Enforce idempotency design for retry-unsafe operations (`Idempotency-Key` or contract-equivalent).
- Require deterministic error mapping and one error model per API surface.
- Require deterministic pagination strategy and bounded list semantics.
- Enforce gateway/BFF ownership boundaries; never expose internal service contracts directly as public API.

#### Event-Driven And Async Workflow Competency
- Decide async usage from workflow properties (latency variability, fan-out, buffering/backpressure), not from tooling preference.
- Classify every message as event or command with explicit ownership semantics.
- Select topology intentionally: pub/sub for independent domain reactions, queue for owned work distribution.
- Require transactional outbox (or equivalent atomic linkage) when DB state change must emit message.
- Require consumer idempotency and durable dedup/inbox for side-effecting handlers.
- Require bounded retry strategy with jitter, poison-message handling, and explicit DLQ ownership/redrive policy.
- Require explicit schema evolution strategy (additive-first, versioned breaking changes).
- Require explicit ordering boundaries and replay-safety semantics; never assume global ordering.
- Require trace/context propagation and async outcome observability across send/process/retry/DLQ stages.

#### Distributed Consistency And Saga Competency
- Enforce local-transaction-only boundaries per service datastore; never assume cross-service global ACID.
- Build and maintain invariant register before selecting consistency mechanism.
- Classify invariants into `local_hard_invariant` and `cross_service_process_invariant`.
- Model multi-step workflow as explicit durable state machine with monotonic transitions.
- Define step contracts explicitly: trigger, local transaction scope, idempotency key, timeout, retry class, compensation/forward-recovery.
- Identify pivot transaction and enforce compensable-before / retryable-after rules.
- Require reconciliation ownership, cadence, and repair path for critical eventual-consistency flows.
- Reject dual writes, hidden invariant ownership, and distributed locks as primary correctness mechanism.

#### Resilience, Degradation, And Evolution Competency
- Classify dependencies per criticality (`critical_fail_closed`, `critical_fail_degraded`, `optional_fail_open`) before fallback design.
- Define per-dependency failure contract: timeout, retry budget, bulkhead, fallback mode, circuit strategy, observability signals.
- Enforce explicit deadline propagation and fail-fast behavior on exhausted remaining budget.
- Enforce bounded retries with jitter and non-retry classes.
- Enforce bounded queues/concurrency, overload shedding behavior, and blast-radius isolation.
- Require explicit degradation modes and activation/deactivation criteria.
- Require graceful startup/shutdown and probe semantics (`livez`/`readyz`/`startupz`) aligned with runtime behavior.
- Require rollout strategy for risky changes (canary/blue-green/strangler) with explicit rollback authority.
- Require error-budget-aware release gates and freeze policy for sustained burn.

#### Cross-Domain Architecture Impact Competency
- API impact obligations:
  - keep contract semantics explicit (resource model, sync/async behavior, idempotency class, consistency disclosure);
  - record contract-impact decisions in `30-api-contract.md` when architecture changes API behavior.
- Data impact obligations:
  - keep service-owned data boundaries explicit;
  - justify datastore-class choices by access-pattern evidence;
  - define migration compatibility window and rollback class for schema evolution;
  - frame cache usage by staleness/correctness contract, not as default optimization.
- Security and identity impact obligations:
  - define trust boundaries, principal model, and tenant isolation path;
  - define identity propagation model per hop (`forward`, `exchange`, `internal`);
  - require fail-closed authorization boundaries and object-level access control ownership.
- Operability impact obligations:
  - define minimum logs/metrics/traces correlation contract and cardinality guardrails;
  - define SLI/SLO/error-budget implications for architecture choices;
  - ensure debuggability endpoints and telemetry escalation model are safe-by-default.
- Delivery and platform impact obligations:
  - ensure architecture decisions are enforceable by CI quality gates (contract, migration, security, drift);
  - ensure runtime/container assumptions are explicit (non-root, startup/shutdown behavior, reproducible builds).

#### Evidence Threshold And Decision Quality Bar
- Every major architecture decision must include at least two options and one explicit rejection reason.
- Every selected option must include measurable acceptance boundaries, not only narrative rationale.
- Every selected option must include failure-mode analysis and control mechanisms.
- Every selected option must include cross-domain impact summary for API/data/security/operability.
- Every selected option must include rollout-safety and rollback limitations.
- Every selected option must include reopen conditions tied to observable triggers.
- Minimum evidence by decision axis:
  - boundary/decomposition: owner map + source-of-truth mapping + transaction-boundary justification;
  - sync interaction: call graph + deadline budget + retry/idempotency classification;
  - async/eventing: outbox/inbox strategy + retry/DLQ policy + schema-evolution path;
  - distributed consistency: invariant register + workflow state machine + compensation/forward-recovery plan;
  - resilience/evolution: dependency-failure matrix + degradation modes + rollout/rollback gate plan.

#### Assumption And Uncertainty Discipline
- Mark unknown critical facts as `[assumption]` immediately.
- Keep assumptions bounded and testable; never hide them inside generic phrasing.
- Resolve assumptions in the same pass by source-backed validation when possible.
- Promote unresolved critical assumptions to blockers in `80-open-questions.md` with owner and unblock condition.

#### Review Blockers For This Skill
- Architecture recommendation without explicit trade-off analysis.
- Architecture decision that shifts unresolved core choice into coding phase.
- New service boundary without data ownership and transaction-boundary proof.
- Sync call chain without explicit deadlines, retry semantics, and idempotency classification.
- Async design without outbox/inbox, bounded retry, or DLQ ownership.
- Distributed flow without invariant register and explicit workflow state model.
- Reliability strategy without fallback/degradation contract and rollback path.
- Cross-domain impact omitted for API/data/security/operability consequences.
- Decision rationale based on preference/tool familiarity instead of workload/constraint evidence.

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and pass goal before drafting decisions. Keep decision scope aligned to that phase.
2. Set phase-specific output targets before drafting decisions:
   - Phase 0: `00/10/80` and skeleton readiness for `15..90`
   - Phase 1: `20/60/80/90`
   - Phase 2 and later: `20/60/80/90` plus impacted `30/40/50/55/70`
3. Load context using the dynamic loading rules in this file and stop loading when all four architecture axes are source-backed: boundaries/ownership, interaction style, consistency model, resilience/rollout safety.
4. Frame the architecture problem: constraints, ownership boundaries, and non-negotiables.
5. For each major architectural decision, evaluate at least two options and select one explicitly.
6. Assign a decision ID and owner for each major architectural decision.
7. Record trade-offs and cross-domain impact (API, data, security, operability) for each selected option.
8. Mark missing critical facts as `[assumption]`, keep assumptions bounded, and resolve each assumption by either validating it with a cited source in the current pass or converting it into a blocker in `80-open-questions.md` with owner and unblock condition.
9. If an uncertainty blocks a decision, record it in `80-open-questions.md` with owner, unblock condition, and next step.
10. Produce the required deliverables in the required structure.
11. Check internal consistency: no conflicts and no hidden architectural decisions deferred to coding.
12. Keep focus on architecture expertise by stating technical positions, decision rationale, and cross-domain implications in architecture artifacts.

## Decision Classification
Treat a decision as major architectural when it changes at least one of:
- service or module boundaries, ownership, or dependency direction
- interaction style (sync/async) or command/event intent
- consistency guarantee shape (local transaction, eventual consistency, outbox/saga frame)
- failure, degradation, recovery, or rollout-safety behavior

## Architectural Decision Protocol
For every major architectural decision, document:
1. decision ID (`ARCH-###`) and current phase
2. owner role
3. context and problem
4. options (minimum two)
5. selected option with rationale
6. at least one rejected option with explicit rejection reason
7. trade-offs (gains and losses)
8. impact on API, data, security, and operability
9. risks and control mechanisms
10. reopen conditions
11. affected artifacts and linked open-question IDs (if any)

## Output Expectations
- Response format: architecture specification package with these artifacts.
- Phase-specific minimum artifacts:
  - Phase 0:
    - `00-input.md`: normalized problem statement, scope, non-goals, constraints, assumptions
    - `10-context-goals-nongoals.md`: context and success frame
    - `80-open-questions.md`: initial architecture blockers and owners
    - skeleton readiness for `15..90` according to `docs/spec-first-workflow.md`
  - Phase 1:
    - `20-architecture.md`: context and constraints, boundaries and ownership, dependency rules, interaction style, consistency choices, architecture risks and trade-offs
    - `60-implementation-plan.md`: architecture-safe implementation sequence with no hidden "decision later"
    - `80-open-questions.md`: architecture-only uncertainties and blockers
    - `90-signoff.md`: decisions accepted in the current pass with rationale and reopen criteria
  - Phase 2 and later:
    - `20-architecture.md`
    - `60-implementation-plan.md`
    - `80-open-questions.md`
    - `90-signoff.md`
- Conditional alignment artifacts (update when architecture decisions affect them):
  - `30-api-contract.md`: contract-level architecture implications only.
  - `40-data-consistency-cache.md`: consistency frame and data-boundary implications.
  - `50-security-observability-devops.md`: architecture-level security and operability constraints.
  - `55-reliability-and-resilience.md`: architecture-level timeout/retry/degradation/shutdown policy frame.
  - `70-test-plan.md`: architecture-driven test obligations only.
- Conditional artifact format for `30/40/50/55/70`:
  - include one explicit status per file: `Status: updated` or `Status: no changes required`
  - when `Status: no changes required`, add one sentence with justification and linked decision IDs
  - when `Status: updated`, list changed sections and linked decision IDs
- Language: match the user language when possible.
- Detail level: concrete and reviewable, with explicit decisions and explicit trade-offs.
- Constraint: keep the output architecture-level with explicit downstream implementation criteria and reviewable acceptance boundaries.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading once all four architecture axes are covered with at least one source-backed input each: boundaries/ownership, interaction style, consistency model, resilience/rollout safety.

Always load:
- `docs/spec-first-workflow.md`:
  - read only sections `2. Core Principles`, `3. Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only if a decision cannot be made without them
- `docs/project-structure-and-module-organization.md`:
  - read only sections relevant to boundaries, ownership, and dependency direction first
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/llm/architecture/10-service-boundaries-and-decomposition.md`

Load by trigger:

Sync request-reply style, API hop rules, or deadline propagation decisions:
- `docs/llm/architecture/20-sync-communication-and-api-style.md`

Eventing, async workflows, queue semantics, or outbox/inbox decisions:
- `docs/llm/architecture/30-event-driven-and-async-workflows.md`

Cross-service consistency, saga choreography/orchestration, or compensation decisions:
- `docs/llm/architecture/40-distributed-consistency-and-sagas.md`

Failure-domain, degradation, startup/shutdown, retry budget, or rollout safety decisions:
- `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`

API surface impact:
- `docs/llm/api/10-rest-api-design.md`
- `docs/llm/api/30-api-cross-cutting-concerns.md`

Data, store, or caching impact:
- `docs/llm/data/10-sql-modeling-and-oltp.md`
- `docs/llm/data/20-sql-access-from-go.md`
- `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
- `docs/llm/data/50-caching-strategy.md`
- `docs/llm/data/30-nosql-and-columnar-decision-guide.md`

Security or identity impact:
- `docs/llm/security/10-secure-coding.md`
- `docs/llm/security/20-authn-authz-and-service-identity.md`

Operability or delivery impact:
- `docs/llm/operability/10-observability-baseline.md`
- `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
- `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- `docs/llm/delivery/10-ci-quality-gates.md`
- `docs/llm/platform/10-containerization-and-dockerfile.md`
- `docs/build-test-and-development-commands.md`
- `docs/ci-cd-production-ready.md`

Deep trade-off support:
- only when core loaded docs are insufficient for a disputed trade-off or when the user requests evidence
- use minimal additional repo sources and cite exact file names in decision rationale

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict still remains, preserve the latest accepted decision in `90-signoff.md` and record a reopen item in `80-open-questions.md` with owner and unblock condition.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by either citing a validating source in the current pass or promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- Phase 0 pass is complete when `00/10/80` are updated and skeleton readiness for `15..90` is confirmed.
- Phase 1 pass is complete when `20/60/80/90` are updated and consistent.
- Phase 2 and later pass is complete when `20/60/80/90` are updated and each affected `30/40/50/55/70` file has explicit status (`updated` or `no changes required`) with decision links.
- Architecture frame is internally consistent across all impacted artifacts.
- Every major decision includes decision ID, owner, selected option, and at least one rejected option with reason.
- No hidden architectural decisions are deferred to coding.
- Key trade-offs, risks, assumptions, and constraints are explicitly documented.
- Every `[assumption]` is either source-validated or tracked as an open question with owner and unblock condition.
- Blockers are closed or explicitly recorded with clear owner and next step.
- Decisions are testable in review without reinterpretation.

## Anti-Patterns
- replacing technical position with generic workflow management
- making vague decisions without trade-off analysis
- pushing architectural uncertainty to coding phase
- mixing architecture scope with low-level implementation details
- copying requirements without explicit architectural choice
- loading full documents by default when section-level loading is enough
