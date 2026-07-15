---
name: grilling
description: Stress-test a plan, decision, idea, or design one material branch at a time. Use when the user explicitly requests grilling, uses a grill trigger phrase, or the workflow invokes internal macro-phase grilling on the existing read-only challenger. Do not use for ordinary Intake clarification.
---

# Grilling

Choose the mode from the request. Ordinary ambiguity, risk, or an unresolved
user-owned Intake decision triggers neither mode; follow the minimal Intake
rules instead.

Reconstruct the exact subject being grilled, including any candidate, accepted
decisions and constraints, evidence and authority boundaries, unresolved
branches, and stop rule. Inspect facts available in the environment (filesystem,
repository, tools) instead of asking another actor to supply them. Select the
highest-impact new or evidence-reopened material branch whose answer can change
the subject, current plan, or phase decision. Resolve dependent branches in
order; skip settled or immaterial branches unless conflicting evidence reopens
them.

## Explicit user mode

Use this mode only when the user explicitly asks to grill or stress-test a plan,
decision, idea, or design; challenge every branch; or run an exhaustive
interview. It is a root-to-user dialogue; never relay it through the internal
challenger.

Ask exactly one question and wait for the answer. For that question:

- state which choice or outcome the answer can change;
- provide the recommended answer and the main tradeoff;
- avoid generic checklist questions and hypothetical branches that cannot affect the plan.

After each answer, update the working decision tree and choose the next highest-impact unresolved branch.

Stop when no unresolved decision remains that can materially change the subject
or outcome. Then summarize the resolved decisions, bounded assumptions, residual
risks, and any condition that would reopen it. Do not continue questioning merely
to cover more categories. Do not act on it until the user confirms that you have
reached a shared understanding.

## Internal challenger mode

Use this mode only when the workflow launches the existing read-only challenger
for an autonomous pre-review probe. Use the branch-selection method above, then
follow the [canonical protocol](../../../docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md)
for event shape, authority, continuation, exhaustion, invalidation, and reviewer
separation. Return only the event that owner permits for the current turn.

The challenger may apply a materially triggered specialist method locally. It
does not edit the candidate, delegate, issue a readiness verdict, serve as the
required reviewer, or create a transcript, receipt, queue, status, or lifecycle
artifact.
