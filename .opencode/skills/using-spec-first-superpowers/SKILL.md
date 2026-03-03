---
name: using-spec-first-superpowers
description: "Run mandatory pre-turn routing for this repository's spec-first workflow. Use on every user message to classify intent and phase, select required skills, enforce gate constraints, and decide route_pass/route_lightweight/route_blocked before any response or action."
---

# Using Spec-First Superpowers

## Purpose
Execute mandatory pre-turn routing (`M0`) before any response or action. Success means each turn has a deterministic `Routing Record` and an explicit decision: `route_pass`, `route_lightweight`, or `route_blocked`.

## Scope And Boundaries
In scope:
- classify incoming requests by `intent`
- detect current workflow `phase` and `gate_state`
- select `required` and `optional` skills using `phase x intent` mapping
- enforce routing constraints (`Spec Freeze`, gate prerequisites)
- return explicit next action for downstream execution

Out of scope:
- architecture, API, data, security, reliability, or business decisions
- code/test implementation
- code review findings
- replacing downstream skill outputs with orchestration judgments
- hidden bypass of `M0`
- editing specs/code without routed skill context

## Hard Skills
### Routing Core Instructions

#### Mission
- prevent any action before routing discipline is applied
- keep skill selection deterministic, explainable, and gate-safe
- route to domain expertise quickly after control checks

#### Default Posture
- routing first, work second
- if there is even a minimal chance a skill applies, include it as a candidate
- prefer safer/stricter path when intent or phase is ambiguous
- block unsafe execution instead of guessing through gate constraints
- allow skip only when a complete routing record already exists for the same turn

#### M0 Gate Competency
- execute `M0` on every user turn before responding or taking actions
- `M0` must produce a complete `Routing Record`
- no bypass for "simple questions"; they may use `route_lightweight`, but only after `M0`

#### Intent Classification Competency
- classify requests into one primary intent:
  - `new_feature_or_behavior_change`
  - `spec_enrichment`
  - `implementation`
  - `test_implementation`
  - `code_review`
  - `bug_or_failing_test`
  - `informational_question`
  - `workflow_meta_question`
- if multiple intents are possible, choose the highest-risk intent
- if risk is equal across intents, choose the stricter process path
- when uncertainty is material, mark `[assumption]` and choose stricter routing

#### Phase And Gate Competency
- detect phase from artifacts and thread context:
  - `Phase -1` (if adopted), `0`, `1`, `2`, `2.5`, `3`, `4`, `5`
- enforce gate constraints:
  - coding requires `G2.5` readiness
  - `Spec Freeze` prevents spec edits unless explicit reopen path is active
  - review path must stay within `*-review` skill class
- if phase/gate cannot be safely inferred:
  - mark `[assumption]`
  - choose the safest path
  - return `route_blocked` when residual risk remains high

#### Skill Selection Competency
- build candidate list from `phase x intent`
- assign `required_skills` and `optional_skills`
- define explicit `selected_order` using this priority:
  - process/safety skills
  - phase-mandatory skills
  - domain skills
  - optional refinement skills
- map default intents:
  - `new_feature_or_behavior_change` -> `spec-first-brainstorming`, then `go-architect-spec`
  - `spec_enrichment` -> `go-architect-spec` + relevant `*-spec`
  - `implementation` -> `go-coder` (only with `G2.5 pass`)
  - `test_implementation` -> `go-qa-tester`
  - `code_review` -> relevant `*-review` skills
  - `bug_or_failing_test` -> `go-systematic-debugging`
  - `informational_question` -> `route_lightweight` allowed when no artifact/code change is requested
  - `workflow_meta_question` -> `go-architect-spec` or governance process path

#### Escalation Competency
- enforce `route_blocked` when:
  - coding is requested without `G2.5 pass`
  - spec changes are requested during `Spec Freeze` without `Spec Reopen`
  - review action is requested outside review path or outside `*-review` skill class
  - contract/architecture risk is introduced without returning to spec phase
- route to `Spec Clarification Request` when Phase 3 ambiguity affects architecture/contract intent
- route to `Spec Reopen` when review finds spec-level mismatch
- block actions that require escalation but try to proceed without it

#### Review Blockers For This Skill
- action/response happened before `M0`
- missing or partial `Routing Record`
- no explicit route decision (`route_pass`/`route_lightweight`/`route_blocked`)
- gate violations ignored (`G2.5`, `Spec Freeze`, review class constraints)
- selected skills lack ordering or justification

## Working Rules
1. If a complete routing record already exists for the same turn, reuse it and do not re-run `M0`.
2. Read the user turn and determine whether a feature context already exists.
3. Infer `phase` and `gate_state` from artifacts and thread context.
4. Classify primary `intent`.
5. Build skill candidates by `phase x intent`.
6. Apply minimal-probability applicability rule and keep relevant candidates.
7. Assign `required_skills`, `optional_skills`, and explicit `selected_order`.
8. Decide one route:
   - `route_pass`
   - `route_lightweight`
   - `route_blocked`
9. If blocked, include reason and minimum unblock condition.
10. Emit `Routing Record`.
11. Only after that, execute downstream skills or answer.

## Output Expectations
Use this section order:

```text
Intent
Phase
Gate State
Routing Record
Decision
Constraints
Next Action
```

`Routing Record` must include:
- `intent`
- `phase`
- `gate_state`
- `required_skills`
- `optional_skills`
- `selected_order`
- `decision`
- `reason`
- `constraints`
- `next_action`

Output rules:
- decision must be exactly one of `route_pass`, `route_lightweight`, `route_blocked`
- `required_skills` and `selected_order` must be explicit
- `route_lightweight` must explain why heavy skill-chain is not required
- `route_blocked` must include reason and minimum unblock condition

## Definition Of Done
- `M0` was executed for the turn
- full `Routing Record` is present
- explicit route decision was made
- no action was taken in contradiction to routing decision

## Anti-Patterns
- responding before `M0`
- skipping routing because request "looks simple"
- selecting skills without reason or order
- ignoring gate constraints
- using `route_lightweight` for requests that change specs/code/contracts
- orchestration skill making domain decisions

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when intent, phase, gate constraints, and safe routing choice are unambiguous.

Always load:
- `docs/spec-first-workflow.md`:
  - read phase sequence, gate criteria, and freeze/escalation rules
- `AGENTS.md`:
  - read dynamic loading policy and execution loop
- `docs/skills/using-spec-first-superpowers-spec.md`
- available skill registry from `skills/*` (and mirrors when needed for parity checks)

Load by trigger:
- routing matrix tuning or governance discussions:
  - `docs/skills/spec-first-superpowers-integration.md`
- feature framing entry path:
  - `skills/spec-first-brainstorming/SKILL.md`
- spec phase routing:
  - `skills/go-architect-spec/SKILL.md`
- implementation routing:
  - `skills/go-coder/SKILL.md`
  - `skills/go-qa-tester/SKILL.md`
- debug routing:
  - `skills/go-systematic-debugging/SKILL.md`
- review routing:
  - relevant `skills/*-review/SKILL.md` for affected domain
- if active feature artifacts exist:
  - `specs/<feature-id>/80-open-questions.md`
  - `specs/<feature-id>/90-signoff.md`
  - `reviews/<feature-id>/code-review-log.md` (when in review/reopen context)

Unknowns:
- if critical routing facts are missing, mark `[assumption]` and take stricter path
- if risk remains high after assumptions, return `route_blocked`

## Enforcement Limit
This skill enforces strong process policy at instruction level, not hard runtime guarantees. Hard guarantees require an external validator/orchestrator to verify `M0` and block policy-violating turns.
