# Subagents And Review

Use built-in subagents only when they reduce a real context, independence, or
coordination problem.

## Read When

- Distinguishing a built-in subagent lane from the harness-native implementation Worker.
- Requiring an independent verdict.

## Inputs

- One accepted goal or one concrete review question.
- The smallest artifact and source bundle needed to answer it.
- Clear read-only and external-action boundaries.

## Outputs

- A local decision or a small lane plan.
- Evidence and recommendations for root synthesis.

## Stop Rule

Do not create a built-in subagent lane for work that is sequential, tightly
coupled to the root's reasoning, duplicates another lane, lacks an independently
checkable evidence boundary, or would reduce the coherence of the final
synthesis.

## Delegation Decision

**Fan-out.** Skills define method; subagents provide separate context and
independence. At the start of each non-implementation macro phase, map the
current decision-changing questions and their dependencies before substantive
work. A question is lane-eligible when all are true:

- it is concrete and bounded and can change a named decision, criterion, or
  downstream disposition;
- it can be answered independently of mutable work and dependent reasoning in
  other lanes;
- it has a checkable evidence boundary;
- separate specialist, clean context, independent evidence, or review
  independence can improve coverage, falsification, or coherence;
- the output can be checked and synthesized by the root.

Dispatch every lane-eligible question to one read-only lane when the current
harness exposes a native carrier and capacity. Run positively independent lanes
concurrently. Research and Technical Design use fan-out as their default
execution shape; other non-implementation phases apply the same rule to
eligible discovery, challenge, and review questions. Intake keeps user-intent
and authorization decisions in the root and routes decision-changing evidence
questions to Research.

Keep in the root any ordered chain whose next decision depends on the previous
result, cross-lane synthesis, final artifact decisions, correction routing, and
all acceptance and completion claims. If no question is lane-eligible or the
native carrier is unavailable, continue root-locally and record the quality,
independence, or coherence reason only in an existing phase artifact or handoff;
do not create a standalone gate record. Cost, token use, latency, task size, and
local convenience do not justify skipping an eligible lane. Required
independent reviews still use a separate read-only reviewer.

Good lanes include independent source research, one specialist design question,
and review of a fixed non-implementation revision. Triggered implementation
review follows its separate [conditional branch](implementation-review.md).
Bad lanes include broad “review everything,” duplicate lenses, questions
without an independent evidence boundary, discretionary implementation review,
implementation specialist analysis, and any implementation or review repair.

The root owns scope, lane choice, synthesis, correction routing, integration,
acceptance, completion claims, and mechanical ledger updates. Built-in
subagents are read-only research, challenge, or review lanes; they never
implement or repair code, config, docs, or tests.
In the Codex App a lane is a project subagent; in Claude Code it is an `Agent`
tool lane ([Agent Harness](../../agent-harness.md#control-map)). The
harness-native implementation Worker is outside this contract and follows the
[implementation
phase](../phases/implementation-validation-closeout.md#worker-execution).

Run one lane per distinct decision-changing question. Current harness capacity,
mutable-state independence, and synthesis coherence bound concurrency; do not
add a duplicate lane for confidence alone. Nested delegation is not a default
and needs its own independent evidence question. If a lane exposes a new owner
decision, return it to the root rather than expanding scope.

For read-only subagents, choose the currently available model and reasoning
effort from task difficulty, evidence volume, and consequence of error, using
the [harness model map](../../agent-harness.md#model-and-effort-selection).
Never choose a weaker lane to reduce cost, token use, or latency; a lower tier
is valid only when current task evidence or representative evaluation shows no
material quality loss. Re-review should be at least as capable as the review
that found the issue. The implementation phase separately owns native Worker
dispatch and execution.

## Lane Brief

One lane owns one question, one evidence boundary, and one consuming
disposition. Keep the brief outcome-first:

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

## Non-Implementation Review Convergence

A triggered non-implementation reviewer:

- reads the current fixed artifact or diff;
- reports anchored findings and a verdict recommendation;
- does not edit or approve its own repair;
- distinguishes blockers, bounded concerns/proof obligations, and non-blocking observations;
- states the evidence boundary.

When independent review is triggered, one whole-artifact reviewer is the default. Run a specialist first only for one concrete high-impact question the root cannot credibly cover locally. If the reviewer discovers such a question, run one bounded specialist follow-up and return only that disposition; do not repeat an unchanged whole candidate. A domain label or desire for more confidence does not justify another reviewer.

For each triggered non-implementation boundary, review convergence is: fixed
candidate -> independent findings and verdict -> disposition. Repair and
focused fresh review apply only to `FAIL` or to a material candidate change
made to resolve a concern. Reuse unaffected findings.

A triggered review moves on `PASS`. `CONCERNS` also moves once each concern has a downstream proof or risk owner, observable, and reopen condition and leaves no behavior or mechanism for implementation to invent; this disposition does not require another review. A concern that cannot be carried that way is `FAIL`. `FAIL` blocks movement until repair or upstream reopen and focused fresh review. Do not repeat an unchanged candidate or launch speculative lenses merely to collect confidence.

A material mutation after review invalidates only the affected findings and proof. Wording-only edits and recorded concern dispositions do not trigger another review.

## Fan-In

For each material lane result, keep only:

- conclusion and strongest evidence;
- uncertainty or conflict;
- consuming disposition: accept, reject, repair, carry as proof/risk, or reopen;
- destination artifact or owner.

Do not paste raw transcripts into authoritative artifacts.
