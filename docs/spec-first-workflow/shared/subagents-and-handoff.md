# Subagents And Handoff

Use built-in subagents and session handoffs only when they reduce a real context, independence, coordination, or resume problem.

## Read When

- Distinguishing a built-in subagent lane from the native App implementation Worker.
- Requiring an independent verdict.
- Resuming a task or handing it to another session.

## Inputs

- One accepted goal or one concrete review question.
- The smallest artifact and source bundle needed to answer it.
- Clear read-only and external-action boundaries.

## Outputs

- A local decision or a small lane plan.
- Evidence and recommendations for root synthesis.
- A compact handoff only when another session or actor is actually needed.

## Stop Rule

Do not create a built-in subagent lane for work that is sequential, tightly coupled to the root's reasoning, or cheaper to do locally. Do not create a session handoff when the current session can safely finish.

## Delegation Decision

Skills define method; subagents provide separate context and independence. At the start of each non-implementation macro phase, identify materially affected domains and apply matching skills locally by default. Delegate only one concrete bounded question when independent evidence, separate context, parallelism, or review independence can change the result. A domain name, matching keyword, skill handoff, or desire for more confidence alone never creates a lane.

If no separate read-only lane helps, keep the reasoning in the root and state the reason only in an existing phase artifact or handoff; do not create a standalone gate record. Required independent reviews still use a separate read-only reviewer.

Use a subagent when all are true:

- the question is concrete and bounded;
- it can be answered independently of mutable work in other lanes;
- separate context or review independence materially improves the result;
- the output can be checked and synthesized by the root.

Good lanes include independent source research, one specialist design question, or review of a fixed non-implementation revision. Bad lanes include broad “review everything,” duplicate lenses, tiny lookups, and any implementation review, acceptance, specialist analysis, re-review, or repair.

The root owns scope, lane choice, synthesis, integration, task acceptance, and completion claims. Built-in subagents are read-only research, challenge, or review lanes; they never implement or repair code, config, docs, or tests. The native App Worker is outside this contract and follows the [implementation phase](../phases/implementation-validation-closeout.md#worker-assignment-and-acceptance).

Default to at most three concurrent research/review lanes, one bounded wave, and no nested delegation. Additional sequential lanes still require distinct decision-changing questions that could not be covered locally or in the first wave. If a lane exposes a new owner decision, return it to the root rather than expanding scope.

For read-only subagents, choose the currently available model and reasoning effort from task difficulty, evidence volume, latency/cost, and consequence of error. Re-review should be at least as capable as the review that found the issue. The implementation phase separately owns the App Worker lifecycle.

## Lane Brief

One lane owns one question, one evidence boundary, and one root disposition. Keep the brief outcome-first:

```text
Question: <one decision or falsification target>
Context: <accepted facts and minimal artifact paths>
Evidence boundary: <what to inspect and what counts>
Constraints: <read-only boundary, non-goals, external-action limits>
Output: <finding/evidence/recommendation shape>
Stop: <missing input, conflict, or completion condition>
```

Do not copy the repository workflow, generic strictness language, or unrelated artifact summaries into every brief.

## Autonomous Pre-Review Challenge

The focused [Autonomous Pre-Review Challenge](autonomous-pre-review-challenge.md) owns its vocabulary, authority, state, continuation, exhaustion, invalidation, and separation from the required reviewer. This document continues to own delegation, review independence, fan-in, resume, and handoff mechanics.

## Review Independence

Use the shared [Review Finding
Envelope](../../subagent-contract.md#shared-review-finding-envelope) for every
review return.

Structured and orchestrated work requires an independent reviewer for a standalone `research only` synthesis, the completed specification, any triggered technical design or test design, and the completed task ledger before implementation. These non-implementation gates use a separate read-only subagent following the owning phase review method. An explicitly requested independent review of completed implementation is a separate read-only boundary after implementation, not an internal gate.

The reviewer:

- reads a fixed artifact revision or diff;
- reports anchored findings and a verdict recommendation;
- does not edit or approve its own repair;
- distinguishes blockers, bounded concerns/proof obligations, and non-blocking observations;
- states the evidence boundary and disposition of every materially affected lens: covered, delegated to a named specialist result, or not triggered with a concrete reason.

When a non-implementation independent gate is required, one whole-artifact reviewer is the default. That reviewer applies compatible matching methods locally in one coherence pass and accounts for every materially affected lens. Run a specialist before that gate only for one concrete high-impact question the root cannot credibly cover locally. If the gate reviewer discovers such an uncovered question, run one bounded specialist follow-up and return only that disposition for a focused verdict update; do not repeat an unchanged whole revision. A domain label, generic handoff, or desire for more confidence does not justify another reviewer.

For each applicable required non-implementation boundary, canonical review convergence is: authoring bar -> required challenge `DONE` -> exact fixed candidate -> independent findings and verdict -> owning-author repair or upstream reopen -> focused fresh re-review of changed and transitively affected lenses -> `PASS`. The root collects all in-scope actionable findings for that fixed revision; the same gate reviewer re-reviews repairs and reuses unaffected lens dispositions by default. Use full affected-surface re-review only when the repair changes shared assumptions or crosses a domain boundary. Re-review must be at least as capable as the review that found the issue. Implementation findings instead follow the root-owned Worker correction loop in the implementation phase.

A non-implementation macro phase reaches review convergence only when its latest required review returns `PASS` and finds no blocker, known current-phase defect, unowned question, uncovered materially affected lens, or unresolved cross-lens contradiction. `CONCERNS` is non-terminal: a bounded risk or downstream proof obligation still needs disposition in the owning phase, and it never permits phase movement. The owning author repairs it. The root records authorized acceptance with evidence, owner, and reopen condition, or splits/reopens scope, then obtains focused fresh review; `PASS` means every concern has a disposition, not that no residual risk exists. `FAIL` blocks movement and requires repair or reopening before focused fresh review. Non-blocking observations do not prevent `PASS`. Repeat only while a concrete new finding or semantic repair changes readiness. If closure needs unavailable evidence, authority, or an upstream decision, mark the phase blocked and reopen that owner. Do not repeat an unchanged `PASS` revision or launch speculative lenses merely to collect confidence.

A semantic mutation after review to the reviewed non-implementation artifact invalidates convergence only for affected lenses. Revalidate affected proof and obtain focused fresh review before phase movement.

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

Treat handoff as a chain of custody: name the accepted source, movement evidence, next owner, authority boundary, proof obligation, next executable action, and exact stop/reopen owner and condition so the receiver can continue without reconstructing chat.

When the current request stops at a true macro-phase boundary and a next macro phase or external/upstream reopen owner exists, the final chat response MUST end with a copy-pastable next-session prompt; this is the default and needs no separate user request. Crossing a macro phase inside the same authorized request does not require a prompt. Do not emit one for an internal checkpoint or when the workflow is honestly complete with no next phase or reopen owner.

Required research-synthesis challenge, specification review, technical-design review, test-design QA review, and task-readiness review are internal checkpoints of their non-implementation owning macro phase. The root launches the required read-only lane, repairs authoritative work, obtains any required fresh verdict, and continues automatically in the same session. A durable review record is only a carrier; it does not create a user-started phase or a next-session prompt.

An explicitly user-requested standalone review remains read-only: return the
complete review result and stop at the requested review boundary. It gains no
repair, implementation, or workflow-handoff authority unless the user separately
grants it.

When that request independently reviews completed implementation, it begins only
after implementation/validation/closeout has ended and never retroactively becomes
an implementation acceptance or closeout gate.

An allowed handoff to a non-implementation macro phase contains:

```text
Objective: <one next outcome>
Read first: <one owning artifact, then only non-obvious context>
Constraints: <only task-specific boundaries not recoverable from AGENTS.md>
Movement evidence: <current closure and required PASS>
Proof: <required evidence or owning ledger section>
Next action: <first executable action>
Stop/reopen: <exact blocker behavior and owner>
```

An implementation/validation/closeout handoff contains:

```text
Goal: <one durable outcome>
Completion: <root acceptance, final integrated review, and fresh proof>
Read first: <accepted inline outcome or ready tasks.md, plus minimal context>
Constraints: <task-specific boundaries>
Movement evidence: <accepted input or ready ledger and current proof>
Proof: <required evidence or ledger section>
Next action: <first executable action>
Stop/reopen: <exact blocker behavior and owner>
```

Reserve `Goal:` for implementation handoffs. Earlier macro phases use `Objective:` and never create or continue a Codex Goal.

Keep prompts in chat unless the user asks for a standalone prompt artifact. Do not include worker command manuals, model catalogs, full repository summaries, repeated authorization policy, or empty headings.
