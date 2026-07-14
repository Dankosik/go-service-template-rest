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

Skills define method; subagents provide separate context and independence. At the start of each active macro phase, identify materially affected domains and apply matching skills locally by default. Delegate only one concrete bounded question when independent evidence, separate context, parallelism, or review independence can change the result. A domain name, matching keyword, skill handoff, or desire for more confidence alone never creates a lane.

If no separate lane helps, keep the work local and state the reason only in an existing phase artifact or handoff; do not create a standalone gate record. Required independent reviews still use a separate read-only reviewer.

Use a subagent when all are true:

- the question is concrete and bounded;
- it can be answered independently of mutable work in other lanes;
- separate context or review independence materially improves the result;
- the output can be checked and synthesized by the root.

Good lanes include independent source research, one specialist design question, or review of a fixed revision. Bad research/review lanes include broad “review everything,” duplicate lenses, and tiny lookups. Dependency-ordered implementation tasks use the worker loop below rather than being treated as evidence or review lanes.

The root owns scope, lane choice, synthesis, integration, task acceptance, and completion claims. Research and review lanes are read-only. Implementation workers may write only inside their assigned task boundary; use isolation when risky experiments justify it, not by default.

Default to at most three concurrent research/review lanes, one bounded wave, and no nested delegation. Additional sequential research/review lanes still require distinct decision-changing questions that could not be covered locally or in the first wave. The one-at-a-time implementation worker loop is separate from that limit. If a lane exposes a new owner decision, return it to the root rather than expanding scope.

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

## Implementation Worker Loop

When an accepted ledger exists, the root selects one ready task in dependency order and assigns only that task to one implementation worker. That worker remains the task owner across corrections until the root accepts the integrated task diff and proof or a genuine upstream blocker reopens another owner.

The worker returns the exact diff, acceptance-criteria mapping, commands/results, and blockers. It does not mark the task complete, start another task, spawn a reviewer, or approve its own work. The root inspects that return against the task boundary and accepted proof. If anything required is missing, the root returns concrete gaps to the same worker and does not advance the ledger or repair the task itself. If the task is accepted, the root records its checkbox and evidence, then assigns the next ready task to a fresh worker. Reuse the previous worker only for corrections to its own task.

This root acceptance is the required review of worker output; it is orchestration, not a separate review lane. A final independent reviewer is optional and risk-triggered under Review Independence below, and never substitutes for accepting each task before advancing.

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

Structured and orchestrated work requires an independent reviewer for a standalone `research only` synthesis, the completed specification, any triggered technical design or test design, and the completed implementation ledger. During implementation, the root instead accepts every worker task before advancing. The final integrated diff adds an independent reviewer only when the user explicitly requests it or a concrete trigger makes the integrated change broad, hard to reverse, materially high-impact, ambiguous, cross-task-sensitive, protected-domain-sensitive, or poorly falsified by the root alone. Small direct work uses root diff inspection and bounded validation under the same trigger rule. Code change, Goal closeout, skill availability, domain keywords, and generic confidence are not review triggers. A required gate reviewer is any separate read-only subagent following the phase review method; named specialist profiles add depth but do not own the root's final gate decision.

The reviewer:

- reads a fixed artifact revision or diff;
- reports anchored findings and a verdict recommendation;
- does not edit or approve its own repair;
- distinguishes blockers, bounded concerns/proof obligations, and non-blocking observations;
- states the evidence boundary and disposition of every materially affected lens: covered, delegated to a named specialist result, or not triggered with a concrete reason.

When an independent gate is required, one whole-artifact or whole-diff reviewer is the default. That reviewer applies compatible matching methods locally in one coherence pass and accounts for every materially affected lens. Run a specialist before that gate only for one concrete high-impact question the root cannot credibly cover locally. If the gate reviewer discovers such an uncovered question, run one bounded specialist follow-up and return only that disposition for a focused verdict update; do not repeat an unchanged whole revision. A domain label, generic handoff, or desire for more confidence does not justify another reviewer.

The root collects all in-scope actionable findings for the fixed revision. For worker-owned implementation, return the findings to the worker that owns the affected task; that worker repairs them as one coherent batch where dependencies allow. The root repairs only direct work and root-owned artifacts, then reruns affected proof. The same gate reviewer performs focused re-review of the repair and any transitively affected behavior, contract, ownership, or proof decision. Reuse unaffected lens dispositions by default; use full affected-surface re-review only when the repair changes shared assumptions or crosses a domain boundary. Re-review must be at least as capable as the review that found the issue.

A macro phase reaches review convergence only when its latest required review returns `PASS` and finds no blocker, known current-phase defect, unowned question, uncovered materially affected lens, or unresolved cross-lens contradiction. `CONCERNS` is non-terminal: a bounded risk or downstream proof obligation still needs disposition in the owning phase, and it never permits phase movement or closeout. The owning author repairs it; for worker-owned implementation, the root returns it to that worker. The root records authorized acceptance with evidence, owner, and reopen condition, or splits/reopens scope, then obtains focused fresh review; `PASS` means every concern has a disposition, not that no residual risk exists. `FAIL` blocks movement and requires repair or reopening before focused fresh review. Non-blocking observations do not prevent `PASS`. Repeat only while a concrete new finding or semantic repair changes readiness. If closure needs unavailable evidence, authority, or an upstream decision, mark the phase blocked and reopen that owner. Do not repeat an unchanged `PASS` revision or launch speculative lenses merely to collect confidence.

A semantic mutation after review to the reviewed artifact, implementation diff, generated outputs, tests, or proof evidence invalidates convergence only for affected lenses. Finalize ledger checkboxes and evidence before review; chat, Goal, and other status-only closeout updates after `PASS` are outside the candidate and do not invalidate it. Revalidate affected proof and obtain focused fresh review before closeout.

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

Required research-synthesis challenge, specification review, technical-design review, test-design QA review, task-readiness review, post-code review, in-scope repair, fresh re-review, validation, and closeout are internal checkpoints of their owning macro phase. The root launches the required read-only lane, repairs authoritative work, obtains any required fresh verdict, and continues automatically in the same session. A durable review record is only a carrier; it does not create a user-started phase or a next-session prompt.

An explicitly user-requested standalone review remains read-only: return the
complete review result and stop at the requested review boundary. It gains no
repair, implementation, or workflow-handoff authority unless the user separately
grants it.

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
