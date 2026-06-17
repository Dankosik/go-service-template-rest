# Spec-First Workflow

Runtime entrypoint and router to [AGENTS.md](../AGENTS.md) for subagent-first spec-first work. This file stays the stable path that agents, skills, and humans open first; detailed mechanics live in the linked phase and shared docs.

## 1. Authority And Purpose

`AGENTS.md` is authoritative for roles, invariants, execution-shape triggers, and hard boundaries. When the two documents diverge, follow `AGENTS.md` first and repair this companion. This document explains how to apply those rules in task-local artifacts without forcing the full bundle onto every bounded change.

The workflow keeps the same quality concerns:

- clear framing and scope cuts;
- final decision ownership by the orchestrator;
- read-only advisory subagents as the normal evidence surface for non-trivial decisions;
- canonical `spec.md` when a task-local decision artifact exists;
- executable task handoff when implementation is non-trivial;
- fresh validation evidence before completion claims.

The simplification is about artifact depth, not decision quality or specialist coverage. Use the smallest workflow shape that preserves correctness.

Subagent-first means non-trivial decisions are normally synthesized from narrow read-only lane summaries. Artifact depth may stay lean, but the decision basis should not be single-threaded when independent evidence, challenge, or specialist review can improve correctness.

Hard decision-quality rules live in [AGENTS.md](../AGENTS.md). [Artifact Model](spec-first-workflow/shared/artifact-model.md) owns where those rules are recorded in task-local artifacts. The router keeps only the loading path; phase and shared files own detailed mechanics.

## 2. Runtime Loading Model

Use this file as the stable workflow entrypoint. Do not load every detailed workflow file by default.

1. Read `AGENTS.md` first for the compact authority and hard invariants.
2. Use the execution shape summary and phase read matrix below to identify the current phase.
3. Read `spec-first-workflow/shared/artifact-model.md` when choosing shape, artifact depth, task-local artifact ownership, status vocabulary, or layout rules.
4. Read exactly the phase file that owns the current work, plus `spec-first-workflow/shared/subagents-and-handoff.md` when subagent gates, resume order, or next-session prompts matter.
5. If a phase exposes a missing decision, proof path, or required artifact, reopen the owning earlier phase instead of solving it in a later phase.

## 3. Execution Shape Summary

| Shape | Use When | Detailed Mechanics |
| --- | --- | --- |
| `direct path` | Tiny, reversible, one surface, obvious validation, no protected-domain trigger. | [Artifact Model](spec-first-workflow/shared/artifact-model.md) |
| `lean local` | Bounded non-trivial single-domain work with stable ownership and limited research. | [Artifact Model](spec-first-workflow/shared/artifact-model.md), [Specification](spec-first-workflow/phases/specification.md), [Specification Review](spec-first-workflow/phases/specification-review.md), triggered [System / Integration Design](spec-first-workflow/phases/system-integration-design.md), triggered [Go Code / Ownership Design](spec-first-workflow/phases/go-code-ownership-design.md), triggered [Test Design](spec-first-workflow/phases/test-design.md), [Planning](spec-first-workflow/phases/planning.md), [Task Review / Readiness](spec-first-workflow/phases/task-review-readiness.md) |
| `full orchestrated` | Cross-domain, ambiguous, hard-to-reverse, high-impact, long-running, user-requested agent-backed, or protected-domain work. | [Artifact Model](spec-first-workflow/shared/artifact-model.md) plus the active phase file below |

`AGENTS.md` owns escalation triggers and hard boundaries. [Artifact Model](spec-first-workflow/shared/artifact-model.md) owns artifact-depth implications, status vocabulary, lean `spec.md`, lean `tasks.md`, and workflow-control artifact rules.

## 4. Phase Read Matrix

| Current Need | Read |
| --- | --- |
| Choose execution shape, artifact depth, task-local layout, or workflow-control ownership. | [Artifact Model](spec-first-workflow/shared/artifact-model.md) |
| Research, evidence fan-out, dependency/OSS diligence, or Pattern Fit research. | [Research](spec-first-workflow/phases/research.md) |
| Write or repair `spec.md`, reconcile clarification challenge, or run lean `Risk Challenge`. | [Specification](spec-first-workflow/phases/specification.md) |
| Review a completed non-trivial `spec.md` before design, test design, planning, or implementation. | [Specification Review](spec-first-workflow/phases/specification-review.md) |
| Decide service behavior as a system participant: REST/API, triggered contract design, external calls, queues, database/cache/source-of-truth, sequence, failure behavior, validation, or rollout. | [System / Integration Design](spec-first-workflow/phases/system-integration-design.md) |
| Decide Go package/file ownership, focused responsibilities, dependency direction, local abstractions, cleanup/removal, and test ownership. | [Go Code / Ownership Design](spec-first-workflow/phases/go-code-ownership-design.md) |
| Review triggered system/integration and Go code ownership design before planning. | [Technical Design Review](spec-first-workflow/phases/technical-design-review.md) |
| Design test scenarios, proof levels, pass/fail observables, fail-before expectations, and quality gates before task breakdown. | [Test Design](spec-first-workflow/phases/test-design.md) |
| Draft or repair `tasks.md` from reviewed specification and required design context. | [Planning](spec-first-workflow/phases/planning.md) |
| Review completed `tasks.md`, run task-ledger review, or approve implementation readiness. | [Task Review / Readiness](spec-first-workflow/phases/task-review-readiness.md) |
| Implement from an approved ledger, run post-code review/reconciliation, validate, or close out. | [Implementation / Validation / Closeout](spec-first-workflow/phases/implementation-validation-closeout.md) |
| Plan subagent lanes, audit subagent gates, resume from artifacts, or render final handoff prompts. | [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md) |

## 5. Global Resume And Handoff Rules

- Resume from artifacts, not chat memory.
- If approved `tasks.md` exists and implementation, review, validation, or closeout is next, read `tasks.md` first and treat it as the execution authority.
- If no approved `tasks.md` exists and `workflow-plan.md` exists, read `workflow-plan.md`, then the current `workflow-plans/<phase>.md` when used, then the files named in the next-session context bundle.
- If no approved `tasks.md` and no `workflow-plan.md` exist because the task is direct or lean, read `spec.md` when present, then the specification-review record when moving beyond specification, then `test-plan.md` when test design is triggered, then `tasks.md` when implementation or validation is next.
- At every non-implementation phase boundary with a next session or reopen target, the final chat response must include a copy-pastable recommended next-session prompt derived from recorded workflow state.
- Implementation from approved, reviewed `tasks.md` is the normal exception: the next-session prompt must use `codex-goal-prompt-composer`, tell the next agent to set a Codex Goal first, and then execute the approved ledger through its named proof.

Detailed subagent, resume, and handoff mechanics live in [Subagents And Handoff](spec-first-workflow/shared/subagents-and-handoff.md).

## 6. Router Guardrails

- `AGENTS.md` remains the compact authority. If this router or any linked detail file diverges from `AGENTS.md`, follow `AGENTS.md` and repair the drift.
- Keep `docs/spec-first-workflow.md` as the stable entrypoint path. Do not point skills or agents directly at every phase file unless the current phase requires that detail.
- Do not duplicate authority across the router and phase files. Keep summaries here, and keep expanded mechanics in exactly one owning phase or shared file.
- Do not create task-local workflow artifacts for ceremony. Use the shape and trigger rules in [Artifact Model](spec-first-workflow/shared/artifact-model.md).

## 7. Anti-Patterns

Avoid:

- treating full orchestrated as the default for every non-trivial task;
- using direct path for risky, ambiguous, public, data, security, money, reliability, concurrency, or rollout work;
- using lean local without `spec.md`, `tasks.md`, inline `Risk Challenge`, and proof;
- using lean local as a local-only decision path without a recorded subagent gate decision;
- making `workflow-plans/<phase>.md` a second master plan, spec, design bundle, or task ledger;
- planning non-trivial implementation from `spec.md` alone when system/integration or Go code ownership design context is triggered;
- letting planning invent test scenarios, proof levels, or fail-before expectations when `test-design` is triggered;
- starting implementation from an unreviewed `tasks.md` or treating a draft ledger as approval;
- approving non-trivial specs while formal challenge is required and unresolved;
- splitting work into MVP plus future hardening when the production-ready decision is knowable and in scope;
- picking the fastest or simplest architecture by default instead of the best production-ready choice for the accepted scope;
- inventing a custom architecture or system-design shape without Pattern Fit Diligence, or applying a named pattern without task-specific evidence and Go/repository fit;
- importing class-oriented design-pattern scaffolding into Go or adding pattern-shaped helpers when direct stdlib/repo-native code is shorter and clearer;
- growing large hand-written source files as an implementation shortcut instead of placing new code in the focused owner file, same-package seam file, or correct package boundary;
- treating system/integration design as enough for planning when package/file responsibility, dependency direction, cleanup/removal, or test ownership still need design;
- treating REST/OpenAPI, event payload, generated-contract, or material internal-interface shape as implementation detail instead of a triggered system/integration contract-design checkpoint;
- creating `test-plan.md`, `rollout.md`, split design files, or review/validation phase files for completeness;
- treating a generic "add tests" task as equivalent to approved test design for behavior-risky work;
- creating new process artifacts after coding starts;
- using subagents for broad ceremony rather than narrow unresolved questions;
- claiming done without fresh, scope-matched evidence.
