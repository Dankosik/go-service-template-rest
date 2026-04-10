# Spec-First Workflow (Orchestrator/Subagent-First)

## 1. Purpose

This document defines the repository-level spec-first workflow.
It keeps one universal loop for small and large changes while moving execution away from skill-centric choreography and toward orchestrator/subagent-first execution.
`AGENTS.md` should explicitly load this file for non-trivial or agent-backed work; this document is the companion runtime reference for detailed workflow mechanics. If this file and this repository contract ever diverge, follow `AGENTS.md`.

The workflow is intentionally simple:
- no mandatory phase matrix,
- no mandatory user intake schema,
- no linear skill chain in the main flow,
- no artifact fan-out unless it removes real risk.

## 2. Core Position

1. The main flow is owned by the orchestrator.
2. The orchestrator keeps the full task context and makes the final decisions.
3. Non-trivial or agent-backed work starts with `workflow planning` before any subagent fan-out.
4. Subagents are the default mechanism for non-trivial research and review tracks.
5. Subagents are always `research-only/read-only`.
6. Read-only is enforced by agent and tool choice, not by prompt wording alone; if a surface cannot reliably stay read-only, keep that work local.
7. Skills are on-demand tools, usually used inside subagents rather than as the main workflow.
8. In the main flow, planning skills may be used only during implementation planning, and implementation skills only during implementation.
9. `spec.md` is the canonical decisions artifact.
10. `research/*.md` stores validated research context and does not replace `spec.md`.
11. Coding starts only after an explicit implementation plan exists.
12. Any single subagent conclusion is advisory until the orchestrator synthesizes it and, when needed, rechecks it.
13. For medium/high-risk or ambiguous work, candidate synthesis is pressure-tested before planning or explicitly waived with rationale.
14. One subagent pass uses at most one skill. If a lane needs multiple skills, split it into multiple lanes or keep that synthesis in the orchestrator.
15. Parallel lanes may reuse the same subagent role; role duplication is normal when the question or chosen skill differs.
16. In `fan-out` mode, do not optimize for low subagent count; prefer enough lanes to cover the real seams, even if that means duplicate or overlapping reads.

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
  workflow-plan.md
  research/
    <topic>.md
  plan.md
  test-plan.md
```

Artifact rules:
1. Keep `spec.md` as the single source of truth for final decisions.
2. Use `workflow-plan.md` for non-trivial or agent-backed work so the orchestration survives beyond chat history and short-term memory.
3. Use `research/*.md` only when the task is long, ambiguous, or likely to benefit from reusable validated context.
4. Keep `research/*.md` flexible and task-driven; there is no mandatory universal template.
5. Use `plan.md` as the dedicated coder-facing execution plan when the workflow plan says implementation should not be driven from `spec.md` alone, especially for non-trivial implementation work, implementation-skill handoff, or parallelized execution.
6. Use `test-plan.md` only when test obligations are large enough to hide the core plan.
7. Do not duplicate decision text across files; link instead.
8. Keep raw pre-spec challenge transcripts out of `spec.md`; record only the resolved decisions and remaining open questions.

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
5. If implementation should be executed from a separate coder plan, keep only the control summary in `spec.md` and link to `plan.md`.
6. If the test strategy becomes too large, move that material to `test-plan.md` and keep only the control summary in `spec.md`.

For non-trivial tasks, keep a compact audit trail in `spec.md` and linked artifacts:
1. intake summary,
2. `workflow-plan.md` or equivalent workflow record, including subagent lanes, order/parallelism, fan-in/challenge path, and phase execution policy,
3. research questions or subagent tracks,
4. decision log,
5. material overrides or rejected paths,
6. open questions with owner and unblock condition,
7. implementation plan,
8. validation evidence,
9. outcome.

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

### Step 2. Workflow Planning And Research Routing

Owner: Orchestrator

The orchestrator decides:
1. which execution shape fits the task,
2. whether research mode is `local` or `fan-out`,
3. which questions can be solved directly in the main flow,
4. which questions need subagent research,
5. which single skill each planned subagent lane should use, if any,
6. which tracks are independent and should run in parallel,
7. which high-impact or ambiguous topics need multi-angle research or a challenger pass,
8. whether later `plan.md` or `test-plan.md` artifacts will be required.

The orchestrator should not treat subagent count as a budget to minimize.
In `fan-out` mode, it is better to over-cover the task with extra read-only lanes than to leave a material seam unexamined.

For non-trivial or agent-backed work, write `workflow-plan.md` before any subagent call.
Preferred shape: a detailed sequence diagram or an equivalent ordered lane list.
Keep it explicit enough that the orchestration can be resumed from the file rather than from memory.
If the number of implementation phases is not known yet, say so directly and still record the execution policy: phased delivery, one phase at a time, with review and validation between phases.

Example:

```text
orchestrator -> workflow-plan.md : record fan-out, fan-in, and phased execution policy
orchestrator -> architecture-agent : boundary ownership and seams
orchestrator -> data-agent : source-of-truth and migration risk
orchestrator -> reliability-agent : timeout / retry / degradation risk
orchestrator -> security-agent : trust boundaries and abuse risk
architecture-agent -> orchestrator : findings
data-agent -> orchestrator : findings
reliability-agent -> orchestrator : findings
security-agent -> orchestrator : findings
orchestrator -> challenger-agent : challenge candidate decisions
challenger-agent -> orchestrator : blocker questions / next action
opt seam needs specialist follow-up
  orchestrator -> relevant specialist agent : targeted re-research
  relevant specialist agent -> orchestrator : findings
  orchestrator -> challenger-agent : rerun challenge on reopened seam if still planning-critical
end
orchestrator -> plan.md : phased coder-facing execution plan after challenge resolution
loop for each phase in plan.md
  orchestrator -> go-coder : execute one phase only
  go-coder -> orchestrator : code + local evidence
  orchestrator -> review agents : phase review for touched domains
  review agents -> orchestrator : findings / approvals / reopen signals
  orchestrator -> validation : run phase checkpoint
end
```

Example: same subagent role, different skills, different research questions

```text
task: add workspace credit accounting with an authoritative Postgres ledger plus a Redis summary cache

orchestrator -> workflow-plan.md : record duplicate-role fan-out with one skill per lane
orchestrator -> data-agent[ledger-ownership | go-data-architect-spec] : research authoritative tables, replay rules, migration/backfill safety
orchestrator -> data-agent[runtime-cache-contract | go-db-cache-spec] : research cache keys, invalidation, staleness, stampede, fallback
data-agent[ledger-ownership | go-data-architect-spec] -> orchestrator : findings
data-agent[runtime-cache-contract | go-db-cache-spec] -> orchestrator : findings
orchestrator -> challenger-agent[pre-spec-challenge] : challenge the combined candidate decisions
challenger-agent[pre-spec-challenge] -> orchestrator : blocker questions / next action
```

Why this example matters:
1. The same `data-agent` role is called more than once, and that is intentional.
2. Each `data-agent` lane uses exactly one skill.
3. Each lane researches a different seam of the same task.
4. This is preferred over one `data-agent` trying to use both skills in one pass.

Lane planning rules:
1. Plan by lane, not by unique role name.
2. A lane owns one question and uses one skill or explicit `no-skill`.
3. Multiple parallel lanes may reuse the same subagent role when that keeps the questions sharp.
4. Do not force a single lane to mix skills just to avoid duplicate role names.
5. If you are unsure whether a material seam deserves its own lane, bias toward opening the lane.

Each subagent task should state:
1. the goal and scope,
2. what must be checked,
3. relevant constraints and risks,
4. the expected output shape,
5. the one skill to use for that pass, or explicit `no-skill`,
6. the explicit read-only boundary: no code, file, git-state, or implementation-plan changes,
7. where evidence is required.

### Step 3. Local Or Parallel Research

Owner: Subagents

Subagents:
1. analyze only their assigned domain,
2. own one question per lane,
3. may use one relevant skill locally or explicit `no-skill`,
4. return concise synthesis-ready output,
5. separate facts, recommendations, risks, confidence, and open points when relevant,
6. never change repository files, code, or the implementation plan.

There is no single rigid response schema for every subagent task.
The orchestrator defines the output shape needed for the current question.
If a planned track cannot be executed with read-only guarantees, keep it in the main flow instead of delegating it.
When research mode is `fan-out`, optimize for domain coverage rather than for the smallest possible number of subagent calls.
If one question seems to need multiple skills, split it into multiple lanes instead of turning one subagent into a mini workflow.

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
5. If the challenger returns a research-needed seam, reopen specialist research rather than letting the orchestrator silently answer the domain question alone.
6. After targeted re-research, come back through synthesis and rerun challenge if that seam is still planning-critical.

### Step 5. Implementation Planning

Owner: Orchestrator

This step is mandatory before coding.
It is distinct from `workflow planning`: the early phase decides orchestration and artifact expectations, while this phase produces the coder-facing execution plan after research and challenge resolution.

The orchestrator turns decisions into:
1. ordered execution steps,
2. dependencies and checkpoints when relevant,
3. minimal validation per meaningful slice,
4. rollback or mitigation notes when the change carries real operational risk.

Planning is context-driven, not template-driven.
Only include rollout, backward-compatibility, migration, or rollback detail when the task actually needs it.
For non-trivial implementation work, long or parallelized execution, or any implementation-skill handoff, this plan should live in `plan.md` rather than being reconstructed from `spec.md`.
Phased implementation is the default for non-trivial work: `phase -> review/reconcile -> validate -> next phase`. A single-pass big-bang plan needs explicit rationale.

Recommended `plan.md` shape for non-trivial work:

```text
# Execution Context
- spec link
- goal of this implementation pass
- non-goals / fixed constraints

# Phase Plan
## Phase 1. <smallest reviewable increment>
- Objective
- Depends On
- Tasks
- Change Surface
- Planned Verification
- Review / Checkpoint
- Exit Criteria

## Phase 2. <next increment>
...

# Cross-Phase Validation Plan
- what is checked after each phase
- what is deferred until the final checkpoint

# Blockers / Assumptions

# Handoffs / Reopen Conditions
```

Phase rules:
1. Each phase should usually be the smallest reviewable increment rather than one large subsystem rewrite.
2. A completed phase should give the orchestrator a natural stop point for review, targeted testing, and decision to continue.
3. Prefer sequential phases by default; parallel phase lanes are for truly disjoint work only.
4. If a phase is not independently mergeable or testable, name the coupling explicitly instead of pretending it is standalone.
5. Each phase should name the review/reconciliation checkpoint that gates the next phase.

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
2. Write `workflow-plan.md` before any non-trivial fan-out.
3. Spawn all read-only subagent tracks needed to cover the materially affected domains, including duplicate-role lanes when they keep one-skill ownership clean.
4. Synthesize candidate decisions and run pre-spec challenge when the task risk or ambiguity justifies it.
5. Write `plan.md` before coding when implementation is non-trivial.
6. Execute one phase at a time.
7. Run phase review, reconciliation, and phase validation before moving to the next phase.

## 7. When To Fan Out

Use subagent fan-out when at least one is true:
1. the task crosses multiple domains,
2. a decision is high-impact or hard to reverse,
3. confidence is low after a first pass,
4. you need an independent challenger or second opinion,
5. a review wave would reduce meaningful delivery risk,
6. the task is long enough to justify preserved research memory.

Do not fan out when the extra orchestration would add more overhead than clarity.
Do not fan out to a write-capable delegate surface just because the prompt says "read-only"; if the surface is not reliably read-only, keep the work local.
Do not optimize for minimal subagent count on a materially cross-domain task; optimize for coverage of the real affected domains.
Subagent count is not a success metric in this workflow. Use as many read-only lanes as needed, and when coverage and economy conflict, choose coverage.
Prefer more lanes over fewer when the extra lanes buy independent opinions, seam isolation, or stronger fan-in evidence.
Do not treat duplicate-role lanes as a smell by themselves; the smell is a lane that tries to answer multiple skill-shaped questions at once.

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
2. For non-trivial or agent-backed work, `workflow-plan.md` exists and makes the research mode, subagent lanes, fan-in path, and later artifact expectations explicit.
3. Critical unknowns are either resolved, delegated for research, or tracked explicitly.
4. For medium/high-risk or ambiguous work, the pre-spec challenge checkpoint is reconciled or explicitly waived.
5. An implementation plan exists before coding starts.

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
5. Spawning write-capable delegate agents under the subagent role instead of keeping those tasks in the main flow.
6. Under-fanning-out to save subagent calls while leaving materially affected domains unexamined.
7. Starting implementation before the planning step is explicit.
8. Running non-trivial implementation as one big-bang pass without phase checkpoints.
9. Forcing the coder to reconstruct execution order from `spec.md` when the task already needs a separate `plan.md`.
10. Filling optional sections or artifacts with placeholder text.
11. Turning pre-spec challenge into ritualized checklist coverage or fixed question quotas.
12. Treating review coverage as more important than solving the real delivery risk.
13. Packing multiple skills into one subagent pass instead of splitting the work into separate lanes.
14. Treating low subagent count as a virtue on a task that actually needs broader specialist coverage.
