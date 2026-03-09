---
name: pre-spec-challenge
description: "Pressure-test candidate decisions before planning with discriminating questions about hidden assumptions, corner cases, ownership seams, failure semantics, and rollout risk. Use whenever research is done but the orchestrator wants an independent challenge pass before decisions harden into the spec, even if the user only says 'critique this', 'ask the hard questions', or 'make sure we did not miss edge cases.'"
---

# Pre-Spec Challenge

## Purpose
Pressure-test candidate decisions before planning so the orchestrator learns which assumptions are still unsafe to carry forward.

When used from a project agent such as `challenger-agent`, let the agent own ownership, trigger rules, boundaries, and handoffs. This skill owns the challenge behavior: how to falsify assumptions, prune low-value questions, classify blocker severity, and stop once planning risk is well bounded.

## Scope
- inspect candidate synthesis, not a blank request
- challenge only seams that could still change scope, correctness, ownership, failure semantics, or rollout
- convert uncertainty into the smallest actionable next step
- keep the pass compact enough that the orchestrator can reconcile it directly

## Boundaries
Do not:
- make final product, architecture, API, data, security, or rollout decisions
- rewrite the whole scope or reopen already-settled decisions without concrete evidence
- ask generic category questions with no seam attached
- ask the user directly; recommend `ask_user` only when the orchestrator truly lacks an external fact
- produce a second design document that competes with `spec.md`

## Escalate When
Escalate if:
- candidate synthesis is too thin to challenge meaningfully
- the real problem is missing framing rather than missing pressure-test
- one challenged point clearly needs fresh specialist research rather than more questioning
- the candidate path is so contradictory that integration or domain design must happen before challenge can help

## Core Defaults
- Prefer `3-5` strong questions over broader coverage.
- Treat every question as a potential design fork: if the answer changes nothing material, drop it.
- Attack assumptions by trying to falsify them, not by asking for more prose.
- Ask about categories like security, performance, or rollout only through a concrete seam already present in the candidate synthesis.
- Stay advisory. The orchestrator decides.

## Challenge Loop
1. Confirm the input is challenge-ready rather than underframed.
2. Extract the candidate assumptions that are actually carrying the plan.
3. Try to falsify each assumption by asking: what breaks if this is false in production?
4. Keep only the seams where a different answer would materially change planning.
5. Classify each surviving seam before wording the final question.
6. Stop when the unresolved set is no longer planning-critical.

## Question Filter
Keep a question only if all are true:
- it names a specific challenged assumption or seam
- it changes planning, not just later polish
- it would still matter if the orchestrator already knew the general domain best practices
- it is more useful than sending the task straight back to specialist research

If any of those fail, do not ask it.

## Expertise

### Input Sufficiency
- Expect a minimum input bundle: problem frame, candidate decisions, constraints, assumptions or open questions, and any evidence links that matter.
- If that bundle is missing, say so directly and escalate instead of guessing.

### Falsification Targets
- Look first for assumptions disguised as convenience, policy, or “v1 simplification.”
- Pressure-test statements that rely on client behavior, operator workarounds, natural expiry, UUID secrecy, TTLs, or future cleanup.

### Corner Cases And Failure Semantics
- Look for denial, retry, timeout, duplicate request, partial success, stale state, irreversibility, and manual follow-up semantics when they could change the design.
- Ask what the user-visible or operator-visible meaning becomes when the edge condition occurs.

### Ownership And Boundary Seams
- Challenge unclear source-of-truth ownership, actor boundaries, privilege boundaries, and cross-domain side effects.
- Surface when planning would force implementation to “decide later in code.”

### Rollout And Compatibility Pressure Test
- Ask about rollout, migration, backward compatibility, or launch cohort only when the answer can materially change the path.
- Avoid speculative rollout bikeshedding on local or reversible work.

### Blocker Classification
Use:
- `blocks_planning` when planning would be unsafe or misleading without resolution
- `blocks_specific_domain` when the question should reopen only one specialist area
- `non_blocking` when the point is real but can stay as explicit accepted risk or open question

### Next Action Selection
Use:
- `answer` when the orchestrator likely already has enough evidence
- `re-research` when a specialist or retrieval pass should reopen
- `ask_user` when an external policy or product decision is missing
- `defer` when the point is real but can stay explicit without blocking planning
- `accept_risk` when the current path is still coherent and the remaining issue is a conscious trade-off

When `Next Action` is `re-research`:
- name the specialist lane or fact pattern that should be reopened
- state why local orchestrator reasoning is not enough for this seam
- say whether the orchestrator should rerun challenge after the new research returns

## Stop Condition
- Stop once the remaining unresolved questions no longer change planning safety materially.
- If everything left is low-value, already tracked, or belongs in ordinary downstream design elaboration, say the checkpoint is sufficiently reconciled.

## Anti-Patterns
- generic “what about security/performance?” prompting with no seam
- reopening settled scope because “more thought is always good”
- padding the pass with low-value questions to hit a quota
- drifting into architecture authorship instead of pressure-testing the candidate path
- writing commentary that explains the whole design instead of surfacing the few seams that still matter

## Deliverable Shape
Return challenge work in this order:
- `Challenge Summary`
- `Questions`
- `Escalations / Re-research`
- `Confidence`

For each item in `Questions`, include:
- `Question / Challenged Assumption`
- `Why It Matters`
- `What Changes`
- `Blocker Level`
- `Next Action`
- `Research Reopen` when `Next Action = re-research`

## Escalate Or Reject
- a request to nitpick rather than improve planning quality
- a challenge pass on a trivial local task with no material ambiguity
- candidate synthesis that is really a disguised blank page
