# Subagents And Handoff

Use built-in subagents and session handoffs only when they reduce a real context, independence, coordination, or resume problem.

## Read When

- Distinguishing a built-in subagent lane from the harness-native implementation Worker.
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

The root owns scope, lane choice, synthesis, integration, task acceptance, and completion claims. Built-in subagents are read-only research, challenge, or review lanes; they never implement or repair code, config, docs, or tests. In the Codex App a lane is a project subagent; in Claude Code it is an `Agent` tool lane ([Agent Harness](../../agent-harness.md#control-map)). The harness-native implementation Worker is outside this contract and follows the [implementation phase](../phases/implementation-validation-closeout.md#worker-execution).

Run one lane per distinct decision-changing question. Current harness capacity, mutable-state independence, and root synthesis cost bound concurrency; do not add a lane for coverage or confidence alone. Nested delegation is not a default and needs its own independent evidence question. If a lane exposes a new owner decision, return it to the root rather than expanding scope.

For read-only subagents, choose the currently available model and reasoning effort from task difficulty, evidence volume, latency/cost, and consequence of error, using the [harness model map](../../agent-harness.md#model-and-effort-selection). Re-review should be at least as capable as the review that found the issue. The implementation phase separately owns native Worker dispatch and execution.

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

## Review Finding Envelope

Lead with surviving findings in severity order. Each actionable finding names
its anchor, impact on the accepted outcome, blocker/concern/non-blocking
classification, and smallest action or reopen owner. If no finding survives,
say so and state the evidence boundary; do not pad a clean review.

## Review Independence

Use an independent reviewer when an artifact controls an orchestrated, high-impact, hard-to-reverse, protected-domain, cross-owner, or materially contested decision that its author cannot credibly falsify alone. Artifact presence alone does not trigger a reviewer. Other structured artifacts use root self-review. An explicitly requested independent review of completed implementation is a separate read-only boundary after implementation, not an internal gate.

The reviewer:

- reads the current fixed artifact or diff;
- reports anchored findings and a verdict recommendation;
- does not edit or approve its own repair;
- distinguishes blockers, bounded concerns/proof obligations, and non-blocking observations;
- states the evidence boundary.

When independent review is triggered, one whole-artifact reviewer is the default. Run a specialist first only for one concrete high-impact question the root cannot credibly cover locally. If the reviewer discovers such a question, run one bounded specialist follow-up and return only that disposition; do not repeat an unchanged whole candidate. A domain label or desire for more confidence does not justify another reviewer.

For each triggered non-implementation boundary, review convergence is: fixed candidate -> independent findings and verdict -> disposition. Repair and focused fresh review apply only to `FAIL` or to a material candidate change made to resolve a concern. Reuse unaffected findings. Implementation findings instead follow the root-owned Worker correction loop in the implementation phase.

A triggered review moves on `PASS`. `CONCERNS` also moves once each concern has a downstream proof or risk owner, observable, and reopen condition and leaves no behavior or mechanism for implementation to invent; this disposition does not require another review. A concern that cannot be carried that way is `FAIL`. `FAIL` blocks movement until repair or upstream reopen and focused fresh review. Do not repeat an unchanged candidate or launch speculative lenses merely to collect confidence.

A material mutation after review invalidates only the affected findings and proof. Wording-only edits and recorded concern dispositions do not trigger another review.

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

Emit a handoff prompt only when the current actor or session cannot continue. Name the outcome, minimal read set, non-obvious constraints, current evidence, next action, and stop/reopen owner; otherwise return the phase result without manufacturing another session.

A risk-triggered research-synthesis challenge and triggered specification, technical-design, test-design, and task-readiness reviews are internal checkpoints. The root launches the applicable read-only lane and continues automatically in the same session when possible.

An explicitly user-requested standalone review remains read-only: return the
complete review result and stop at the requested review boundary. It gains no
repair, implementation, or workflow-handoff authority unless the user separately
grants it.

When that request independently reviews completed implementation, it begins only
after implementation/validation/closeout has ended and never retroactively becomes
an implementation acceptance or closeout gate.
