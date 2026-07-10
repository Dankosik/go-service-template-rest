# Spec-First Workflow

Runtime entrypoint and router to [AGENTS.md](../AGENTS.md) for subagent-first spec-first work. This file stays the stable path that agents, skills, and humans open first; detailed mechanics live in the linked phase and shared docs.

## 1. Authority And Purpose

`AGENTS.md` is authoritative for roles, invariants, execution-shape triggers, and hard boundaries. When the two documents diverge, follow `AGENTS.md` first and repair this companion. This document explains how to apply those rules in task-local artifacts without forcing the full bundle onto every bounded change.

The workflow keeps the same quality concerns:

- confirmed understanding of the user's raw intent before routing;
- clear framing and scope cuts;
- final decision ownership by the orchestrator;
- read-only advisory subagents as the normal evidence surface for non-trivial decisions;
- phase-owned review, repair, and fresh re-review without user-managed internal handoffs;
- canonical `spec.md` when a task-local decision artifact exists;
- executable task handoff when implementation is non-trivial;
- fresh validation evidence before completion claims.

The simplification is about artifact depth, not decision quality or specialist coverage. Use the smallest workflow shape that preserves correctness.

Subagent-first means non-trivial decisions are normally synthesized from narrow read-only lane summaries. Artifact depth may stay lean, but the decision basis should not be single-threaded when independent evidence, challenge, or specialist review can improve correctness.

Hard decision-quality rules live in [AGENTS.md](../AGENTS.md). [Artifact Model](spec-first-workflow/shared/artifact-model.md) owns where those rules are recorded in task-local artifacts. The router keeps only the loading path; phase and shared files own detailed mechanics.

## 2. Runtime Loading Model

Use this file as the stable workflow entrypoint. Do not load every detailed workflow file by default.

1. Read `AGENTS.md` first for the compact authority and hard invariants.
2. If the raw request is not yet an accepted task brief, read [Intake / Phase 0](spec-first-workflow/phases/intake.md) before choosing execution shape.
3. Use the execution shape summary and phase read matrix below to identify the current phase.
4. Choose shape only from the ordered `SHAPE-*` and `FULL-*` rules in `AGENTS.md`; then read `spec-first-workflow/shared/artifact-model.md` for artifact depth, typed state, reclassification, task-local ownership, status, or layout rules.
5. Read the file for the current user-started macro phase and the internal checkpoint owners it names, plus `spec-first-workflow/shared/subagents-and-handoff.md` when subagent gates, review/repair loops, model routing, resume order, or next-session prompts matter.
6. If a phase exposes a missing decision, proof path, or required artifact, reopen the owning earlier phase instead of solving it in a later phase.

### Skill Wrapper Boundary

Repository-local workflow session skills are thin phase adapters, not workflow authorities. A wrapper keeps only its exact eligibility, phase-specific outcome, allowed writes or side effects, unique method, and success/blocked/reopen rules. It links to this router, the active phase owner, the artifact model when typed state matters, and the shared subagent/handoff owner when lanes or prompts matter.

Session skills must not copy the global read order, typed-state tables, authorization line, final-handoff template, phase procedure, completion matrix, or generic operating rules. Load a skill reference only when that reference can change the active decision. When a wrapper and its linked canonical owner diverge, follow the canonical owner and repair the wrapper.

## 3. Execution Shape Summary

| Shape | Selected by | Detailed mechanics |
| --- | --- | --- |
| `direct_path` | `SHAPE-DIRECT` after every `DIRECT-*` predicate is proven. | [Artifact Model](spec-first-workflow/shared/artifact-model.md) |
| `lean_local` | `SHAPE-LEAN` after every `LEAN-*` predicate is proven and no `FULL-*` floor applies. | [Artifact Model](spec-first-workflow/shared/artifact-model.md), [Specification](spec-first-workflow/phases/specification.md), [Specification Review](spec-first-workflow/phases/specification-review.md), triggered [System / Integration Design](spec-first-workflow/phases/system-integration-design.md), triggered [Go Code / Ownership Design](spec-first-workflow/phases/go-code-ownership-design.md), triggered [Test Design](spec-first-workflow/phases/test-design.md), [Planning](spec-first-workflow/phases/planning.md), [Task Review / Readiness](spec-first-workflow/phases/task-review-readiness.md) |
| `full_orchestrated` | `SHAPE-FULL-FLOOR` or `SHAPE-FALLBACK-FULL`; capability-only agent authorization never selects it. | [Artifact Model](spec-first-workflow/shared/artifact-model.md) plus the active phase file below |

`AGENTS.md` owns the complete classifier, agent-request meaning, direct writer, adequacy predicate, escalation triggers, and hard boundaries. [Artifact Model](spec-first-workflow/shared/artifact-model.md) owns artifact-depth implications, typed state, reclassification/status mechanics, lean `spec.md`, lean `tasks.md`, and workflow-control artifact rules.

Phase 0 intake precedes shape selection. It usually stays in chat as an accepted task brief; preserve it in workflow-control only when multi-session resume or later routing needs durable state.

## 4. Phase Read Matrix

| Current Need | Read |
| --- | --- |
| Understand a raw, dictated, vague, or mixed request before choosing workflow shape or writing artifacts. | [Intake / Phase 0](spec-first-workflow/phases/intake.md) |
| Choose execution shape, then record artifact depth, task-local layout, or workflow-control ownership. | `AGENTS.md` `SHAPE-*` rules first, then [Artifact Model](spec-first-workflow/shared/artifact-model.md) |
| Research, evidence fan-out, dependency/OSS diligence, or Pattern Fit research. | [Research](spec-first-workflow/phases/research.md) |
| Run the user-started specification phase: write or repair `spec.md`, reconcile clarification, run mandatory independent specification review, repair findings, and obtain a fresh verdict. | [Specification](spec-first-workflow/phases/specification.md), then its internal [Specification Review](spec-first-workflow/phases/specification-review.md) checkpoint |
| Execute the read-only specification-review method inside an active specification session, or satisfy an explicitly read-only review request. | [Specification Review](spec-first-workflow/phases/specification-review.md) |
| Run the user-started technical-design phase: decide system/integration behavior, then Go package/file ownership, then review, repair, and re-review the combined packet. | [System / Integration Design](spec-first-workflow/phases/system-integration-design.md), [Go Code / Ownership Design](spec-first-workflow/phases/go-code-ownership-design.md), and internal [Technical Design Review](spec-first-workflow/phases/technical-design-review.md) |
| Execute the read-only technical-design-review method inside an active technical-design session, or satisfy an explicitly read-only review request. | [Technical Design Review](spec-first-workflow/phases/technical-design-review.md) |
| Run the user-started test-design phase: design scenarios and proof levels, obtain independent QA review, repair findings, and close with a current verdict. | [Test Design](spec-first-workflow/phases/test-design.md) |
| Run the user-started planning phase: draft or repair `tasks.md`, run task-review/readiness, repair planning-local findings, and obtain implementation readiness. | [Planning](spec-first-workflow/phases/planning.md) and internal [Task Review / Readiness](spec-first-workflow/phases/task-review-readiness.md) checkpoint |
| Execute the read-only task-review/readiness method inside an active planning session, or satisfy an explicitly read-only review request. | [Task Review / Readiness](spec-first-workflow/phases/task-review-readiness.md) |
| Run the user-started implementation phase from an approved ledger through worker patching, post-code review, repair, fresh re-review, validation, and closeout. | [Implementation / Validation / Closeout](spec-first-workflow/phases/implementation-validation-closeout.md) |
| Plan subagent lanes, audit subagent gates, resume from artifacts, or render final handoff prompts. | [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md) |

## 5. Global Resume And Handoff Rules

- Resume from current artifacts, not chat memory; approved `tasks.md` is the first execution-state read when implementation or closeout is next.
- [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md) is the canonical owner for lane planning, subagent gates, task-aware model routing, phase-internal review/repair loops, resume order, next-session prompt rendering, and implementation Goal handoff shape.
- [Implementation / Validation / Closeout](spec-first-workflow/phases/implementation-validation-closeout.md) owns isolated-worker execution, patch intake, validation, and closeout mechanics.

## 6. Router Guardrails

- `AGENTS.md` remains the compact authority. If this router or any linked detail file diverges from `AGENTS.md`, follow `AGENTS.md` and repair the drift.
- Keep `docs/spec-first-workflow.md` as the stable entrypoint path. Do not point skills or agents directly at every phase file unless the current phase requires that detail.
- Do not duplicate authority across the router and phase files. Keep summaries here, and keep expanded mechanics in exactly one owning phase or shared file.
- Do not create task-local workflow artifacts for ceremony. Use `AGENTS.md` for shape/trigger rules and [Artifact Model](spec-first-workflow/shared/artifact-model.md) only for artifact/state consequences.
- Render a next-session prompt only at a macro-phase boundary. Internal review, repair, and fresh re-review return to the owning root session and never require the user to start another task.
- Do not skip Phase 0 intake when the request is rough enough that routing would depend on interpretation.
