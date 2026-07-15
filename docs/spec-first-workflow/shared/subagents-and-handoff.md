# Subagents And Handoff

Use built-in subagents and session handoffs only when they reduce a real context, independence, coordination, or resume problem.

## Read When

- Distinguishing a built-in subagent lane from an external implementation Worker.
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

Skills define method; subagents provide separate context and independence. At the start of each non-implementation macro phase, identify materially affected domains and apply matching skills locally by default. Delegate only one concrete bounded question when independent evidence, separate context, parallelism, or review independence can change the result. During implementation/validation/closeout, the root applies every matching review skill locally and launches no built-in subagent lane. A domain name, matching keyword, skill handoff, or desire for more confidence alone never creates a lane.

If no separate read-only lane helps, keep the reasoning in the root and state the reason only in an existing phase artifact or handoff; do not create a standalone gate record. Required independent reviews still use a separate read-only reviewer.

Use a subagent when all are true:

- the question is concrete and bounded;
- it can be answered independently of mutable work in other lanes;
- separate context or review independence materially improves the result;
- the output can be checked and synthesized by the root.

Good lanes include independent source research, one specialist design question, or review of a fixed non-implementation revision. Bad lanes include broad “review everything,” duplicate lenses, tiny lookups, and any implementation review, acceptance, specialist analysis, re-review, or repair.

The root owns scope, lane choice, synthesis, integration, task acceptance, and completion claims. Built-in subagents are read-only research, challenge, or review lanes; they never implement or repair code, config, docs, or tests. External CLI Workers are not subagents and follow the [implementation phase contract](../phases/implementation-validation-closeout.md#cli-worker-launch-and-resume).

Default to at most three concurrent research/review lanes, one bounded wave, and no nested delegation. Additional sequential lanes still require distinct decision-changing questions that could not be covered locally or in the first wave. If a lane exposes a new owner decision, return it to the root rather than expanding scope.

For read-only subagents, choose the currently available model and reasoning effort from task difficulty, evidence volume, latency/cost, and consequence of error. Re-review should be at least as capable as the review that found the issue. The implementation phase separately pins the external Worker runtime.

## Lane Brief

Keep the brief outcome-first:

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

For each candidate routed here by the workflow, the root launches the existing
read-only challenger in internal grilling mode after the candidate meets its
authoring bar. Supply the current phase method, exact candidate revision,
accepted constraints, evidence boundary, authority boundary, and stop rule.
Explicit user-requested grilling remains a root-to-user dialogue; the internal
challenger never relays those questions.

The challenger inspects repository facts rather than asking for them, then
selects the highest-impact unresolved material branch. It may apply a materially triggered specialist method locally but never delegates recursively. Each turn
returns exactly one event: `QUESTION`, `HUMAN_REQUIRED`, `REOPEN`, or `DONE`.
Do not emit a questionnaire or a readiness verdict.

```text
QUESTION
Changes: <one current-phase decision>
Question: <one root-answerable question>
Recommended: <recommended answer>
Tradeoff: <main cost or risk>
Evidence: <artifact or repository anchor, or bounded assumption>

HUMAN_REQUIRED
Decision: <user-owned decision>
Authority reason: <why the root cannot decide>
Recommended: <recommended option when evidence supports one>
Tradeoff: <main cost or risk>
Dependency impact: <independent continuation or WAIT_HUMAN>

REOPEN
Owner: <evidence or upstream owner>
Gap or conflict: <missing evidence or contradicted decision>
Impact: <choice or readiness that cannot close>
Next evidence or repair: <smallest resolution route>

DONE
Resolved: <material decisions dispositioned in the latest candidate>
Assumptions: <bounded assumptions or none>
Residual risks: <owned risks or none>
Reopen when: <objective invalidation condition or none>
```

The root decides mechanism, system/package/file/instruction ownership, proof
strategy, and task order inside accepted behavior and authority. Undecided user
intent, observable behavior or scope, policy, new authority or external action,
and user-owned material risk acceptance return `HUMAN_REQUIRED`. Missing facts,
conflicting evidence, or an upstream decision gap return `REOPEN`.

For `QUESTION`, the root verifies the evidence and responds with `ACCEPT`, `OVERRIDE`, or `RECLASSIFY`. Record the selected decision or corrected owner,
strongest basis, destination in the owning candidate, bounded assumption, reopen
condition, and exact latest revision before the next turn. `RECLASSIFY` prevents
the root from answering for the user or hiding an upstream gap.

For `HUMAN_REQUIRED` or `REOPEN`, record the deduplicated item in the existing
owner, then respond with `CONTINUE_INDEPENDENT`, `WAIT_HUMAN`, or `REOPEN_OWNER`,
plus its destination, exact latest revision, and relevant open items.
`CONTINUE_INDEPENDENT` permits only branches that do not depend on the recorded
item; the other responses wait for the human answer or owner repair.

Continue dependent turns through the same challenger with the exact latest candidate.
The owning candidate is authoritative; the child transcript is not. If the
runtime cannot resume that child, relaunch the existing challenger from the
exact latest candidate and named open items rather than remembered chat. Do not create a probe transcript, receipt, queue, status, lifecycle field, or review verdict.

There is no question quota. Return `DONE` when no new or evidence-reopened
material current-phase decision remains; repeated dispositions, generic
category coverage, and questions with no affected choice are no progress.
Wording-only edits and repairs that apply an existing disposition reuse
completion. New evidence or a material change to a decision, assumption,
authority boundary, source of truth, or upstream dependency requires a fresh
probe; uncertain resume state also reruns it.

After `DONE`, a different read-only child reviews the exact latest candidate
under the owning phase's existing review method. The challenger never supplies
that verdict or replaces the required reviewer.

## Review Independence

Use the shared [Review Finding
Envelope](../../subagent-contract.md#shared-review-finding-envelope) for every
review return.

Structured and orchestrated work requires an independent reviewer for a standalone `research only` synthesis, the completed specification, any triggered technical design or test design, and the completed task ledger before implementation. These non-implementation gates use a separate read-only subagent following the owning phase review method. Implementation/validation/closeout never launches a built-in subagent: the root reviews and accepts every Worker result, applies matching review skills and specialist lenses locally, reviews the final integrated diff, and re-inspects corrections. An explicitly requested independent review of completed implementation is a separate read-only boundary after implementation, not an internal gate.

The reviewer:

- reads a fixed artifact revision or diff;
- reports anchored findings and a verdict recommendation;
- does not edit or approve its own repair;
- distinguishes blockers, bounded concerns/proof obligations, and non-blocking observations;
- states the evidence boundary and disposition of every materially affected lens: covered, delegated to a named specialist result, or not triggered with a concrete reason.

When a non-implementation independent gate is required, one whole-artifact reviewer is the default. That reviewer applies compatible matching methods locally in one coherence pass and accounts for every materially affected lens. Run a specialist before that gate only for one concrete high-impact question the root cannot credibly cover locally. If the gate reviewer discovers such an uncovered question, run one bounded specialist follow-up and return only that disposition for a focused verdict update; do not repeat an unchanged whole revision. A domain label, generic handoff, or desire for more confidence does not justify another reviewer.

The root collects all in-scope actionable findings for the fixed non-implementation revision. The owning author repairs them, and the same gate reviewer performs focused re-review of the repair and any transitively affected behavior, contract, ownership, or proof decision. Reuse unaffected lens dispositions by default; use full affected-surface re-review only when the repair changes shared assumptions or crosses a domain boundary. Re-review must be at least as capable as the review that found the issue. Implementation findings instead follow the root-owned Worker correction loop in the implementation phase.

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

A next-session handoff is permitted only when work intentionally stops at a true macro-phase boundary and the next outcome belongs to a later macro phase, including when the user explicitly requests that macro-phase handoff, or when a stop condition in [Phase Movement](../../spec-first-workflow.md#phase-movement) prevents the current root from continuing without an earlier-owner decision, user authority, external evidence/action, or required tooling. Crossing a macro phase inside the same authorized request does not itself require a prompt, and requesting a separate session for an internal checkpoint does not make it eligible.

Required research-synthesis challenge, specification review, technical-design review, test-design QA review, and task-readiness review are internal checkpoints of their non-implementation owning macro phase. The root launches the required read-only lane, repairs authoritative work, obtains any required fresh verdict, and continues automatically in the same session. Implementation acceptance, Worker repair, root re-inspection, validation, and closeout are root-owned internal checkpoints with no built-in subagent lanes. A durable review record is only a carrier; it does not create a user-started phase or a next-session prompt.

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
Proof: <required evidence or owning ledger section>
Stop/reopen: <exact blocker behavior and owner>
```

When the next outcome enters implementation/validation/closeout, use `codex-goal-prompt-composer` and its `Goal:` shape instead. `Goal:` is reserved for that implementation handoff; earlier macro-phase handoffs use `Objective:` and never create or continue a Codex Goal.

Keep prompts in chat unless the user asks for a standalone prompt artifact. Do not include worker command manuals, model catalogs, full repository summaries, repeated authorization policy, or empty headings. Emit no next-session prompt for an internal checkpoint or completed task.
