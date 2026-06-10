---
name: specification-review-session
description: "Own a session dedicated only to mandatory specification review for this repository. Use after a non-trivial task-local `spec.md` is marked review-ready and before technical design, compact lean tasking, planning, or implementation starts, to run read-only subagent review lanes, reconcile findings to PASS, CONCERNS, or FAIL, update workflow-control review records, and stop without editing `spec.md`, `design/`, `tasks.md`, or code."
---

# Specification Review Session

## Purpose
Run only the post-specification review checkpoint for one task-local session.

This wrapper makes the `spec.md -> specification review -> technical design/planning` boundary explicit. It reviews the completed spec for breadth, depth, decision coverage, assumptions, proof obligations, and downstream readiness. It does not write the spec, design, task ledger, tests, runtime code, or implementation handoff.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist.
- Use the minimum context, references, tools, and validation loops that can change the deliverable.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change review verdict, scope, safety, or ownership.
- When evidence is missing or conflicting, retry once with a targeted read or label the blocker instead of treating absence as proof.
- Finish only when the review verdict, readiness consequence, reopen target, and next-session route are recorded.

## Use When
- a non-trivial task-local `spec.md` exists and is marked review-ready or equivalent by the specification phase;
- workflow routing says the next session starts with `specification review`;
- technical design, compact lean tasking, planning, or implementation is blocked on the mandatory spec-review verdict;
- a repaired `spec.md` needs follow-up review after a prior `FAIL`;
- workflow-control artifacts need the specification-review gate status recorded or repaired without changing `spec.md`.

## Skip When
- the work is direct path and no task-local `spec.md` is expected;
- `spec.md` is missing, still draft, or still needs specification authoring;
- the user asks to write or repair `spec.md`; use `specification-session` or `spec-document-designer`;
- the task has already moved into technical design or later and no reopen target points back to specification review;
- the request tries to combine review with design, planning, implementation, validation, or code changes.

## Required Inputs
Specification review may begin only when the minimum review packet exists:
- review-ready task-local `spec.md`;
- task-local `workflow-plan.md` when used;
- task-local `workflow-plans/specification.md` when used;
- prior `workflow-plans/specification-review.md` when this is follow-up review;
- preserved `research/*.md`, formal clarification fan-in, and source-of-truth artifacts named by the spec when the spec relies on them;
- known blockers, accepted assumptions, non-goals, constraints, proof obligations, and reopen conditions.

If a required input is missing or contradictory, record `FAIL` or blocked with the smallest reopen target instead of reconstructing the spec inside review.

## Read First
Always read:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/subagent-contract.md`
- `docs/subagent-brief-template.md`

Then read current phase context in this order:
1. task-local `workflow-plan.md`, if present
2. task-local `workflow-plans/specification-review.md`, if present
3. task-local `workflow-plans/specification.md`, if present
4. task-local `spec.md`
5. selected `research/*.md`, formal clarification fan-in, and source-of-truth artifacts only when the spec or workflow-control record names them

Rules:
- follow `AGENTS.md` if workflow guidance conflicts;
- read the master `workflow-plan.md` before phase-local files when both exist;
- do not broad-read unrelated repository surfaces when the spec already names the evidence boundary;
- if phase context shows the task should reopen specification, research, or user decision before review, stop and record that route.

## Allowed Writes
This session may write or update only:
- task-local `workflow-plan.md`;
- task-local `workflow-plans/specification-review.md`;
- the `workflow-plans/` directory only when it must be created to hold the phase-local review file;
- a narrow status-only marker in `spec.md` only when an existing spec status field must be synchronized from `review_ready` to `approved` after `PASS` or eligible `CONCERNS`.

Do not edit spec content, decisions, assumptions, validation text, `design/`, `tasks.md`, tests, runtime code, generated files, or implementation handoff. If spec content must change, return `FAIL` and reopen specification.

## Core Defaults
- Specification review is mandatory for non-trivial task-local `spec.md`.
- Review lanes are read-only and advisory; the orchestrator owns fan-in, verdict, accepted risks, proof obligations, and reopen routing.
- Use at least one distinct read-only review lane for non-trivial work. Use multiple lanes when independent lenses could change readiness.
- Common lenses are product/scope coherence, domain invariants and edge cases, API/data/source-of-truth effects, architecture ownership, security/reliability/delivery, validation/QA, dependency/OSS, Pattern Fit, and legacy cleanup.
- A scoped-down review must list candidate lenses considered and why omitted lenses cannot change readiness. Local-only review is valid only for explicit direct-path/prototype waiver or unavailable read-only lane execution with recorded consequence.
- Missing explicit subagent authorization is not unavailable read-only lane execution and is not a valid local-only or scoped-down review rationale. If required review lanes are blocked only because the current prompt lacks explicit subagent/delegation authorization, record the review gate as blocked and return a next-session prompt with `Subagent authorization:`.
- Review does not patch the spec. It either passes, passes with named concerns, or fails with a reopen target.

## Workflow

### 1. Confirm This Session Owns Specification Review Only
- confirm the task is at `specification review` or explicitly resuming that checkpoint;
- if specification is incomplete, route back to `specification`;
- if review already passed and the real next step is technical design or planning, stop and point to the recorded next phase;
- if implementation already started and the review gate is missing, report the missing gate as a reopen blocker instead of creating post-hoc approval.

### 2. Build The Review Lane Plan
- identify the decision frontier as concrete review questions, not generic topics;
- start from the default lenses and retain only lenses that can change downstream readiness;
- split independent questions across lanes; merge duplicate or consequence-only lenses;
- choose one skill or `no-skill` per lane;
- enforce read-only execution by tool choice, not prompt wording alone.
- prepare a lens coverage table with each considered lens marked `covered`, `not_applicable`, `concern`, or `fail`; `PASS` is not valid until every readiness-critical lens has a status and evidence pointer.

### 3. Run Read-Only Review Lanes
- use `docs/subagent-brief-template.md` specification review variant for each lane;
- ask lanes to cite artifacts and separate facts, assumptions, and risks;
- require each finding to use: `Spec anchor`, `Evidence`, `Impact`, `Classification`, and `Required disposition`;
- keep lane outputs compact and findings-first;
- do not let lanes approve, edit, mutate git state, or alter task handoffs.

### 4. Reconcile Findings
- deduplicate overlapping findings;
- resolve blocker-severity conflicts explicitly;
- classify surviving findings as `blocks_spec_approval`, `reopens_specification`, `reopens_research`, `requires_user_decision`, `accepted_risk_candidate`, `proof_obligation`, or `record_only`;
- choose `PASS`, `CONCERNS`, or `FAIL`;
- for `CONCERNS`, name every accepted risk and proof obligation and the downstream artifact that must carry it;
- for `FAIL`, name the smallest reopen target and the condition a follow-up review must verify.

### 5. Write Or Repair `workflow-plans/specification-review.md`
Record only phase-local review orchestration:
- reviewed `spec.md` path and review packet;
- lane plan, lane status, and read-only enforcement;
- lens coverage table with status and evidence pointer;
- lane result summary;
- orchestrator fan-in;
- gate result: `PASS`, `CONCERNS`, or `FAIL`;
- accepted risks and proof obligations;
- readiness consequence;
- reopen target when blocked or failed;
- phase status;
- completion marker;
- local stop rule;
- next action and what can run in parallel.

Keep this file routing and review-status only. Do not turn it into a second spec, design bundle, task ledger, or raw transcript dump.

### 6. Write Or Repair `workflow-plan.md`
When workflow-control exists, update the master file with:
- current phase set to `specification-review` and current phase status;
- link or status for `workflow-plans/specification-review.md`;
- `spec.md` status as `approved`, `approved_with_concerns`, `review_ready`, `blocked`, or `draft`;
- specification-review gate result and lane status;
- accepted spec risks and proof obligations when `CONCERNS`;
- blockers and reopen target when `FAIL` or blocked;
- `Session boundary reached`;
- `Ready for next session`;
- `Next session starts with`;
- `Next session context bundle`.

Do not leave the review verdict or next phase only in chat.

### 7. Stop At The Boundary
- If `PASS`, route to triggered `system-integration-design`, compact lean planning, or the next recorded phase.
- If `CONCERNS`, route to the next phase only with named obligations.
- If `FAIL`, route to the smallest reopen target.
- Do not begin the next phase in this session.

## Required Final Chat Handoff
When this session ends with `Session boundary reached: yes` and `Ready for next session: yes`, the final chat response must include a `Recommended next-session prompt` section with one copy-pastable fenced text block.

The prompt must name exactly one next phase or reopen target, list the artifacts to read first, state the expected output for that one phase, and include the stop rule. For any non-trivial next phase or reopen target that may use read-only lanes, include `Subagent authorization: I explicitly request and authorize read-only subagents, delegation, and parallel agent work for every repository workflow gate that requires or benefits from fan-out in this session. Spawn the required read-only lanes without asking again; the orchestrator retains final authority and reconciles results.` If there is no next session or `Ready for next session: no`, do not invent a prompt.

Keep the prompt chat-only; do not write it into workflow artifacts or create a new artifact for it. Workflow files may record only the next phase, start point, context bundle, blockers, accepted risks, and proof obligations needed to render the chat prompt.

## Stop Condition
The session is complete when:
- the review packet was inspected or a missing-input blocker was recorded;
- required lanes have completed, been scoped down with rationale, or are explicitly blocked;
- findings are reconciled;
- the review gate result is `PASS`, `CONCERNS`, or `FAIL`;
- workflow-control state and next-session routing are updated;
- no spec content, design, planning, implementation, or validation work has started.

## Escalate When
Escalate instead of forcing a verdict when:
- `spec.md` is not review-ready;
- a lane needs a domain decision outside its scope;
- a missing fact belongs to research or a user decision;
- review findings require changing `spec.md` content;
- the request tries to combine review with specification repair, design, planning, or implementation.

## Anti-Patterns
- treating `spec-clarification-challenge` as a substitute for post-spec review;
- approving a spec from the author's own confidence without a distinct review record;
- asking one generic lane to "review everything" when independent lenses can change readiness;
- patching spec decisions during review instead of reopening specification;
- downgrading missing product, contract, validation, or ownership decisions to proof obligations;
- starting technical design or planning because the review looks likely to pass;
- leaving `CONCERNS` without named obligations and downstream owner artifacts.
