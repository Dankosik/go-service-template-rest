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
Status: draft | review_ready | approved | implementing | verified
Subagent gate: complete | scoped_down | local_only | waived | not_expected | blocked

## Intent
What changes and why.

## Scope / Non-goals
In:
- ...

Out:
- ...

## Behavior / Contract Delta
ADDED:
- ...

MODIFIED:
- ...

REMOVED:
- ...

## Decisions
- D1: ...
- D2: ...

## Dependency / OSS Due Diligence
Applies: yes | no
Selected approach:
- <stdlib | existing repo pattern | OSS dependency | custom implementation>
Evidence:
- <current source/date, adoption, maintenance, license, security, fit>
Rejected options:
- <option> because <reason>
Custom-code justification:
- <required when selected approach is custom implementation>

## Pattern Fit Diligence
Applies: yes | no
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
- Does this change replace an existing path? If yes, list known old identifiers, routes, configs, commands, generated outputs, fixtures, docs, or mirrors to remove, refactor, retain, or prove not applicable. If no, record `No known replacement surface`.

Ownership / source of truth:
- ...

Sequence / failure behavior:
- ...

## Subagent Gate Decision
Gate type: <research fan-out | spec-clarification | local-only rationale | not expected>
Required lane policy: <default lens set | expanded lane set | scoped-down lane set | local-only rationale | not expected>
Consumed lane summaries or rationale:
- <lane/fan-in evidence pointer, or local-only rationale with candidate lanes considered>
Fan-in result:
- <orchestrator-owned resolution>
Readiness consequence: <next phase allowed yes/no, with proof obligations when allowed with concerns>
Reopen target: <none | research | specification | technical-design | planning>

## Risk Challenge
1. What irreversible or externally visible decision could be wrong?
   Answer: ...
2. What hidden invariant or owner could this break?
   Answer: ...
3. Are we writing custom code for a problem already solved by current stdlib, an established repo pattern, or mature OSS?
   Answer: ...
4. What fresh proof will make the completion claim trustworthy?
   Answer: ...
Gate: PASS | CONCERNS | FULL_REQUIRED

## Task Handoff
Use `tasks.md` only after specification review is `PASS` or `CONCERNS` with named proof obligations.

## Validation
Forward-looking proof obligations.

## Outcome
Pending until fresh validation evidence exists.
```

Rules:

- `Behavior / Contract Delta` describes added, modified, and removed behavior instead of restating the whole system.
- Replacement specs must name known legacy surfaces, expected remove/refactor/retain semantics, and the proof that will show each old surface is gone, active through the new path, intentionally retained, or out of scope.
- If the change does not replace an existing path, record `No known replacement surface` so planning does not invent cleanup work.
- `Dependency / OSS Due Diligence` is required when the change would add a dependency, create custom infrastructure, introduce a meaningful helper/abstraction, or solve a problem with plausible standard-library, repository-pattern, or OSS alternatives. If not applicable, record `Applies: no` only when the reason is obvious from scope.
- `Pattern Fit Diligence` is required when the change needs a non-trivial architecture, workflow, integration, resilience, consistency, data-flow, abstraction, or system-design choice. Keep it compact in lean specs when the choice is simple; preserve `research/pattern-fit.md` when multiple candidates, external sources, or examples need to survive; use `design/pattern-fit.md` only when the comparison is planning-critical and too large for `spec.md` or `design/overview.md`.
- `Compact Design` answers affected surfaces, ownership/source-of-truth, and sequence/failure behavior. If those answers become dense or contested, split into design artifacts or escalate.
- `Subagent Gate Decision` is required for non-trivial lean specs. If workflow-control already records the same audit, link to it instead of duplicating raw lane output. A non-trivial lean `spec.md` without this section or link remains draft.
- `Risk Challenge` is the lean replacement for a formal challenge lane only when no escalation trigger is present.
- `FULL_REQUIRED` blocks lean coding and routes to full orchestrated work.
- `Outcome` stays pending until fresh evidence exists.

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
- store final reconciled outcomes in `spec.md`;
- run a separate specification review after the completed `spec.md` exists and before technical design or planning;
- record gate status in `workflow-plan.md` and the active phase file when those files are used.

Formal `spec-clarification-challenge` is not waivable while the work remains full-orchestrated, protected-domain, high-impact, hard-to-reverse, cross-domain, or user-requested deep challenge. If the trigger no longer applies, first record shape reclassification with trigger-matrix evidence, then record the required subagent gate decision or local-only rationale for the new shape. Otherwise, missing formal clarification blocks `spec.md` from becoming review-ready.

Formal clarification asks only review-readiness-changing questions. Ordinary downstream design detail should be recorded as a constraint, proof obligation, follow-up, or `defer_to_design`, not as a reason to inflate `spec.md`. Do not classify architecture, ownership, contract, reliability, security, rollout, or validation choices this way when they are required to choose a production-ready solution for the accepted scope.

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
Resolution: <orchestrator-owned fan-in result>
```
