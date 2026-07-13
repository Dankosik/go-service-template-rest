---
name: spec-first-brainstorming
description: "Turn a chosen or mostly chosen feature/refactor direction into an engineering-ready problem frame with behavior delta, scope, constraints, assumptions, prioritized questions, and an explicit readiness decision. Use after idea-refine or when framing remains unclear before specialist spec work; skip raw ideation, final design decisions, and task breakdown."
---

# Spec-First Brainstorming

## Outcome

Produce a compact, falsifiable problem frame that downstream challenge and specification work can consume without mistaking a proposed implementation for the desired behavior.

## Method

1. Normalize the request into actor, current behavior, desired behavior, and the smallest behavior delta.
2. Separate in-scope outcomes, non-goals, and constraints that materially shape later decisions.
3. Label critical assumptions, risk, validation path, owner, and unblock condition; ask only questions that can change readiness.
4. Compare `2-3` behavior-level directions only when a real framing fork remains, then recommend one or fail readiness.
5. Classify pre-spec challenge as `required`, `recommended`, or `skippable` and name the `1-3` seams worth testing.

## Decision Rules

- Prefer outcome over proposed mechanism and one chosen direction over an unresolved menu.
- Keep statements concrete and testable; do not hide uncertainty in generic prose.
- Do not choose final architecture, API, data, security, reliability, rollout, implementation, or task-order decisions.
- Reopen `idea-refine` when product directions remain live; route material external facts or specialist choices to their owner.
- Mark challenge `required` only when assumptions, ownership, edge/failure semantics, or rollback risk can still change design; mark it `skippable` only for sharply bounded low-risk work.

## Reference Selector

Load at most one reference by default; load more only for independent framing pressures.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| The prompt is a mechanism, slogan, or vague improvement. | [problem-and-behavior-delta.md](references/problem-and-behavior-delta.md) | Write a falsifiable actor/current/desired delta. |
| Scope expands or constraints/non-goals are vague. | [scope-constraints-and-non-goals.md](references/scope-constraints-and-non-goals.md) | Separate approved scope from adjacent ideas. |
| Implied facts, risky assumptions, or question piles remain. | [assumptions-and-open-questions.md](references/assumptions-and-open-questions.md) | Route labeled uncertainty to an owner and unblock condition. |
| Several behavior-level frames remain plausible. | [approach-comparison-and-direction-selection.md](references/approach-comparison-and-direction-selection.md) | Recommend or block one direction without designing architecture. |
| Challenge routing is hard to classify. | [challenge-recommendation-examples.md](references/challenge-recommendation-examples.md) | Tie challenge need to concrete risk seams. |
| Readiness is close or drifting into “ready enough.” | [readiness-decision-examples.md](references/readiness-decision-examples.md) | Emit a decisive pass/fail and next owner. |
| The draft smuggles downstream decisions or ceremony. | [framing-anti-patterns.md](references/framing-anti-patterns.md) | Remove design/task/stakeholder theater from the frame. |

## Output

Return: `Problem`, `Behavior Delta`, `Scope`, `Constraints`, `Assumptions`, `Open Questions`, `Challenge Recommendation`, `Readiness Decision`, and `Handoff`. Add `Approaches` only for a real unresolved framing fork.

## Success And Stop

Return `pass` only when behavior, scope, constraints, critical unknowns, question ownership, and challenge route are explicit enough for the next phase. Return `fail` with the minimum missing data or upstream owner when product direction, actors, behavior, high-risk semantics, or material constraints remain unstable.
