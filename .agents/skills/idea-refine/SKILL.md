---
name: idea-refine
description: "Refine raw product or feature ideas into one concrete direction worth taking into engineering. Use when the request is still idea-shaped, solution-led, or ambiguous at the user/problem level and the repository is not ready for `spec-first-brainstorming` or specialist spec work yet. Skip once the direction is already chosen and the task mainly needs engineering framing."
---

# Idea Refine

## Purpose
Turn a raw concept into one concrete direction that is specific enough to hand off into engineering framing and later spec work.

## Scope
- restate the idea as a user/problem statement instead of a proposed implementation
- identify the target user or operator, desired outcome, and success signal
- explore a small set of plausible directions, then converge on one recommendation
- surface critical assumptions, MVP boundaries, and a clear `Not Doing` list
- produce a compact handoff artifact that can feed `spec-first-brainstorming`

## Lazy Reference Loading
Keep this skill compact. Load only the reference file that matches the uncertainty in front of you:

- `references/problem-vs-solution-framing.md` when the idea starts as a feature, implementation, or "we need X" request and the underlying problem is still blurry.
- `references/target-user-and-success-signal.md` when the actor, operator, job, or success signal is vague or output-shaped.
- `references/direction-options-and-convergence.md` when several product directions are plausible and the pass needs a defensible recommendation.
- `references/assumptions-and-kill-criteria.md` when the idea depends on weak assumptions, risky beliefs, or untested value claims.
- `references/mvp-scope-and-not-doing.md` when scope is expanding, the MVP is unclear, or the `Not Doing` list needs sharper boundaries.
- `references/spec-first-handoff-examples.md` when the refined idea is ready to hand off into `spec-first-brainstorming`.

## Boundaries
Do not:
- write final architecture, API, data, security, reliability, or rollout decisions
- turn the output into a task breakdown or coder plan
- keep several equally valid directions alive just to avoid choosing
- hide weak assumptions behind generic enthusiasm
- ignore real repository constraints when the idea clearly lands in this codebase

## Escalate When
Escalate if:
- no clear user or operator can be identified
- success cannot be described in concrete terms
- the idea hides high-risk semantics like destructive actions, money movement, identity, privacy, or irreversible state and needs explicit policy input
- the user is really asking for engineering framing, not ideation
- multiple directions remain viable but no recommendation can be defended

## Core Defaults
- Prefer one recommended direction over a menu of half-committed options.
- Prefer the smallest meaningful MVP that can validate the core bet.
- Prefer explicit trade-offs over aspirational scope.
- Prefer a short list of strong assumptions over a longer list of vague unknowns.
- Prefer saying what we are not doing yet; focus is part of the deliverable.

## Workflow

### 1. Clarify The Real Problem
- Restate the idea as a user/problem statement.
- Make the target user or operator explicit.
- Name the desired outcome and what success looks like.
- If the request is still too vague, ask only the minimum questions needed to remove ambiguity.

### 2. Expand, Then Converge
- Explore a small set of genuinely different directions.
- Stress-test them against:
  - user value
  - feasibility in this repository and stack
  - simplicity of MVP scope
  - differentiation or real payoff
- Converge on one direction instead of carrying several into spec work.

### 3. Surface The Bets
- List the assumptions that must be true for the recommendation to hold.
- Name what could kill the idea or force a different direction.
- Mark constraints that the later spec work must preserve.

### 4. Define The Handoff Boundary
- State the MVP scope in plain language.
- State the `Not Doing` list explicitly.
- End with what engineering framing should do next, not with implementation detail.

## Deliverable Shape
Return ideation work in this order:
- `Problem`
- `Target User / Operator`
- `Recommended Direction`
- `Why This Direction`
- `Key Assumptions To Validate`
- `MVP Scope`
- `Not Doing`
- `Open Questions`
- `Next Handoff`

`Next Handoff` should usually point to `spec-first-brainstorming`.

## Artifact Guidance
- Keep the result inline when the conversation is short and the handoff is obvious.
- When the result should be preserved, save it under `specs/<feature-id>/research/idea-refine.md` or another user-approved location.
- Do not invent a feature folder only for archival neatness.

## Definition Of Done
The pass is complete when:
- one recommended direction exists
- the user/problem and success signal are explicit
- the MVP boundary is narrow enough to guide framing
- the `Not Doing` list removes obvious scope creep
- the output is ready to hand off into `spec-first-brainstorming`

## Escalate Or Reject
- brainstorming that never converges on one direction
- shipping implementation detail instead of idea refinement
- a yes-machine response that never challenges weak assumptions
- handoff output with no MVP boundary or `Not Doing` list
