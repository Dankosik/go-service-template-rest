---
name: grilling
description: Conduct an exhaustive, one-question-at-a-time stress test of a plan or design only when the user explicitly asks to grill, stress-test, challenge every branch, or run an exhaustive design interview. Do not use for ordinary Phase 0 clarification.
---

# Grilling

Use this skill only for an explicit request to pressure-test a plan or design exhaustively. Ordinary ambiguity, risk, or an unresolved user-owned intake decision is not a trigger; follow the minimal Phase 0 rules instead.

Before asking, reconstruct the current plan, its decided constraints, and its unresolved decision branches. If a question is repository-factual, answer it through bounded codebase inspection instead of asking the user.

Walk every material unresolved branch that could change the plan or build choice. Resolve dependent decisions in order, but do not reopen a settled branch unless a later answer conflicts with it.

Ask exactly one question at a time and wait for the answer before continuing. For every question:

- state which plan or build choice the answer can change;
- provide the recommended answer and the main tradeoff;
- avoid generic checklist questions and hypothetical branches that cannot affect the plan.

After each answer, update the working decision tree and choose the next highest-impact unresolved branch.

Stop when no unresolved decision remains that can materially change the plan or build choice. Then summarize the resolved decisions, bounded assumptions, residual risks, and any condition that would reopen the plan. Do not continue questioning merely to cover more categories.
