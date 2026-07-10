# Subagents And Handoff

Use delegation and handoffs only when they reduce a real context, independence, coordination, or resume problem.

## Read When

- Considering research, review, challenge, or implementation delegation.
- Requiring an independent verdict.
- Resuming a task or handing it to another session.

## Inputs

- One accepted goal or one concrete review question.
- The smallest artifact and source bundle needed to answer it.
- Clear read/write and external-action boundaries.

## Outputs

- A local decision or a small lane plan.
- Evidence and recommendations for root synthesis.
- A compact handoff only when another session or actor is actually needed.

## Stop Rule

Do not delegate work that is sequential, tightly coupled to the root's reasoning, or cheaper to do locally. Do not create a handoff when the current session can safely finish.

## Delegation Decision

At the start of each active macro phase, the root evaluates whether separate evidence, specialist, challenge, or review lanes would improve the result. If none would, keep the work local and state the reason only in an existing phase artifact or handoff; do not create a standalone gate record.

Use a subagent when all are true:

- the question is concrete and bounded;
- it can be answered independently of mutable work in other lanes;
- separate context or review independence materially improves the result;
- the output can be checked and synthesized by the root.

Good lanes include independent source research, one specialist design question, or review of a fixed revision. Bad lanes include broad “review everything,” duplicate lenses, tiny lookups, and tasks whose next step depends on each preceding result.

The root owns scope, lane choice, synthesis, edits, and completion claims. Research and review lanes are read-only. Implementation workers may write only inside their assigned boundary; use isolation when concurrent writes or risky experiments justify it, not by default.

Default to at most three concurrent lanes and no nested delegation. Run dependent questions sequentially. If a lane exposes a new owner decision, return it to the root rather than expanding scope.

Choose the currently available model and reasoning effort from task difficulty, evidence volume, latency/cost, and consequence of error. Do not hard-code a dated model catalog in repository instructions or assume the largest setting is best. Re-review should be at least as capable as the review that found the issue.

## Lane Brief

Keep the brief outcome-first:

```text
Question: <one decision or falsification target>
Context: <accepted facts and minimal artifact paths>
Evidence boundary: <what to inspect and what counts>
Constraints: <read-only/write boundary, non-goals, external-action limits>
Output: <finding/evidence/recommendation shape>
Stop: <missing input, conflict, or completion condition>
```

Do not copy the repository workflow, generic strictness language, or unrelated artifact summaries into every brief.

## Review Independence

Structured and orchestrated work requires an independent reviewer for the completed specification, any triggered technical design or test design, and the completed implementation ledger. Direct work uses an independent reviewer when required by the user or when the decision is hard to reverse, materially high-impact, ambiguous, or poorly falsified by the author alone.

The reviewer:

- reads a fixed artifact revision or diff;
- reports anchored findings and a verdict recommendation;
- does not edit or approve its own repair;
- distinguishes blockers, bounded concerns/proof obligations, and non-blocking observations.

The root repairs in-scope findings and decides whether the changed surface needs focused or full re-review. Repeat only while new evidence or repairs change the result; stop and name the blocker when the same issue survives reasonable attempts without a new decision path.

## Fan-In

For each material lane result, keep only:

- conclusion and strongest evidence;
- uncertainty or conflict;
- root disposition: accept, reject, repair, carry as proof/risk, or reopen;
- destination artifact or owner.

Do not paste raw transcripts into authoritative artifacts.

## Resume

Resume from artifacts, not remembered chat:

1. `tasks.md` for implementation/validation;
2. otherwise `workflow-plan.md` for multi-session coordination;
3. then only the spec/design/test/research/rollout artifacts named by the current next action.

If those sources disagree, the task is blocked until the narrowest owner reconciles them.

## Handoff

A next-session handoff is permitted only when work intentionally stops at a true macro-phase boundary and the next outcome belongs to a later macro phase, including when the user explicitly requests that macro-phase handoff, or when a stop condition in [Phase Movement](../../spec-first-workflow.md#phase-movement) prevents the current root from continuing without an earlier-owner decision, user authority, external evidence/action, or required tooling. Crossing a macro phase inside the same authorized request does not itself require a prompt, and requesting a separate session for an internal checkpoint does not make it eligible.

Specification review, technical-design review, test-design QA review, task-readiness review, post-code review, in-scope repair, fresh re-review, validation, and closeout are internal checkpoints of their owning macro phase. The root launches the required read-only lane, repairs authoritative work, obtains any required fresh verdict, and continues automatically in the same session. A durable review record is only a carrier; it does not create a user-started phase or a next-session prompt.

An explicitly user-requested standalone review remains read-only: return findings and stop at the requested review boundary. It gains no repair, implementation, or workflow-handoff authority unless the user separately grants it.

An allowed handoff contains:

```text
Goal: <one next outcome>
Read first: <one owning artifact, then only non-obvious context>
Constraints: <only task-specific boundaries not recoverable from AGENTS.md>
Proof: <required evidence or owning ledger section>
Stop/reopen: <exact blocker behavior and owner>
```

Keep prompts in chat unless the user asks for a standalone prompt artifact. Do not include worker command manuals, model catalogs, full repository summaries, repeated authorization policy, or empty headings. Emit no next-session prompt for an internal checkpoint or completed task.
