# Execution Shape And Routing Hardening Workflow Plan

## Accepted Intake Brief

- Objective: repair every finding from the B01 execution-shape, artifact-depth, and workflow-planning routing audit without narrowing the accepted scope.
- Intent: make routing deterministic, executable, resumable, and regression-protected across canonical docs, session skills, challenger/status helpers, agent configuration, guardrails, evals, and runtime mirror policy.
- Scope:
  - direct-path code-writing authority and execution eligibility;
  - the canonical owner and invocation boundary for execution-shape classification and dedicated workflow planning;
  - execution-shape precedence, subagent-authorization semantics, and protected-trigger handling;
  - artifact expectation, lifecycle, phase, gate, session, and waiver status namespaces;
  - escalation, downgrade, reclassification, stale-state invalidation, and artifact-first resume;
  - research-not-expected routing and phase-local workflow-file triggers;
  - workflow-plan adequacy trigger, shape-falsification responsibility, status checks, and advisory authority;
  - workflow-status coverage for shape, adequacy, resume, and artifactless direct work;
  - guardrail, eval, CI, agent, skill-sync, and runtime-mirror coverage.
- Non-goals:
  - service runtime, REST/OpenAPI, database, deployment, or Go application behavior;
  - rewriting completed historical bundles under `specs/`;
  - changing unrelated workflow phases except where a direct routing/status contract consequence must be carried through their shared interface;
  - temporary compatibility wording, staged policy, or partial closure of the eleven findings.
- Constraints:
  - all eleven B01 findings remain in scope until traced to approved decisions, implementation tasks, and proof;
  - `AGENTS.md` remains the compact authority; detailed mechanics must have one canonical owner and lower-precedence surfaces must link rather than diverge;
  - subagents remain read-only and advisory; the orchestrator owns decisions and artifact writes;
  - implementation later requires an approved and reviewed `tasks.md`; this workflow-planning session writes only workflow-control files;
  - existing user work and completed historical task bundles must remain unchanged unless an approved task explicitly requires a compatibility correction outside those bundles.
- Success signal:
  - every B01 finding has an approved disposition, implementation task, and regression proof;
  - direct, lean, and full routing have executable and mutually stable semantics;
  - status and reclassification state are deterministic across master, phase-local, status-helper, and resume surfaces;
  - adequacy challenge, guardrails, evals, CI, and mirror policy protect the repaired contract;
  - canonical checks and targeted negative searches pass with no stale lower-precedence wording.
- Open questions: the precise repair design, compatibility treatment for existing status words, and whether a separate compact design artifact is needed belong to research and specification; none changes the accepted objective.

## Routing

- Execution shape: `full orchestrated`.
- Rationale: the accepted scope is a broad workflow-contract repair spanning multiple authoritative and mirrored owners, CI/eval enforcement, session routing, and hard-to-reverse execution authority. The user explicitly requires every finding to be fixed.
- Research mode: `fan-out`, complete.
- Current phase: `specification`.
- Phase status: `complete`.
- Session boundary reached: `yes`.
- Ready for next session: `yes`.
- Next session starts with: `specification-review`; run the distinct read-only multi-lane review of the review-ready `spec.md`, record the verdict in `workflow-plans/specification-review.md`, and stop before test design, planning, or implementation.
- Next session context bundle:
  - `specs/execution-shape-routing-hardening/workflow-plan.md` — accepted scope, complete finding map, artifact expectations, and current route.
  - `specs/execution-shape-routing-hardening/workflow-plans/specification.md` — completed formal clarification audit, dispositions, and specification stop boundary.
  - `specs/execution-shape-routing-hardening/spec.md` — canonical review-ready decisions for every `B01-F01`-`B01-F11`, proof obligations, and reopen conditions.
  - `specs/execution-shape-routing-hardening/research/synthesis.md` — evidence provenance and cross-finding audit to consult only when a spec claim needs verification.
  - `AGENTS.md` — hard authority, trigger matrix, agent/worker boundaries, and session invariants.
  - `docs/spec-first-workflow.md` — stable router and loading model.
  - `docs/spec-first-workflow/phases/specification-review.md` — owning review contract, verdict semantics, allowed writes, and stop rule.
  - `docs/spec-first-workflow/shared/subagents-and-handoff.md` — review fan-out, gate audit, authorization, and handoff contract.
  - `.agents/skills/specification-review-session/SKILL.md` — next-session review procedure; it may record review evidence but must not author the candidate decisions.

## B01 Finding Coverage

| ID | Required closure | Research owner |
| --- | --- | --- |
| `B01-F01` | Give code-writing direct path one coherent actor/authority route, or remove that advertised route explicitly. | `R1` |
| `B01-F02` | Establish one authoritative owner and invocation table for classification versus dedicated workflow planning. | `R1` |
| `B01-F03` | Separate or formally compose expectation, lifecycle, phase, gate, session, and waiver statuses. | `R2` |
| `B01-F04` | Define atomic escalation/downgrade/reclassification and stale-state disposition, including artifactless direct work. | `R2` |
| `B01-F05` | Route `research: not expected` independently from same-session collapse. | `R2` |
| `B01-F06` | Make adequacy challenge test shape selection against the authoritative trigger matrix and reclassification evidence. | `R3` |
| `B01-F07` | Distinguish capability-only subagent authorization from substantive user-requested agent-backed execution. | `R1` |
| `B01-F08` | Apply the independent durable-orchestration trigger to `workflow-plans/specification-review.md`. | `R2` |
| `B01-F09` | Unify adequacy-challenge trigger predicates and define any non-authoritative terms. | `R3` |
| `B01-F10` | Make workflow-status report shape/adequacy and handle or explicitly exclude artifactless direct work. | `R2` |
| `B01-F11` | Add semantic guardrail/eval/CI coverage and make canonical-versus-runtime mirror availability explicit. | `R3` |

No finding may be dropped because another repair incidentally appears to cover it. Research synthesis, specification, tasks, and validation must preserve this ID map.

## Artifact Expectations

| Artifact | Status | Trigger or rationale |
| --- | --- | --- |
| `workflow-plan.md` | `approved` | Cross-phase control completed after read-only adequacy challenge reconciliation. |
| `workflow-plans/workflow-planning.md` | `approved` | Dedicated routing phase completed with lane plan, stop rule, and reconciled adequacy result. |
| `workflow-plans/research.md` | `approved` | Multi-lane research control is complete; Subagent Gate Audit and fan-in are reconciled. |
| `research/shape-and-execution-authority.md` | `present, complete evidence` | Preserves R1 evidence for `B01-F01`, `B01-F02`, and `B01-F07`; recommendations remain non-authoritative. |
| `research/status-reclassification-and-resume.md` | `present, complete evidence` | Preserves R2 evidence for `B01-F03`, `B01-F04`, `B01-F05`, `B01-F08`, and `B01-F10`; recommendations remain non-authoritative. |
| `research/adequacy-enforcement-and-mirrors.md` | `present, complete evidence` | Preserves R3 evidence for `B01-F06`, `B01-F09`, and `B01-F11`; external runtime discovery limits remain proof-only. |
| `research/pattern-fit.md` | `present, complete evidence` | Pattern Fit Diligence compares decision-table, state-machine, typed-status, conditions, and event-history patterns and rejects disproportionate machinery. |
| `research/synthesis.md` | `present, complete evidence` | Reconciles all findings, candidate models, rejected alternatives, proof needs, classifications, and cross-reopen interactions. |
| `spec.md` | `review_ready` | Canonical decisions for all eleven findings are complete; formal clarification is reconciled, but independent specification review has not yet run. |
| `workflow-plans/specification.md` | `complete` | Five-lens formal clarification, targeted rechecks, Subagent Gate Audit, and phase boundary are reconciled. |
| `workflow-plans/specification-review.md` | `missing, expected next` | This task needs a distinct multi-lane specification review with durable routing; the reason is task-specific, not merely that `spec.md` exists. |
| `design/overview.md` | `not expected` | Research found the authority, decision, typed-state, transition, ownership, and proof tables can remain compact in `spec.md`; reopen only if specification clarification proves otherwise or introduces a real architecture/dependency decision. |
| Separate system/integration and Go code ownership design | `not expected` | No service runtime or Go package ownership change is in accepted scope; reopen only if specification introduces one. |
| Technical design review | `not expected` | No separate design artifact is currently triggered; reopen automatically if `design/overview.md` or split design becomes expected. |
| `test-plan.md` | `missing, expected later` | Direct/lean/full, escalation, downgrade, resume, adequacy, eval, and mirror behavior require a scenario matrix and fail-before proof. |
| `tasks.md` | `missing, expected later` | Non-trivial multi-surface implementation requires an approved executable ledger and task-review/readiness gate. |
| `rollout.md` | `not expected` | Repository-local contract and tooling repair has no deployment or mixed-version rollout surface. |
| Review/validation phase files | `conditional, trigger unknown` | Planning may create only the named files needed for real multi-session review or validation. |

## Gates And Assumptions

- Workflow-plan adequacy challenge: `complete`; read-only lane `A1` returned two `blocks_phase_handoff` findings, both reconciled by strengthening completion/gate state and loading the owning research/handoff contracts.
- Subagent gate for workflow planning: `complete`.
  - Trigger: full-orchestrated, broad, multi-owner workflow-control repair.
  - Gate type: `workflow-adequacy`.
  - Required lane policy: one narrow read-only challenger after draft control artifacts.
  - Lane result: `A1` confirmed the shape and finding coverage, then identified missing final-state reconciliation and incomplete research-session loading.
  - Orchestrator fan-in: both blockers repaired in the master and phase-local files; no finding was waived or accepted as risk.
  - Readiness consequence: research may start only in the recorded next session after loading the named contracts and activating `workflow-plans/research.md`.
  - Reopen target: `workflow-planning` if the research session cannot start from this packet without reclassifying shape, lanes, artifacts, or stop state.
- Subagent gate for research: `complete`; `R1`-`R4` returned, all required evidence was preserved, and the full audit/fan-in is recorded in `workflow-plans/research.md` and `research/synthesis.md`.
- Research fan-in classification: all `B01-F01`-`B01-F11` normative choices are `blocks_spec`, meaning specification must decide them before review-ready handoff; named runtime/carrier/eval-runner limits are `proof_only`; accepted risk and needs-specialist registers are empty.
- Subagent gate for specification: `complete`; all five default formal clarification lenses ran read-only with `spec-clarification-challenge`, approval-changing questions were repaired from existing evidence, and targeted rechecks returned `CLEAR`.
- Formal clarification gate: `complete and reconciled`; disposition and proof destinations are recorded in `spec.md` and `workflow-plans/specification.md`.
- Pattern Fit Diligence: `complete`; the research packet recommends a small decision-table plus typed-state/guarded-transition composite for specification to evaluate and rejects imported workflow engines/event sourcing as disproportionate.
- Dependency/OSS diligence: `not expected` because no new runtime dependency is currently proposed; reopen if a candidate repair introduces one.
- Accepted assumption: the repository-local sources and current clean-checkout mirror behavior inspected in research are the current evidence baseline; external runtime mirror discovery and exact platform need for authorization wording remain proof-only limits.
- Accepted risk: none.
- Blockers: none for specification review entry.
- Reopen target: specification if independent review finds an approval blocker; research only if the review exposes missing evidence that cannot be resolved from the current packet; workflow planning only if shape/artifact routing must change.

## Phase Workflow Plans

- `workflow-plans/workflow-planning.md`: `complete`; adequacy findings reconciled and research handoff ready.
- `workflow-plans/research.md`: `complete`; `R1`-`R4`, Pattern Fit, Subagent Gate Audit, synthesis, classifications, and specification route reconciled.
- `workflow-plans/specification.md`: `complete`; review-ready spec, five-lens clarification, targeted rechecks, and Subagent Gate Audit reconciled.
- `workflow-plans/specification-review.md`: expected next; create only in the distinct specification-review session.
- Later phase-local files: create only at their recorded trigger and phase boundary.

## Phased Delivery Policy

- Target state is one coherent replacement contract, not an MVP plus deferred hardening.
- Completed historical bundles remain evidence only and are not rewritten to simulate current-state consistency.
- Implementation must update every canonical, lower-precedence, enforcement, eval, agent, and mirror-source surface named by the approved ledger; unexplained stale wording is a completion blocker.
