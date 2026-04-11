---
name: spec-first-brainstorming
description: "Turn a refined idea or rough feature, refactor, or behavior-change request into an engineering-ready problem frame with scope, constraints, assumptions, prioritized questions, and an explicit design-readiness decision. Use after `idea-refine`, or whenever the direction is mostly chosen but the task still needs crisp framing before specialist spec work or pre-spec challenge. Skip raw ideation, final architecture/API/data/security decisions, and task breakdown."
---

# Spec-First Brainstorming

## Purpose
Turn a mostly chosen request into a concrete, falsifiable, challenge-ready problem frame before deeper design starts.

## Scope
- normalize feature, refactor, or behavior-change requests into a precise problem statement
- identify the behavior delta, affected actors, and relevant system boundaries
- define scope, non-goals, constraints, success criteria, and hidden assumptions
- seed prioritized open questions with owner and unblock condition
- decide whether the request is ready for deeper design and whether a pre-spec challenge pass is required, recommended, or skippable

## Lazy Reference Loading
References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Before loading a reference, name the symptom and the behavior change you need. Load at most one reference by default; load multiple only when the task clearly spans independent decision pressures, such as solution-led framing plus a separate readiness-blocker decision.

| Symptom | Reference | Behavior Change |
| --- | --- | --- |
| The prompt is a proposed mechanism, slogan, or vague improvement and needs actor/current/desired/boundary phrasing. | `references/problem-and-behavior-delta.md` | Choose a falsifiable behavior delta instead of restating the requested implementation as the problem. |
| Scope is expanding, non-goals are vague, or constraints may reshape downstream design. | `references/scope-constraints-and-non-goals.md` | Separate approved framing scope from adjacent ideas and delivery promises instead of letting scope creep enter the handoff. |
| The frame depends on implied facts, missing owner decisions, risky assumptions, or an unprioritized question pile. | `references/assumptions-and-open-questions.md` | Convert uncertainty into labeled assumptions and owner-routed blocking questions instead of inventing answers or listing generic TODOs. |
| Multiple plausible framing approaches remain, but the task is beyond raw ideation. | `references/approach-comparison-and-direction-selection.md` | Compare behavior-level frames and recommend or block one direction instead of carrying every option forward or designing architecture. |
| The pre-spec challenge recommendation is hard to classify or needs sharper seams. | `references/challenge-recommendation-examples.md` | Tie `required`, `recommended`, or `skippable` to concrete risk seams instead of recommending challenge as ritual. |
| The pass/fail readiness decision is close, blocked, or drifting into "ready enough." | `references/readiness-decision-examples.md` | Emit a decisive readiness outcome with next handoff or minimum missing data instead of vague approval. |
| The draft starts smuggling architecture/API/data/security/reliability decisions, task breakdown, implementation design, or stakeholder theater. | `references/framing-anti-patterns.md` | Use this as smell triage to remove downstream design drift instead of treating it as primary framing guidance. |

Treat reference examples as calibration only. Repository-local spec-first workflow rules stay authoritative.

## Boundaries
Do not:
- run open-ended ideation when the real problem is that no concrete direction has been chosen yet; use `idea-refine` first
- make final architecture, API, data, security, reliability, or rollout decisions that belong to downstream specialists
- jump into implementation design, code, or test-writing
- hide ambiguity behind generic wording or unexamined assumptions
- confuse the requested outcome with the user’s proposed implementation idea
- treat challenge routing as optional hand-waving when the framing still has material blind spots

## Escalate When
Escalate if:
- multiple product directions are still live and no recommended direction has been chosen yet
- goals, actors, or behavior change remain ambiguous after focused clarification
- the request sounds local but actually touches money, identity, destructive actions, privacy, or irreversible state
- critical constraints are missing but materially affect design direction
- the discussion is drifting into downstream design decisions that this skill should not own
- the request cannot support a meaningful pre-spec challenge because even the problem frame is still unstable

## Core Defaults
- Prefer outcome over proposed solution.
- Prefer one chosen direction over carrying several product directions into engineering framing.
- Keep statements concrete and testable.
- Prefer explicit blockers over hidden assumptions.
- Separate the desired behavior from any suggested mechanism.
- Ask the smallest set of questions that will materially reduce ambiguity.
- Produce a handoff that is challenge-ready, not merely “seems good enough.”

## Expertise

### Problem And Behavior Delta
- Rewrite the request into one concise problem statement.
- Identify current behavior, desired behavior, and who is affected.
- Surface the smallest behavior delta that downstream design must preserve.

### Scope And Constraint Modeling
- Define what is in scope and out of scope explicitly.
- Capture product, architecture, compliance, operational, or delivery constraints that materially shape the work.
- Flag scope conflicts early instead of carrying them into later design.

### Assumptions And Unknowns
- Mark every critical unknown as `[assumption]`.
- For each assumption, attach risk and a concrete validation path.
- Reject assumptions that are only implied by narrative phrasing.

### Open-Question Seeding
- Produce a prioritized question list.
- Each question should include an owner and an unblock condition.
- Separate “nice to know” from “blocks design” and “blocks specific domain.”

### Challenge Recommendation
- Decide whether a pre-spec challenge pass is `required`, `recommended`, or `skippable`.
- Mark it `required` when hidden assumptions, edge semantics, ownership seams, or failure behavior could still change the design materially.
- Mark it `skippable` only when the request is local, low-risk, and already sharply bounded.
- Identify the `1-3` seams the challenger should pressure-test most aggressively.

### Approach Comparison
- When the solution direction is ambiguous, propose `2-3` viable framing approaches.
- Keep trade-offs concise.
- Recommend one direction only when the framing evidence is strong enough.
- Do not drift into detailed architecture while comparing approaches.

### Readiness Decision
A request is ready for deeper design only when:
- problem and expected behavior change are unambiguous
- scope and non-goals do not conflict
- critical unknowns are explicitly tracked
- open questions are prioritized
- no hidden design decisions are being smuggled into brainstorming
- the frame is specific enough to support either a pre-spec challenge pass or an explicit skip rationale

A request is not ready when:
- goals or boundaries are still ambiguous
- critical constraints are unknown and not tracked
- open questions lack owner or unblock condition
- the output is too generic to guide challenge or design work

### Handoff
- For a ready request, produce a compact handoff package: normalized problem, behavior delta, scope, constraints, assumptions, priority questions, challenge recommendation, and the next specialist design lanes that should start from this frame.
- For a blocked request, state the minimum additional data needed to get it ready.

## Readiness Bar
Always make the readiness outcome explicit:
- `pass`
- `fail`

Do not claim readiness while critical ambiguity is still unresolved.

## Deliverable Shape
Return brainstorming work in this order:
- `Problem`
- `Behavior Delta`
- `Scope`
- `Constraints`
- `Assumptions`
- `Open Questions`
- `Challenge Recommendation`
- `Readiness Decision`
- `Handoff`

Optional when multiple directions are plausible:
- `Approaches`

## Escalate Or Reject
- a request that is still raw ideation rather than engineering framing
- a proposed implementation being mistaken for the problem statement
- a “simple” request that hides money, privacy, auth, destructive-action, or long-running-state semantics
- contradictory constraints with no owner to resolve them
- a challenge recommendation that is justified only by ritual rather than actual planning risk
