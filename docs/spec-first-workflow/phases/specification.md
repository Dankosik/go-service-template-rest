# Specification Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this when writing or repairing `spec.md`, reconciling clarification challenge, or running lean `Risk Challenge`.

## Read When

- The active phase is specification or specification repair.
- Lean work needs a compact `spec.md`, or full-orchestrated work needs formal clarification fan-in recorded before review.
- A `Risk Challenge`, dependency/OSS due diligence, Pattern Fit Diligence, or replacement-surface decision belongs in the spec.

## Inputs

- Accepted scope and non-goals from the user request or workflow-control state.
- Research outputs, provider contracts, or source-of-truth evidence needed for spec decisions.
- Subagent lane summaries or a valid local-only rationale when the spec is non-trivial.

## Outputs

- Review-ready `spec.md` with final orchestrator-owned decisions, assumptions, accepted risks, and proof obligations.
- Clarification challenge fan-in or lean `Risk Challenge` result.
- Next route to specification review, or a reopen target when the spec cannot become review-ready.

## Stop Rule

Stop when `spec.md` is review-ready or blocked. Do not start specification review, technical design, task planning, or implementation inside this phase.

## Lean `spec.md`

Lean specs should answer the planning-critical questions without becoming a design bundle.

Recommended shape:

```markdown
# <Feature / Change>

Mode: lean local
Status: draft | review_ready | blocked
Subagent gate: complete | scoped_down | local_only | waived | not_expected | blocked

## Intent
- Decision statement: <what changes, for whom, and why now>
- Source evidence: <user request, research, contract, incident, or repo fact that justifies the scope>

## Scope / Non-goals
In:
- ...

Out:
- <true exclusion, not future hardening for in-scope work>

## Behavior / Contract Delta
ADDED:
- <observable behavior, route, command, config, event, error, status, metric, doc promise, or operator workflow>

MODIFIED:
- <old behavior> -> <new behavior>, including caller/operator-visible compatibility effects

REMOVED:
- <old behavior or surface>, including replacement or retention decision when relevant

UNCHANGED:
- <important compatibility, source-of-truth, or invariant that downstream work must preserve>

## Decisions
- D1: <decision>
  - Rationale: <why this is the selected production-ready answer for the accepted scope>
  - Evidence: <source, research note, contract, repository precedent, or bounded assumption>
  - Assumptions: <bounded assumptions, or none>
  - Rejected alternatives: <option> because <reason>
  - Downstream consequence: <constraint, proof obligation, accepted risk, or none>
  - Open questions: none

## Dependency / OSS Due Diligence
Applies: yes | no
Reason:
- <why the trigger applies or clearly does not apply>
Contract to satisfy:
- <capability, ownership, operational, security, or compatibility requirement>
Selected approach:
- <stdlib | existing repo pattern | OSS dependency | custom implementation>
Evidence:
- <current source/date or repository precedent, adoption, maintenance/release signal, license, security, transitive cost, API stability, repository fit>
Rejected options:
- <option> because <reason>
Custom-code justification:
- <required when selected approach is custom implementation>

## Pattern Fit Diligence
Applies: yes | no
Reason:
- <why architecture/workflow/integration/resilience/consistency/data-flow/abstraction pattern choice is or is not in scope>
Forces:
- <task forces that a pattern or straightforward repo-native design must satisfy>
Selected pattern:
- <named pattern or "straightforward repo-native design">
Evidence:
- <source/date or repository precedent, concise pattern description, and real-use example>
Applicability:
- <why the pattern's forces match this task, or why no known pattern fits>
Go fit:
- <why the shape preserves idiomatic Go: explicit control flow, small interfaces, context/cancellation, package ownership, and simple composition>
Rejected patterns:
- <pattern> because <scope/reliability/operability/Go-fit mismatch>

## Compact Design
Affected surfaces:
- `internal/...`

Legacy surfaces:
- Does this change replace an existing path? If yes, classify each known old identifier, route, config, command, generated output, fixture, doc, skill, agent, or mirror as `remove`, `refactor_into_active_path`, `retain`, or `out_of_scope`. If no, record `No known replacement surface`.
- Retained surfaces require current owner, reason, proof of continued need, and exit condition.
- Legacy cleanup audit:
  - `Surface | Decision | Evidence checked | Proof obligation | Retention owner/reason/proof/exit`

Ownership / source of truth:
- ...

Sequence / failure behavior:
- ...

Spec/design boundary:
- <decisions and constraints this spec owns>
- <mechanism choices intentionally deferred to technical design, if any>

## Subagent Gate Decision
Gate type: <research fan-out | spec-clarification | local-only rationale | not expected>
Required lane policy: <default lens set | expanded lane set | scoped-down lane set | local-only rationale | not expected>
Consumed lane summaries or rationale:
- <lane/fan-in evidence pointer, or local-only rationale with candidate lanes considered>
Fan-in result:
- <orchestrator-owned resolution>
Accepted risks:
- <risk, boundary, and why it is acceptable now, or none>
Proof obligations:
- <claim that later design, planning, implementation, or validation must prove>
Readiness consequence: <next phase allowed yes/no, with proof obligations when allowed with concerns>
Reopen target: <none | research | specification | system-integration-design | go-code-ownership-design | planning>

## Risk Challenge
1. What irreversible or externally visible decision could be wrong?
   Answer: ...
2. What hidden invariant or owner could this break?
   Answer: ...
3. Are we writing custom code for a problem already solved by current stdlib, an established repo pattern, or mature OSS?
   Answer: ...
4. Are we inventing an architecture, workflow, integration, resilience, data-flow, or abstraction shape without Pattern Fit Diligence?
   Answer: ...
5. What legacy surface could survive without a remove/refactor/retain/out-of-scope decision?
   Answer: ...
6. What fresh proof will make the completion claim trustworthy?
   Answer: ...
Gate: PASS | CONCERNS | FULL_REQUIRED

## Accepted Risks / Reopen Conditions
Accepted risks:
- <risk> | <why accepted> | <boundary/non-goal> | <owner> | <downstream carry target> | <reopen trigger>

Reopen if:
- <new evidence, failed proof, missing owner decision, or escalation trigger that invalidates this spec>

## Task Handoff
Next phase: specification review.
Use `tasks.md` only after specification review is `PASS` or `CONCERNS` with named proof obligations.

## Validation
Forward-looking proof obligations:
- <claim> -> <fresh command, manual check, contract source, generated diff, log, or other evidence required later; include freshness requirement, negative proof, and downstream artifact that must carry it when relevant>

## Outcome
Pending until fresh validation evidence exists.
```

Rules:

- `Intent` states the decision outcome, not a research question. If the reason for the work is still unknown or disputed, keep the spec draft and reopen intake, research, or user decision.
- `Status` in the specification phase is limited to `draft`, `review_ready`, or `blocked`. Later approval, implementation, verification, and outcome updates belong to the owning review, planning, implementation, validation, or closeout phase.
- `Scope / Non-goals` cuts the accepted problem. It must not hide required target-state work as future hardening when the production-ready decision is knowable and in scope.
- `Behavior / Contract Delta` describes added, modified, removed, and important unchanged observable behavior instead of restating the whole system. Include caller, operator, or maintainer-visible effects such as routes, payload fields, errors, status mapping, config, metrics, generated artifacts, docs promises, and source-of-truth ownership when they matter.
- `Decisions` must be complete enough that design or planning can preserve the chosen outcome without rediscovering product meaning. A decision row should name the selected answer, rationale, evidence or bounded assumption, rejected alternatives, and downstream consequence. Do not mark `spec.md` review-ready with `TBD`, unresolved alternatives, or "decide during implementation" placeholders.
- Replacement specs must name known legacy surfaces, expected remove/refactor/retain/out-of-scope semantics, and the proof that will show each old surface is gone, active through the new path, intentionally retained, or out of scope.
- If the change does not replace an existing path, record `No known replacement surface` with the bounded evidence checked so planning does not invent cleanup work.
- If there is no behavior or contract delta, record `No behavior/contract delta` with the bounded rationale instead of leaving the section blank.
- `Dependency / OSS Due Diligence` is required when the change would add a dependency, create custom infrastructure, introduce a meaningful helper/abstraction, or solve a problem with plausible standard-library, repository-pattern, or OSS alternatives. If not applicable, record `Applies: no` only when the reason is obvious from scope.
- `Pattern Fit Diligence` is required when the change needs a non-trivial architecture, workflow, integration, resilience, consistency, data-flow, abstraction, or system-design choice. Keep it compact in lean specs when the choice is simple; preserve `research/pattern-fit.md` when multiple candidates, external sources, or examples need to survive; use `design/pattern-fit.md` only when the comparison is planning-critical and too large for `spec.md` or `design/overview.md`.
- When dependency/OSS or Pattern Fit evidence lives in `research/` or `design/`, `spec.md` still records the final selected option, rejected-options summary, and custom-code or custom-design justification when relevant.
- `Compact Design` records only spec-level constraints: affected surfaces, ownership/source-of-truth, and sequence/failure behavior required to choose the production-ready outcome. If package layout, detailed algorithms, object lifecycles, task ordering, test matrices, rollout sequencing, or dense mechanism trade-offs become planning-critical, split into design artifacts or escalate.
- `Subagent Gate Decision` is required for non-trivial lean specs. If workflow-control already records the same audit, link to it instead of duplicating raw lane output. A non-trivial lean `spec.md` without this section or link remains draft.
- `Risk Challenge` is the lean replacement for a formal challenge lane only when no escalation trigger is present.
- `Accepted Risks / Reopen Conditions` may carry bounded risks only after the orchestrator names the boundary, why it does not block review-ready status, and what evidence would reopen the spec. Do not use accepted risk to bypass a required user, domain, security, reliability, data, compatibility, dependency/OSS, Pattern Fit, legacy-cleanup, or validation decision.
- `Validation` records forward-looking proof obligations, not a test plan. Each obligation should tie a claim to the fresh evidence later phases must produce; skipped, unavailable, stale, failing, or too-narrow proof cannot satisfy the claim.
- `FULL_REQUIRED` blocks lean coding and routes to full orchestrated work.
- `Outcome` stays pending until fresh evidence exists.

## Review-Ready Bar

Mark `spec.md` `review_ready` only when all of the following are true:

- accepted scope, non-goals, and target-state behavior are explicit;
- `Behavior / Contract Delta` covers every planning-critical added, modified, removed, and important unchanged behavior;
- every material decision has rationale, evidence or bounded assumption, rejected alternatives when real, downstream consequence, and no unresolved `TBD`;
- dependency/OSS diligence and Pattern Fit Diligence are complete or not applicable with rationale;
- known replacement and legacy surfaces are classified as remove, refactor into active path, retain with owner/reason/proof/exit condition, out of scope, or `No known replacement surface`;
- subagent gate state is recorded, and formal clarification fan-in or valid local-only/scoped-down rationale is present where required;
- lean `Risk Challenge` is `PASS` or `CONCERNS` with named proof obligations, or the spec routes to `FULL_REQUIRED`;
- accepted risks and proof obligations are named with boundaries and reopen conditions, or explicitly recorded as none;
- spec-owned constraints are separated from design mechanism choices and planning-owned task order;
- the next route is specification review.

If any item is missing, keep status `draft` or `blocked`; when blocked, name the smallest reopen target. Do not rely on the later specification-review phase to author the missing decision.

## Specification, Clarification, And Review Gates

`spec.md` is always the decision authority for task-local decisions.

For direct path, no `spec.md` is usually needed.

For lean local:

- write a compact review-ready `spec.md`;
- consume multiple narrow subagent summaries or record a local-only rationale;
- run the inline `Risk Challenge`;
- proceed to specification review only when the orchestrator has reconciled lane outputs or the local-only rationale, and the gate is `PASS` or `CONCERNS` with named proof obligations;
- proceed to compact tasking or technical design only after specification review is `PASS` or `CONCERNS` with named proof obligations;
- escalate when the gate is `FULL_REQUIRED`.

For full orchestrated or protected-domain work:

- run formal `spec-clarification-challenge` before `spec.md` is marked review-ready;
- for broad or multi-domain full-orchestrated, protected-domain, high-impact, hard-to-reverse, cross-domain, or user-requested deep challenge work, use multi-challenger lens fan-out rather than one generic challenger by default;
- use read-only challenger output as questions for orchestrator reconciliation, not as authority;
- store final reconciled outcomes in `spec.md`, not raw lane transcripts;
- run a separate specification review after the completed `spec.md` exists and before technical design or planning;
- record gate status in `workflow-plan.md` and the active phase file when those files are used.

Formal `spec-clarification-challenge` is not waivable while the work remains full-orchestrated, protected-domain, high-impact, hard-to-reverse, cross-domain, or user-requested deep challenge. If the trigger no longer applies, first record shape reclassification with trigger-matrix evidence, then record the required subagent gate decision or local-only rationale for the new shape. Otherwise, missing formal clarification blocks `spec.md` from becoming review-ready.

Formal clarification asks only review-readiness-changing questions. Ordinary downstream design detail should be recorded as a constraint, proof obligation, follow-up, or `defer_to_design`, not as a reason to inflate `spec.md`. Do not classify architecture, ownership, contract, reliability, security, rollout, or validation choices this way when they are required to choose a production-ready solution for the accepted scope.

Clarification fan-in should classify each surviving point by destination:

- `spec_decision`: write or repair a final decision, scope rule, behavior delta, accepted risk, or reopen condition in `spec.md`;
- `spec_constraint`: record a constraint that technical design or planning must preserve;
- `proof_obligation`: record the claim and the later fresh evidence required to prove it;
- `accepted_risk_candidate`: either name the accepted risk boundary in `spec.md` or reopen the owning decision;
- `defer_to_design`: allowed only for mechanism choices that cannot change review-ready scope, ownership, contract, reliability, security, rollout, validation, dependency/OSS, Pattern Fit, or legacy-cleanup decisions;
- `reopen`: send the work back to research, specification, specialist review, or user decision before review-ready status.

Use `defer_to_design` only after `spec.md` records the invariant, observable contract, owner or source of truth, constraints, and proof obligation that design must preserve. Do not defer a production-readiness decision merely because its implementation mechanism will be designed later.

If lanes conflict on a review-readiness-critical fact or severity, the orchestrator must resolve the conflict from evidence, accept a bounded risk, or mark the spec blocked with a reopen target. Do not average conflicting lane results into vague wording.

Default broad clarification lenses:

- scope and spec coherence;
- domain invariants and edge cases;
- architecture ownership and dependency boundaries;
- API, data, compatibility, and source-of-truth consequences;
- security, reliability, delivery, and validation proof.

Each lens is a separate read-only lane, usually `challenger-agent` with `spec-clarification-challenge`. Lanes may run in parallel when their questions are independent. Add extra lanes for real independent review-readiness-risk domains, including when one default lens bundles domains that are independently review-readiness-critical for the task. Use fewer lanes only with a recorded scoped-down rationale; a single lane is appropriate only for a narrow formal gate whose review-readiness risk is concentrated in one question.

Before spawning, convert every lens into a concrete review-readiness-critical question and lens-specific inspect-first list. Do not send five challengers the same generic "challenge this spec" prompt. If two lenses produce the same question, merge them or split the real underlying owner question before fan-out.

Do not collapse broad formal clarification into one generic challenger merely because one agent could inspect all domains. Use the default lens set as separate read-only lanes. Fewer lanes require `Scoped-down rationale:` listing every default lens, the review-readiness-critical question considered for that lens, retained lane or lanes, and evidence-backed reason each omitted lens cannot change `spec.md` review-readiness. If any omitted lens has an unresolved review-readiness-critical question, that lens must run.

`Risk Challenge=CONCERNS` in lean local does not by itself trigger formal multi-challenger clarification. It requires named proof obligations and a check for unresolved scope, ownership, validation, or escalation gaps. Route to formal clarification only when those gaps cannot be honestly closed inline or another escalation trigger appears.

Multi-lane workflow-control records should use:

```text
Clarification challenge: complete | blocked | not_expected
Lanes: <agent + skill summary>
Lenses: <lens list>
Scoped-down rationale: <why fewer than the broad default, when applicable>
Disposition table: <Lens | Question | Strongest finding | Disposition | Evidence pointer | Owner/artifact updated>
Resolution: <orchestrator-owned fan-in result with destination classifications>
```
