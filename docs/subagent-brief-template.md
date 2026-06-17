# Subagent Brief Template

Use this template when the orchestrator asks a read-only specialist lane for research, review, adjudication, or challenge. For non-trivial phases, lanes are planned as a coverage map over independent expert questions. Fill only the sections that matter and keep the brief compact.

```text
Goal:
- <one sentence describing what the lane must decide or check>

Scope:
- Agent: <agent-name>
- Mode: <research | review | adjudication | challenge>
- Skill: <skill-name | no-skill>
- Lens or specialist domain: <required for multi-lane fan-out; omit only when not applicable>
- Sibling lenses: <names of related lanes, when part of fan-out>
- Read-only enforcement: <execution choice that prevents writes, or "kept local because no reliable read-only lane exists">; do not edit files, mutate git state, or change task ledgers or implementation handoffs.

Context:
- Workflow phase: <phase or "none">
- Task artifacts: <paths to workflow-plan.md/spec.md/design/tasks and any triggered test-plan.md or rollout.md when relevant>
- Source-of-truth inputs: <contracts, docs, diffs, files, commands, specialist outputs>
- Constraints and non-goals: <short list>
- Known blockers or assumptions: <short list>
- Dependency/OSS scope: <new dependency, custom infrastructure, material abstraction, or "not in scope"; include expected stdlib/repo-pattern/OSS comparison when relevant>
- Pattern fit scope: <architecture/workflow/integration/resilience/consistency/data-flow/abstraction pattern choice, or "not in scope"; include expected known-pattern comparison when relevant>
- Contract-design scope: <REST/API, OpenAPI/generated, event/webhook, material internal interface, or "not in scope"; include expected resource/status/error/retry/async/freshness/compatibility question when relevant>
- Legacy cleanup scope: <known old surfaces or "not in scope"; include expected remove/refactor/retain proof when relevant>
- Fan-in owner: orchestrator reconciles lane outputs; this lane does not approve final decisions.

Inspect first:
- <small ordered list of files, directories, docs, or diffs>

Question:
- <the exact question this lane owns>

Evidence requirement:
- Cite exact files, artifact sections, commands, or source facts.
- Separate facts from assumptions and inferences.
- When dependency choice or custom implementation is in scope, compare current Go stdlib, established repo patterns, and mature OSS options; cite maintenance/release activity, adoption such as stars or domain-equivalent signals, license, security posture, transitive dependency cost, API stability, fit, selected option, rejected options, and custom-code justification when applicable.
- When design/system patterns are in scope, compare known applicable patterns; cite concrete pattern descriptions and real-use examples, task applicability, Go/repository fit, selected pattern, rejected patterns, and custom-design justification when no pattern fits.
- When contract design is in scope, return whether `design/contracts/`, compact contract design, `not_expected`, or `blocked` is the right checkpoint result; cite source-of-truth authority and the resource/status/error/retry/async/freshness/compatibility decision that planning must preserve.
- When cleanup is in scope, report unexplained surviving replaced or unused surfaces and classify each as removed, refactored, retained with owner/reason/proof/exit condition, not applicable, or reopen risk.
- Do not invent missing artifacts or validation results.

Expected output:
- If the chosen skill defines an output shape, follow that shape.
- Otherwise use the shared envelope from docs/subagent-contract.md:
  Decision or findings / Evidence / Open risks or gaps / Recommended handoff / Confidence.
- For material review or challenge findings, include the fan-in destination, owner or reopen target, and why the severity is not stronger or weaker.
- When adjacent domains are touched, prefer classifying each major point as `must_decide_now`, `constraint_only`, `proof_only`, or `follow_up_only`.
- Recommended handoff must use one classification:
  spawn_agent, reopen_phase, needs_user_decision, accept_risk, record_only, or no_action.
```

Short variant:

```text
Use <agent-name> in <mode> with <skill-name | no-skill>.
Read-only enforcement: <read-only execution choice>; no edits, no git mutation, no task-ledger or handoff changes.
Question: <exact question>.
Inspect first: <paths>.
Evidence: cite concrete files/artifacts/commands; label assumptions; when dependency/custom-code choice is in scope, include stdlib/repo-pattern/OSS comparison with current maintenance, adoption, license, security, and fit signals; when pattern choice is in scope, include known-pattern comparison with source descriptions, examples, task applicability, Go fit, and rejected alternatives; when contract design is in scope, include checkpoint result, runtime source of truth, generated outputs, compatibility class, proof carrier, and reopen trigger.
Legacy cleanup: if in scope, report any unexplained surviving old surfaces and retained-surface proof.
Return: skill output shape, or docs/subagent-contract.md envelope with one handoff classification.
Prefer `must_decide_now` / `constraint_only` / `proof_only` / `follow_up_only` for adjacent-domain effects when relevant.
```

Short challenge/review variant:

```text
Use <agent-name> for read-only <challenge | review> with <skill-name | no-skill>.
Scope: <artifact, diff, or decision being challenged/reviewed>.
Lens: <specific approval-risk lens or specialist domain when part of multi-lane fan-out>.
Question: <one approval-, risk-, or correctness-critical question>.
Inspect first: <small path list>.
Evidence: cite exact files/artifacts/commands; separate facts from assumptions; when dependency/custom-code choice is in scope, include stdlib/repo-pattern/OSS comparison with current maintenance, adoption, license, security, and fit signals; when pattern choice is in scope, include known-pattern comparison with source descriptions, examples, task applicability, Go fit, and rejected alternatives.
Legacy cleanup: if replacement work is in scope, report retired surfaces still present without approved retention owner/reason/proof/exit condition.
Return: findings only, ordered by impact, with one recommended handoff classification.
Do not edit files, mutate git state, approve decisions, or change task ledgers/handoffs.
```

Specification review variant:

```text
Use <reviewer-agent | specialist-agent> for read-only specification review with <specialist skill | no-skill>.
Gate: specification review after `spec.md` is review-ready and before technical design, planning, or implementation.
Scope: <task>/spec.md and named supporting artifacts.
Question: Is this `spec.md` complete, deep, and explicit enough for the next phase, or must specification/research/user decision reopen?
Inspect first:
- <task>/spec.md
- <task>/workflow-plan.md and <task>/workflow-plans/specification*.md when present
- <task>/research/*.md or formal clarification fan-in when the spec relies on them
- source-of-truth artifacts named by the spec
Evidence: cite concrete artifact sections and source facts; label assumptions.
Coverage: check scope/non-goals, behavior/contract delta, product/operator expectations, domain invariants, edge cases, API/data/source-of-truth effects, dependency/OSS diligence, Pattern Fit Diligence, legacy-surface handling, security/reliability/delivery implications, validation proof obligations, and downstream handoff clarity.
Lens coverage: for each considered lens, return `Lens | Trigger/source | Owned readiness question | Falsification check | Status | Evidence pointer | Reason | Disposition`, where status is `covered`, `not_applicable`, `concern`, or `fail`. `covered` means the lane tried to disprove readiness for that lens; related prose alone is insufficient.
Finding format: `Spec anchor`, `Evidence`, `Impact`, `Decision owner`, `Primary classification`, `Owner/reopen target`, `Why not stronger/weaker`, `Required disposition`.
Return: Findings classified as `blocks_spec_approval`, `reopens_specification`, `reopens_research`, `requires_user_decision`, `accepted_risk_candidate`, `proof_obligation`, or `record_only`; required fixes or reopen targets; accepted-risk candidates; downstream proof obligations; recommended gate result: PASS | CONCERNS | FAIL with status rationale.
Gate decision order: any `fail` lens, missing required spec decision, or unresolved blocker classification means FAIL; otherwise bounded accepted risks or proof obligations mean CONCERNS; otherwise covered or justified not-applicable readiness-critical lenses mean PASS.
Read-only enforcement: <read-only execution choice>; no edits, no git mutation, no approval authority, no task-ledger or implementation handoff changes.
```

Technical design review variant:

```text
Use <design-integrator-agent | specialist-agent> for read-only technical design review with <go-design-spec | specialist skill | no-skill>.
Gate: technical design review before planning.
Scope: <system/integration and Go code ownership design packet being reviewed>.
Question: Is this design packet coherent and safe enough for planning, or must system/integration design, Go code ownership design, or specification reopen?
Inspect first:
- <task>/spec.md
- <task>/design/overview.md
- <task>/design/system-integration.md when triggered
- <task>/design/go-code-ownership.md when triggered
- <task>/design/<triggered artifacts>
- <task>/workflow-plan.md and <task>/workflow-plans/system-integration-design.md, <task>/workflow-plans/go-code-ownership-design.md, <task>/workflow-plans/technical-design-review.md when present
- docs/repo-architecture.md when boundaries, ownership, dependency direction, or runtime flow matter
Evidence: cite concrete artifact sections and source facts; label assumptions.
Planning-safety check: name the first task-planning decision that would still require architecture, system behavior, contract shape, package/file ownership, sequencing, rollout, validation, cleanup, or test-ownership judgment. If any exists, classify it as a planning blocker and name the owning reopen target.
Planning blocker test: what would `tasks.md` have to invent before it could name task sources, owner files/packages, order, proof, checkpoint, cleanup/test obligations, or stop/reopen conditions?
System-handoff falsification: when system/integration design is triggered, verify each planning-critical mechanism has selected or preserved behavior, source-of-truth owner, affected runtime or failure branch, code-carrying constraint, rejected live alternative and closure rule, proof carrier, and reopen trigger; missing or contradictory fields are `blocks_planning` unless not-applicable is evidenced.
Decision quality: for each material finding, state the planning decision at risk, the strongest counterargument or simpler alternative considered, and why the severity/gate result is not stronger or weaker.
Finding format: include owner or reopen target, planning impact, primary classification, and why the issue is not stronger or weaker.
Return: Findings classified as `blocks_planning`, `reopens_design`, `reopens_spec`, `accepted_risk_candidate`, `proof_obligation`, or `record_only`; required fixes or reopen targets; accepted-risk candidates; planning proof obligations; recommended gate result: PASS | CONCERNS | FAIL with status rationale.
Follow-up after FAIL: include `Prior finding | Repair/evidence anchor | Rechecked areas | Closure status | Residual proof obligation/reopen target`.
Read-only enforcement: <read-only execution choice>; no edits, no git mutation, no approval authority, no task-ledger or implementation handoff changes.
```

Multi-challenger clarification variant:

```text
Shared candidate bundle:
- Workflow phase: <specification phase or equivalent>
- Candidate spec: <path>
- Related artifacts: <workflow-plan/design/research/tasks paths if relevant>
- Shared constraints/non-goals: <short list>
- Sibling lenses: <names of the other planned challenge lenses, so this lane avoids duplicate coverage>
- Fan-in owner: orchestrator reconciles all lane outputs; lanes do not approve decisions.
- Scoped-down rationale: <required when using fewer lanes than the broad formal default; list default lenses considered, retained lanes, and why omitted lenses cannot change approval>

Lane:
- Agent: challenger-agent
- Mode: challenge
- Skill: spec-clarification-challenge
- Lens: <scope/spec coherence | domain invariants | architecture ownership | API/data/compatibility | security/reliability/delivery/validation | custom lens>
- Question: <one concrete approval-critical question for this lens; not a generic topic label>
- Inspect first: <small path list, plus lens-specific files only>
- Evidence: cite exact files/artifacts/commands; separate facts from assumptions.
- Return: spec-clarification-challenge shape, with the lens named and only the strongest non-duplicative findings/questions ordered by approval impact.
- Read-only enforcement: <read-only execution choice>; no edits, no git mutation, no task-ledger or handoff changes.
```
