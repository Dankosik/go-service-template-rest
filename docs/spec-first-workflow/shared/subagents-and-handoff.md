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

Do not create a built-in subagent lane for work that is sequential, tightly
coupled to the root's reasoning, duplicates another lane, lacks an independently
checkable evidence boundary, or would reduce the coherence of the final
synthesis. Do not create a session handoff when the current session can safely
finish.

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
review of a fixed non-implementation revision, and a triggered [independent
implementation review](#implementation-review-independence) of a fixed
acceptance unit.
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

## Review Independence

Use an independent reviewer when an artifact controls an orchestrated,
high-impact, hard-to-reverse, protected-domain, cross-owner, or materially
contested decision that its author cannot credibly falsify alone. Artifact
or `tasks.md` presence alone does not trigger a reviewer. Other structured
artifacts and ordinary implementation use root self-review. Apply the same
trigger to a fixed implementation acceptance unit; an explicit user request for
independent review also triggers it.

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

## Implementation Review Independence

Each triggered implementation review uses one fresh, one-shot read-only lane
against one fixed acceptance unit. [Agent
Harness](../../agent-harness.md#read-only-lanes) chooses the ordinary or
critical role and harness-native clean-context mechanism. The implementation
actor and implementation Worker are not eligible reviewers, and a lane used for
one unit is not resumed for another task ID or unit.

Give the lane the `tasks.md` path and grouped unit IDs or singleton task ID,
cited accepted sources,
authoritative candidate location, and irreproducible external evidence. The
reviewer derives its result from the current contract, candidate, production
path, dependencies, retained scope, and claim-scoped proof rather than an
implementation summary. It may run safe non-mutating checks.

The reviewer keeps the candidate fixed, edits and repairs nothing, and returns
the [phase-defined verdict and
evidence](../phases/implementation-validation-closeout.md#independent-implementation-review)
unchanged. A material candidate or proof-precondition change invalidates that
verdict; the root routes correction and opens a fresh lane only when the review
trigger still applies. The reviewer uses existing proof receipts and runs only
the missing or adversarial falsifier required by its question.

## Fan-In

For each material lane result, keep only:

- conclusion and strongest evidence;
- uncertainty or conflict;
- consuming disposition: accept, reject, repair, carry as proof/risk, or reopen;
- destination artifact or owner.

Do not paste raw transcripts into authoritative artifacts.

## Resume

Resume from artifacts, not remembered chat:

1. `tasks.md` for implementation/validation;
2. otherwise `workflow-plan.md` for multi-session coordination;
3. then only the spec/design/test/research/rollout artifacts named by the current next action.

If those sources disagree, the task is blocked until the narrowest owner reconciles them.

When compaction or accumulated lane history makes completed coordination larger
than the live decision state, refresh the ledger's compact `Active wave` block
and continue from that artifact in a fresh root context when the harness
supports it. Carry no transcript replay; retain only accepted inputs, unit and
candidate identities, proof receipts, open causal class, and next action.

## Handoff

Treat handoff as a chain of custody: name the accepted source, movement evidence, next owner, authority boundary, proof obligation, next executable action, and exact stop/reopen owner and condition so the receiver can continue without reconstructing chat.

Emit a handoff prompt only when the current actor or session cannot continue. Name the outcome, minimal read set, non-obvious constraints, current evidence, next action, and stop/reopen owner; otherwise return the phase result without manufacturing another session.

A risk-triggered research-synthesis challenge and triggered specification, technical-design, test-design, and task-readiness reviews are internal checkpoints. The root launches the applicable read-only lane and continues automatically in the same session when possible.

An explicitly user-requested standalone review remains read-only: return the
complete review result and stop at the requested review boundary. It gains no
repair, implementation, or workflow-handoff authority unless the user separately
grants it.

An explicitly requested review of completed implementation may run inside
implementation when the request makes it an acceptance condition; otherwise it
begins after implementation/validation/closeout and never retroactively becomes
an acceptance or closeout gate.
