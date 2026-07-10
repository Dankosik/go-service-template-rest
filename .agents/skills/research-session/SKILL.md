---
name: research-session
description: "Own a session dedicated only to research for this repository when research needs a durable checkpoint. Use when the orchestrator already has an accepted Phase 0 task brief plus workflow routing and needs one bounded session to resolve evidence questions before specification."
---

# Research Session

## Eligibility And Outcome

Use when current routing enters research, research_expectation is expected or conditional-and-triggered, and evidence gaps must close before specification. Skip for SHAPE-DIRECT, research_expectation=not_expected, unresolved intake, specification authoring, design, planning, or implementation.

The outcome is a bounded evidence set with source limits, conflicts, assumptions, and explicit specification destinations. Research supports decisions; it does not make the final spec decision.

## Canonical Owners

- [Research phase](../../../docs/spec-first-workflow/phases/research.md) owns entry inputs, evidence quality, research outputs, fan-in, and the research stop rule.
- [Artifact model](../../../docs/spec-first-workflow/shared/artifact-model.md) owns research expectation, phase-control eligibility, typed state, and routing validity.
- [Subagents and handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) owns lane gates, authorization wording, resume order, and final prompt rendering.
- [Specification phase](../../../docs/spec-first-workflow/phases/specification.md) owns the decisions synthesized from research.

Additional reads are lazy: use the question-framing reference only for an unclear evidence question, lane-planning only for real fan-out, anti-patterns only when research is expanding without decision value, and fan-in examples only when the handoff is too dense to express compactly.

## Allowed Side Effects

This session may write preserved task-local `research/*.md`, including `research/pattern-fit.md`, when the canonical preservation test passes. It may update existing `workflow-plan.md` and create or update `workflow-plans/research.md` only when ROUTING-PHASE-CONTROL allows it.

It must not write or repair `spec.md`, design, `test-plan.md`, `tasks.md`, code, tests, generated output, or later-phase control files.

## Unique Method

1. Convert each uncertainty into an evidence question naming the decision it can change, authoritative source or lane, freshness requirement, minimum useful proof, and specification destination.
2. Choose the smallest evidence plan. Use read-only lanes only for concrete independent bounded questions when separate context materially improves speed or quality. Keep small, sequential, single-chain, or shared-state work local. Default to no more than three concurrently active subagent lanes; exceeding three requires a task-specific reason.
3. Compare stdlib, repository patterns, mature OSS, or established system patterns when dependency/OSS or Pattern Fit Diligence is triggered.
4. Preserve only evidence that aids synthesis, auditability, or resume. Record absence as a source limit unless the searched source is authoritative for absence.
5. Reconcile conflicts and classify unresolved points as blocks_spec, proof_only, accepted_risk, or needs_specialist.

Missing explicit subagent authorization is not a valid local research rationale. Record the blocked gate and obtain the exact authorization line from the canonical handoff owner.

## Success, Blocked Stop, And Reopen

Success means every required evidence question is answered or honestly bounded, important sources and dates are recorded, conflicts and weak evidence are visible, preserved notes have a reason to exist, and the specification handoff names decisions, constraints, assumptions, risks, proof obligations, blockers, and files to consume.

Stop blocked when a user decision, authoritative source, current external proof, required lane, or routing anchor is missing. Do not disguise the gap as a confident conclusion.

Reopen intake for unresolved user intent, workflow planning for invalid routing/control, research for targeted evidence gaps, or specification when evidence is sufficient. Stop before writing the spec. Render the next-session or reopen prompt only through [Subagents and handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md).
