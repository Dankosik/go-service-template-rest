---
name: idea-refine
description: "Refine a raw, solution-led, or ambiguous product/feature idea into one concrete direction worth engineering. Use before spec-first-brainstorming when the user/problem, success signal, core bet, or MVP boundary is still unsettled; skip once the direction is chosen and only engineering framing remains."
---

# Idea Refine

## Outcome

Choose one defensible product direction with an explicit user/problem, success signal, core assumptions, narrow MVP, and `Not Doing` boundary ready for engineering framing.

## Method

1. Reframe the proposed feature/tool as a problem for one primary user or operator and one observable outcome.
2. Ask at most the gating questions needed to distinguish materially different directions; use labeled safe assumptions for the rest.
3. Compare a small set of genuinely different directions against user value, repository feasibility, learning value, and MVP simplicity.
4. Recommend one direction, name the bets and kill/pivot criteria, then set the MVP and `Not Doing` boundary.
5. Hand off product intent and open questions without smuggling architecture, endpoints, data models, rollout, or tasks.

## Decision Rules

- Prefer one direction over a menu and a learning-shaped MVP over a platform bundle.
- Keep weak assumptions falsifiable; do not hide them behind enthusiasm.
- Use repository facts as constraints when the idea clearly lands here, but do not turn ideation into implementation design.
- Escalate money, identity, privacy, destructive action, or irreversible-state policy when it changes the product direction.

## Reference Selector

Load at most one reference by default; load a second only for an independent product pressure.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| The idea is a feature/tool/vendor while the pain is blurry. | [problem-vs-solution-framing.md](references/problem-vs-solution-framing.md) | Reframe to user/operator problem. |
| Actor or success signal is generic. | [target-user-and-success-signal.md](references/target-user-and-success-signal.md) | Choose one actor and behavior/outcome signal. |
| Thin input tempts a long questionnaire. | [clarifying-questions-and-safe-assumptions.md](references/clarifying-questions-and-safe-assumptions.md) | Ask only gating questions and label assumptions. |
| Several directions or a feature grab bag remain. | [direction-options-and-convergence.md](references/direction-options-and-convergence.md) | Compare and converge on one direction. |
| Demand, trust, policy, or “AI magic” is uncertain. | [assumptions-and-kill-criteria.md](references/assumptions-and-kill-criteria.md) | State falsifiable bets and pivot criteria. |
| MVP or `Not Doing` is vague. | [mvp-scope-and-not-doing.md](references/mvp-scope-and-not-doing.md) | Set a learning-shaped boundary. |
| The direction is ready for engineering framing. | [spec-first-handoff-examples.md](references/spec-first-handoff-examples.md) | Hand off intent without downstream design. |

## Output

Return: `Problem`, `Target User / Operator`, `Recommended Direction`, `Why This Direction`, `Key Assumptions To Validate`, `MVP Scope`, `Not Doing`, `Open Questions`, and `Next Handoff`. `Next Handoff` normally points to `spec-first-brainstorming`.

Keep the result inline unless persistence is already authorized and useful. Do not invent a feature folder for archival neatness.

## Success And Stop

Success means one direction, actor/problem, outcome signal, MVP boundary, exclusions, and critical bets are explicit. Stop or escalate when no actor or success signal can be identified, several directions remain equally live, or a missing high-risk policy decision prevents a defensible recommendation.
