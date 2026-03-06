---
name: spec-first-brainstorming
description: "Turn raw feature or behavior-change requests into a clear problem frame with scope, constraints, assumptions, prioritized questions, and an explicit design-readiness decision."
---

# Design Brainstorming

## Purpose
Turn raw requests into a clear, bounded, reviewable problem frame before deeper design starts.

## Scope
- normalize feature, refactor, or behavior-change requests into a precise problem statement
- define scope, non-goals, constraints, affected actors, and success criteria
- capture explicit assumptions and their validation path
- seed prioritized open questions with owner and unblock condition
- decide whether the request is ready for deeper design work

## Boundaries
Do not:
- make architecture, API, data, security, or reliability decisions that belong to downstream specialists
- jump into implementation design, code, or test-writing
- hide ambiguity behind generic wording or unexamined assumptions
- confuse the requested outcome with the user’s proposed implementation idea

## Core Defaults
- Be strict about scope and unknowns.
- Keep statements concrete and testable.
- Prefer explicit blockers over hidden assumptions.
- Separate the desired outcome from any proposed solution direction.
- When clarification is needed, ask narrow questions that reduce ambiguity quickly.

## Expertise

### Problem Framing
- Rewrite the request into one concise problem statement.
- Separate the requested outcome from the solution the requester happens to suggest.
- Identify the expected behavior change and who is affected by it.

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
- Separate “nice to know” from “blocks design.”

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

A request is not ready when:
- goals or boundaries are still ambiguous
- critical constraints are unknown and not tracked
- open questions lack owner or unblock condition
- the output is too generic to guide design work

### Handoff
- For a ready request, produce a compact handoff package: normalized problem, scope, constraints, assumptions, and priority questions.
- For a blocked request, state the minimum additional data needed to get it ready.

## Readiness Bar
Always make the readiness outcome explicit:
- `pass`
- `fail`

Do not claim readiness while critical ambiguity is still unresolved.

## Deliverable Shape
Return brainstorming work in this order:
- `Problem`
- `Scope`
- `Constraints`
- `Assumptions`
- `Open Questions`
- `Readiness Decision`
- `Handoff`

Optional when multiple directions are plausible:
- `Approaches`

## Escalate When
Escalate if:
- goals, actors, or behavior change remain ambiguous after focused clarification
- scope conflicts cannot be resolved cleanly
- critical constraints are missing but affect design direction materially
- the discussion is drifting into downstream design decisions that this skill should not own
