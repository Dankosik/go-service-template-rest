---
name: spec-clarification-challenge
description: "Run a specification-approval clarification challenge for non-trivial `spec.md` work. Use inside a read-only challenger subagent during `specification`, after candidate decisions exist and before `spec.md` approval, to surface non-obvious questions, hidden assumptions, corner cases, architecture/data/API/security/reliability seams, and approval blockers the orchestrator must reconcile."
---

# Spec Clarification Challenge

## Purpose
Surface the few questions that could still make `spec.md` approval dishonest.

This skill is a gate inside `specification`, not a workflow phase and not a second design document. It gives the orchestrator approval-focused questions to answer from evidence, route to targeted expert research, defer explicitly, or record as accepted risk.

## Scope
- inspect candidate decisions that are already close to `spec.md` approval
- find non-obvious questions that could change scope, acceptance semantics, architecture boundaries, source-of-truth ownership, data/API/security/reliability behavior, failure semantics, rollout, or validation strategy
- classify each question by approval impact and recommend the smallest next action
- keep the output compact enough for direct orchestrator reconciliation

## Boundaries
Do not:
- make final product, architecture, API, data, security, reliability, rollout, or validation decisions
- edit files, write `spec.md`, update workflow plans, mutate git state, or alter the implementation plan
- ask the human by default; return questions for the orchestrator to reconcile
- produce a parallel design document, task breakdown, or research transcript
- repeat generic checklist prompts such as "what about security?" unless tied to a concrete seam in the candidate decisions
- reopen settled scope without showing why a different answer would change spec approval

## Required Input Bundle
Expect a compact bundle from the orchestrator:
- problem frame
- scope and non-goals
- candidate decisions
- constraints and validation expectations
- known assumptions and open questions
- links to relevant `research/*.md` or lane outputs when they matter

If the bundle is too thin to challenge, say what is missing and classify the result as blocking approval rather than guessing.

## Question Selection
Prefer 5-10 high-signal questions for complex work. Fewer is fine when fewer questions materially affect approval.

Keep a question only if all are true:
- it names a specific hidden assumption, corner case, or seam
- a different answer could change `spec.md` scope, acceptance semantics, ownership, failure behavior, rollout, or validation
- the orchestrator could answer it from evidence or route it to a targeted expert lane
- it is not ordinary downstream design elaboration

Drop questions that only ask for best-practice coverage, implementation detail, or "more thinking" without changing approval.

## Classification
Use exactly one:
- `blocks_spec_approval` when `spec.md` cannot be honestly approved until the answer is resolved, accepted as risk, or marked as an upstream blocker
- `blocks_specific_domain` when approval depends on reopening one expert domain such as API, data, security, reliability, domain, QA, delivery, or architecture
- `non_blocking_but_record` when the point should be explicit in `spec.md` but does not block approval if recorded, deferred, or accepted as risk

## Next Action
Use exactly one:
- `answer_from_existing_evidence` when the orchestrator should resolve it from current repo evidence, research notes, or candidate synthesis
- `targeted_research` when local repository or external retrieval should answer a bounded factual gap
- `expert_subagent` when one read-only specialist lane should reopen a domain question using one skill
- `accept_risk` when the current path remains coherent and the remaining uncertainty is a conscious trade-off
- `defer_to_design` when the spec can be approved with an explicit constraint and the detail belongs in `design/`
- `requires_user_decision` when the question is truly external product, business, policy, or legal judgment that repo evidence and safe assumptions cannot answer

Do not use `requires_user_decision` for questions the orchestrator can answer from repository evidence or expert research. If used, explain why the spec should remain blocked or partially draft until the user decision exists.

## Reconciliation Expectations
The orchestrator owns reconciliation after the subagent returns:
- answer each planning-critical question from evidence where possible
- reopen targeted research or one read-only expert subagent per expert question when evidence is missing
- record final resolved outcomes in existing `spec.md` sections, not raw subagent transcripts
- update `workflow-plans/specification.md` with clarification challenge status, lane used, targeted research status, resolution status, and approval or block rationale
- update `workflow-plan.md` with `spec.md` status and clarification gate status
- rerun this challenge once if material decisions changed or a major seam was reopened

## Lazily Loaded Examples
Keep this file compact. Load only the reference that matches the uncertainty in the current challenge:

- `references/approval-blocker-question-examples.md` for approval-changing hidden assumptions and blocker wording.
- `references/input-bundle-sufficiency.md` when the orchestrator's bundle may be too thin to challenge honestly.
- `references/domain-reopen-classification.md` when deciding between `blocks_spec_approval` and `blocks_specific_domain`.
- `references/defer-to-design-vs-block-spec.md` when a question may be downstream design instead of spec-approval work.
- `references/requires-user-decision-examples.md` when the question may need external product, business, policy, or legal judgment.
- `references/clarification-anti-patterns.md` when pruning generic checklist questions, answer-writing, design drift, or approval theater.

Use references as examples, not templates to fill. Do not copy an example unless the same candidate-decision seam is present.

## Deliverable Shape
Return:
- `Clarification Summary`
- `Questions`
- `Reopen / Rerun Recommendation`
- `Confidence`

For each item in `Questions`, include:
- `Question / Hidden Assumption`
- `Why It Matters`
- `What Could Change`
- `Classification`
- `Recommended Next Action`
- `Evidence Or Expert Lane`

Keep the wording concise and concrete. If no question survives the filter, say the clarification gate is clear and name the evidence boundary that supports that conclusion.

## Stop Condition
Stop when:
- all approval-changing questions have been surfaced or the input gap is clearly blocking
- each surviving question has a classification and next action
- low-value checklist items have been pruned
- the output is short enough for the orchestrator to reconcile without reading a second spec

## Anti-Patterns
- writing the answer instead of the approval question
- asking broad category questions with no candidate-decision seam
- padding to hit a quota
- treating `defer_to_design` as a way to hide missing spec decisions
- using `requires_user_decision` to avoid targeted research
- copying raw challenge output into `spec.md`
- turning the pass into architecture authorship, implementation planning, or approval theater
