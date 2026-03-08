---
name: spec-first-brainstorming
description: "Turn raw feature, refactor, or behavior-change requests into a design-ready problem frame before deeper spec work. Use whenever the user brings an ambiguous feature idea, bug or behavior change, scope discussion, or proposed implementation that needs to be normalized into clear outcome, actors, constraints, hidden invariants, prioritized unblock questions, and an explicit readiness decision before architecture, API, data, security, or reliability design starts."
---

# Spec-First Brainstorming

## Purpose
Turn ambiguous requests into a concrete, falsifiable framing artifact that downstream spec and review skills can safely build on without rediscovering the core problem.

## Scope
- normalize raw feature, refactor, or behavior-change requests into a precise problem frame
- identify current behavior, desired behavior, affected actors, and the material behavior delta
- define scope, non-goals, constraints, hidden invariants, acceptance semantics, and blocking unknowns
- capture explicit assumptions, their risk, and how they should be validated
- decide whether the request is ready for deeper design work and which specialist tracks it should route to next

## Boundaries
Do not:
- make final architecture, API, schema, security-control, or reliability decisions that belong to downstream specialists
- let a user-proposed implementation shortcut problem framing
- treat policy, domain semantics, or operational expectations as obvious when they are not explicit
- hide ambiguity behind generic wording such as “improve UX,” “support undo,” or “make it scalable”
- mark a request ready when behavior, ownership, or acceptance semantics are still materially ambiguous

## Escalate When
Escalate if:
- a core business term is overloaded or undefined
- the desired outcome conflicts with an existing policy, invariant, or compliance obligation
- the request mixes multiple services or ownership domains but the source of truth is unclear
- the request sounds local but actually changes money, identity, destructive actions, privacy, or irreversible state
- a proposed solution is already smuggling in architecture or data decisions before the problem is stable

## Core Defaults
- Prefer outcome over proposed solution.
- Treat every behavior change as `actor -> trigger -> object -> behavior delta -> success/failure semantics`.
- Distinguish fixed constraints from assumed or negotiable ones.
- Surface hidden invariants and irreversible effects before deeper design starts.
- Ask the smallest set of questions that would materially change scope, correctness, ownership, or routing.
- Scale depth to task risk: compress the output for simple low-risk changes, but never skip problem, scope, blockers, or readiness.
- Preserve downstream freedom: make the problem sharper, not narrower.

## Expertise

### Outcome Normalization And Behavior Delta
- Rewrite the request into one concise problem statement in user terms, not implementation terms.
- Capture:
  - current behavior
  - desired behavior
  - affected actor or actors
  - trigger or entry point
  - object, resource, or state being changed
- Separate the user’s goal from the mechanism they suggested.
- Identify why the change matters now when that materially shapes prioritization or constraints.

### Domain Language And Ambiguity Control
- Normalize overloaded verbs such as `cancel`, `delete`, `undo`, `restore`, `export`, `sync`, or `archive`.
- Convert vague adjectives such as `fast`, `simple`, `safe`, `transparent`, or `real-time` into concrete questions or measurable expectations.
- Treat temporal words carefully: `now`, `immediately`, `within 7 days`, `eventually`, `period end`, and similar phrasing usually hide policy decisions.
- Split one term into multiple meanings when it could refer to product semantics, UX semantics, or backend semantics.

### Scope And Non-goal Shaping
- Define the primary path the request is trying to change.
- Make non-goals explicit so downstream design does not accidentally absorb adjacent work.
- Mark adjacent systems, actors, or data classes that are touched, excluded, or suspiciously absent.
- Flag scope contradictions early, especially when the stated boundary cannot actually satisfy the requested outcome.
- Call out when “service-only,” “no UI change,” or “no data migration” constraints leave a material gap.
- When semantics may differ by plan, provider, lifecycle state, or customer cohort, require an explicit launch-eligibility boundary instead of assuming one default path.

### Constraint Modeling
- Classify constraints rather than listing them generically. Useful buckets:
  - product or customer promise
  - domain or policy rule
  - legal or compliance rule
  - operational or SLA expectation
  - compatibility, rollout, or migration limit
  - organizational, ownership, or deadline constraint
- Mark each material constraint as:
  - `fixed`
  - `assumed`
  - `negotiable`
- Reject constraint filler that does not actually change the design space.

### Hidden Invariants And Acceptance Semantics
- Surface correctness rules that are implied by the request even before full design begins.
- Look for hidden invariants such as:
  - authorization boundaries
  - tenant isolation
  - money conservation
  - uniqueness or deduplication
  - reversibility limits
  - retention obligations
  - auditability or traceability
- Define what success means from an external observer’s point of view.
- Define what denial, partial completion, timeout, cancellation, or duplicate request means when those outcomes matter.
- If support load, operator trust, or customer clarity is part of the stated goal, define the visible pending, failed, and manual-follow-up states instead of treating them as secondary details.
- Flag idempotency, replay, ordering, or late-event risk when repeated execution could change the meaning of success.
- Do not design the full state machine here, but do expose missing semantics that will change downstream design.

### Assumptions And Evidence Quality
- Mark every critical unknown as `[assumption]`.
- For each assumption, include:
  - why it matters
  - what risk it introduces
  - how it should be validated
  - who is most likely to confirm or reject it
- If an assumption changes customer-visible semantics, policy interpretation, or launch eligibility, upgrade it into a blocking question instead of leaving it as a passive note.
- Do not silently promote narrative hints into facts.
- Reduce readiness when the frame depends on weak evidence, even if the document looks tidy.

### Question Design And Prioritization
- Ask only questions whose answers would change scope, correctness, ownership, or readiness.
- Prefer blockers that collapse ambiguity fastest: customer promise, eligibility boundary, exception policy, source-of-truth ownership, and failure-state semantics usually matter more than implementation mechanics.
- Classify questions as:
  - `blocks_design`
  - `blocks_specific_domain`
  - `nice_to_know`
- Every blocking question should include:
  - owner
  - why it matters now
  - unblock condition
- Prefer fewer discriminating questions over a long checklist.

### Alternative Framing Paths
- When the same request could legitimately mean different problems, present `2-3` framing options.
- Compare options by:
  - outcome
  - scope
  - risk
  - likely downstream specialist impact
- Recommend a default only when the framing evidence is strong enough.
- Do not drift into detailed architecture while comparing options.

### Cross-Domain Impact And Specialist Routing
- Identify which downstream domains clearly need follow-up once the frame is stable:
  - API or contract
  - domain behavior or invariants
  - data or persistence
  - security or privacy
  - reliability or async semantics
  - observability
  - delivery or rollout
- Keep this as routing guidance, not as detailed design.
- Note reopen conditions when one unresolved point could materially change specialist routing.

### Readiness And Handoff Quality
A request is ready for deeper design only when:
- problem, actor, and behavior delta are unambiguous
- scope and non-goals do not contradict the desired outcome
- material constraints are explicit
- hidden invariants or acceptance semantics that could change design are visible
- critical assumptions and blocking questions are explicit
- a downstream specialist can continue without rediscovering the actual problem

A request is not ready when:
- key terms remain overloaded or undefined
- correctness depends on an unspoken policy or invariant
- scope excludes systems or actors required to satisfy the stated outcome
- blocking questions lack owner or unblock condition
- the output is generic enough that multiple contradictory designs could all claim to fit it

## Decision Quality Bar
For non-trivial requests, include:
- normalized problem statement
- current vs desired behavior delta
- affected actors, systems, and boundaries
- launch cohort, eligibility, or exception boundary when semantics may vary by case
- material constraints with classification
- hidden invariants or acceptance semantics
- explicit non-goals and scope gaps
- assumptions with risk and validation path
- prioritized questions with owner and unblock condition
- readiness decision with blockers, confidence, and recommended next specialist track or tracks

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
- `Hidden Invariants / Acceptance Semantics`
- `Assumptions`
- `Open Questions`
- `Impact / Specialist Routing`
- `Readiness Decision`
- `Handoff`

Optional when multiple interpretations are genuinely plausible:
- `Approaches`

## Escalate Or Reject
- a proposed implementation being mistaken for the problem statement
- a “simple” request that hides money, privacy, auth, destructive-action, or long-running-state semantics
- contradictory policy or compliance constraints with no owner to resolve them
- a scope boundary that makes success impossible but is being treated as fixed without acknowledgment
- a readiness call based on confidence language instead of explicit evidence and blockers
