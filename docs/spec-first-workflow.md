# Spec-First Workflow (Orchestrator/Subagent-First)

## 1. Purpose

This document defines the repository-level spec-first workflow.
It keeps one universal loop for small and large changes while moving execution away from skill-centric choreography and toward orchestrator/subagent-first execution.

The workflow is intentionally simple:
- no mandatory phase matrix,
- no mandatory user intake schema,
- no linear skill chain in the main flow,
- no artifact fan-out unless it removes real risk.

## 2. Core Position

1. The main flow is owned by the orchestrator.
2. The orchestrator keeps the full task context and makes the final decisions.
3. Subagents are the default mechanism for non-trivial research and review tracks.
4. Subagents are always `research-only/read-only`.
5. Skills are on-demand tools, usually used inside subagents rather than as the main workflow.
6. In the main flow, planning skills may be used only during implementation planning, and implementation skills only during implementation.
7. `spec.md` is the canonical decisions artifact.
8. `research/*.md` stores validated research context and does not replace `spec.md`.
9. Coding starts only after an explicit implementation plan exists.
10. Any single subagent conclusion is advisory until the orchestrator synthesizes it and, when needed, rechecks it.
11. For medium/high-risk or ambiguous work, candidate synthesis is pressure-tested before planning or explicitly waived with rationale.

For very small and low-risk tasks, the orchestrator may keep research local and skip subagent fan-out.
The invariants above still apply.

## 3. Artifact Model

Default layout:

```text
specs/<feature-id>/
  spec.md
```

Optional only when needed:

```text
specs/<feature-id>/
  research/
    <topic>.md
  plan.md
  test-plan.md
```

Artifact rules:
1. Keep `spec.md` as the single source of truth for final decisions.
2. Use `research/*.md` only when the task is long, ambiguous, or likely to benefit from reusable validated context.
3. Keep `research/*.md` flexible and task-driven; there is no mandatory universal template.
4. Use `plan.md` only when the implementation plan becomes too large or parallelized to stay readable inside `spec.md`.
5. Use `test-plan.md` only when test obligations are large enough to hide the core plan.
6. Do not duplicate decision text across files; link instead.
7. Keep raw pre-spec challenge transcripts out of `spec.md`; record only the resolved decisions and remaining open questions.

## 4. Default `spec.md` Shape

Use a lightweight structure inside `spec.md`.
The default sections are:

1. `Context`
2. `Scope / Non-goals`
3. `Constraints`
4. `Decisions`
5. `Open Questions / Assumptions`
6. `Implementation Plan`
7. `Validation`
8. `Outcome`

Rules:
1. Keep the sections that help the reader and merge them when that is clearer.
2. Do not create empty sections or filler text for completeness.
3. Record final decisions in `Decisions`; do not dump raw research there.
4. Link to `research/*.md` when evidence history matters.
5. If the implementation plan or test strategy becomes too large, move that material to `plan.md` or `test-plan.md` and keep only the control summary in `spec.md`.

For non-trivial tasks, keep a compact audit trail in `spec.md` and linked artifacts:
1. intake summary,
2. research questions or subagent tracks,
3. decision log,
4. material overrides or rejected paths,
5. open questions with owner and unblock condition,
6. implementation plan,
7. validation evidence,
8. outcome.

## 5. Universal Execution Loop

### Step 1. Intake And Framing

Owner: Orchestrator

The orchestrator clarifies:
1. what must change,
2. what is in scope and out of scope,
3. which constraints and risk hotspots already exist,
4. which success checks are relevant,
5. which facts are still missing.

Rules:
1. User input stays free-form; no mandatory YAML or JSON intake is required.
2. Missing facts become explicit assumptions or open questions, not invented structure.

### Step 2. Research Decomposition

Owner: Orchestrator

The orchestrator decides:
1. which questions can be solved directly in the main flow,
2. which questions need subagent research,
3. which tracks are independent and should run in parallel,
4. which high-impact or ambiguous topics need multi-angle research or a challenger pass.

Each subagent task should state:
1. the goal and scope,
2. what must be checked,
3. relevant constraints and risks,
4. the expected output shape,
5. where evidence is required.

### Step 3. Parallel Subagent Research

Owner: Subagents

Subagents:
1. analyze only their assigned domain,
2. may use relevant skills locally,
3. return concise synthesis-ready output,
4. separate facts, recommendations, risks, confidence, and open points when relevant,
5. never change repository files, code, or the implementation plan.

There is no single rigid response schema for every subagent task.
The orchestrator defines the output shape needed for the current question.

### Step 4. Candidate Synthesis, Pre-Spec Challenge, And Decision Logging

Owner: Orchestrator

The orchestrator:
1. compares subagent outputs,
2. separates terminology noise from real conflicts,
3. checks evidence quality, assumptions, and applicability,
4. produces candidate decisions and the remaining open assumptions,
5. for medium/high-risk or ambiguous work, runs a pre-spec challenge pass before treating candidate decisions as stable,
6. resolves conflicts against user priorities inside non-negotiable correctness and safety boundaries,
7. resolves each material challenge by answering with evidence, triggering targeted re-research, asking the user, explicitly deferring it, or explicitly accepting risk,
8. records final decisions and unresolved items in `spec.md`,
9. writes validated research memory to `research/*.md` when it is worth preserving,
10. triggers a targeted recheck or second opinion when confidence is too low or the impact is too high.

Decision discipline:
1. Final decisions always stay with the orchestrator.
2. `pre-spec challenge` is a checkpoint inside synthesis, not a separate authority phase.
3. Rejected alternatives, challenge rejections, and overrides should be recorded when they materially affect the path forward.
4. If uncertainty remains important, keep it as an open question with owner and unblock condition.

Pre-spec challenge expectations:
1. Pass only the minimum relevant slice of context: problem frame, candidate decisions, constraints, assumptions/open questions, and evidence links when needed.
2. Ask only discriminating questions whose answers could change scope, correctness, ownership, failure semantics, or rollout.
3. Prefer a few high-signal questions over checklist coverage.
4. Keep the output compact and resolution-oriented rather than turning it into a second design document.

### Step 5. Implementation Planning

Owner: Orchestrator

This step is mandatory before coding.

The orchestrator turns decisions into:
1. ordered execution steps,
2. dependencies and checkpoints when relevant,
3. minimal validation per meaningful slice,
4. rollback or mitigation notes when the change carries real operational risk.

Planning is context-driven, not template-driven.
Only include rollout, backward-compatibility, migration, or rollback detail when the task actually needs it.

Planning skills may be used only in this step.

### Step 6. Implementation

Owner: Orchestrator in the main flow

Implementation:
1. follows the approved decisions and plan,
2. keeps code and spec in sync continuously,
3. may use implementation skills only in this step,
4. escalates back to planning if coding reveals a real design gap.

### Step 7. Parallel Domain Review

Owner: Review subagents, orchestrated by the orchestrator

When change size or risk justifies it, run parallel review-only subagents for the relevant domains.
Typical examples include security, reliability, performance, code quality, or test quality, but the set is task-driven rather than fixed.

Review rules:
1. Review subagents stay `research-only/read-only`.
2. Findings are advisory until reconciled by the orchestrator.
3. The goal is risk reduction, not procedural coverage.

### Step 8. Reconciliation, Validation, And Closeout

Owner: Orchestrator

The orchestrator:
1. reconciles review findings,
2. avoids fixes that improve one domain by breaking another,
3. runs targeted re-review when needed,
4. executes the smallest command set that proves correctness,
5. updates `Outcome` and the remaining open questions to match reality.

Any readiness or completion claim must include fresh command evidence.

## 6. Daily Operating Loop

For day-to-day work, use this short loop:
1. Frame the task in the main flow.
2. Spawn only the subagent tracks that reduce real uncertainty or review risk.
3. Synthesize candidate decisions and run pre-spec challenge when the task risk or ambiguity justifies it.
4. Write the implementation plan before coding.
5. Implement in the main flow.
6. Run review, recheck, and validation only as far as the task risk requires.

## 7. When To Fan Out

Use subagent fan-out when at least one is true:
1. the task crosses multiple domains,
2. a decision is high-impact or hard to reverse,
3. confidence is low after a first pass,
4. you need an independent challenger or second opinion,
5. a review wave would reduce meaningful delivery risk,
6. the task is long enough to justify preserved research memory.

Do not fan out when the extra orchestration would add more overhead than clarity.

## 8. Scaling Rule

The workflow does not change by feature size.
Only these things scale:
1. the number of subagent tracks,
2. the amount of preserved research,
3. the detail of the implementation plan,
4. the depth of review and validation.

Small work stays small inside the same workflow.

## 9. Definition Of Ready / Done

Definition of Ready:
1. `spec.md` or the active spec note has clear scope, constraints, and current decisions.
2. Critical unknowns are either resolved, delegated for research, or tracked explicitly.
3. For medium/high-risk or ambiguous work, the pre-spec challenge checkpoint is reconciled or explicitly waived.
4. An implementation plan exists before coding starts.

Definition of Done:
1. Implementation matches the recorded decisions.
2. Validation was run with fresh evidence.
3. `Outcome` reflects what was actually shipped and what remains open.

## 10. Legacy Compatibility

This document and `AGENTS.md` are the current repository-level source of truth for spec-first execution.

If older skill or workflow documents still mention:
- phase or gate choreography,
- `freeze/reopen` language,
- linear skill-first execution,

treat those references as legacy guidance until the skill refactor is complete.
When a legacy reference conflicts with this document, prefer the orchestrator/subagent-first model defined here.

## 11. Anti-Patterns

1. Forcing structured user intake before understanding the task.
2. Running a long linear chain of skills in the main flow.
3. Copying raw subagent reasoning into `spec.md`.
4. Letting subagents write code or mutate repository files.
5. Starting implementation before the planning step is explicit.
6. Filling optional sections or artifacts with placeholder text.
7. Turning pre-spec challenge into ritualized checklist coverage or fixed question quotas.
8. Treating review coverage as more important than solving the real delivery risk.
