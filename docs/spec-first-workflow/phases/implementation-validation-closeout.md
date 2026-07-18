# Implementation / Validation / Closeout

On entering this macro phase, and only then, the root establishes the single Codex Goal required by `AGENTS.md`, starts the native Codex App Worker task that produces the requested change, reviews and integrates the returned diff, and proves the accepted outcome.

## Read When

- The request authorizes change/build/fix and required decisions are ready.
- Direct work has a clear inline outcome and proof, or structured/orchestrated work has a ready independently reviewed ledger.
- Existing implementation needs repair, review, validation, or closeout.

## Inputs

- Accepted direct outcome or current reviewed `tasks.md`.
- Required spec, design, test, and rollout decisions named by the work.
- Current repository state, including pre-existing user changes.
- Repository-owned generation and validation commands.

## Outputs

- In-scope code, tests, config, migrations, generated output, and docs.
- Updated task progress when a ledger exists.
- Review findings and repairs proportionate to risk.
- Fresh validation evidence and an evidence-clamped final claim.

### Acceptance Loop

These labels summarize the existing sections below; they are not new workflow states, statuses, artifacts, receipts, or checkpoints.

- **Assignment gate:** the single active root Codex Goal, accepted input, App task and managed base, model and effort, constraints, and success criteria are explicit.
- **Candidate gate:** one bounded Worker diff returns criteria disposition, proof, and every unmet item.
- **Acceptance gate:** the root independently accepts only after every evidence-backed gap is closed; otherwise it returns the complete compatible finding set to the owning Worker.
- **Wave gate:** when ledger work has a wave, it is the acceptance unit; every member passes on the frozen combined candidate, then atomic promotion commits the accepted delta to the authoritative integration branch before any later task is dispatched, and every later Worker starts from that resulting commit/ref.
- **Validation gate:** terminal fresh evidence proves the unchanged integrated frozen candidate.
- **Claim gate:** the final claim is no broader than the accepted outcome and terminal fresh evidence.

## Implement

Implement surgically: make the smallest complete change at the earliest valid owner, including required proof and cleanup, without touching unrelated surfaces.

Before every production Go edit, apply `go-coder`. Route an unknown-cause
defect first to `go-systematic-debugging`, and test-only implementation to
`go-test-implementation`. The selected skill reconstructs the Go Change Surface
and loads only the references matching its triggered pressures.

1. Inspect the owning code, callers, siblings, tests, and generated/manual boundary before editing.
2. For a defect, fix the narrowest owning surface whose contract the reproducer proves is violated; do not patch only the reported entry point when sibling callers share that contract.
3. Preserve accepted behavior, ownership, failure, cleanup, rollout, and proof decisions.
4. Prefer stdlib and existing repository patterns. Do not add a dependency, interface, helper layer, or architectural pattern without a present need and an owner.
5. Remove replaced paths and adjacent stale artifacts unless current compatibility evidence requires retention.
6. Keep changes reviewable and avoid unrelated cleanup.
7. If implementation exposes a missing product, contract, source-of-truth, ownership, test-strategy, or rollout decision, stop and reopen that owner instead of inventing it.

### Worker Assignment And Acceptance

#### Shared Execution Mindset

The root and Worker jointly own the shortest evidence-backed path from the assigned outcome to root acceptance, not a low turn count or first-pass appearance. Correctness, maintainability, and adequate proof are non-negotiable. Remove coordination only when it cannot improve the candidate, decision, or proof. Review and correction are means, not deliverables. Each correction has terminal intent: close every currently known compatible gap and return a candidate intended to be accepted, not another partial role handoff. The acceptance frontier must shrink: every correction closes known compatible gaps or materially changes the supported causal model or recovery route; an unchanged frontier does not justify another identical loop.

Every authorized implementation change is produced by one native Codex App task for the repository project in a managed-worktree environment backed by a dedicated Codex-managed Git worktree ([official Worktrees](https://learn.chatgpt.com/docs/environments/git-worktrees)). Direct work assigns one accepted outcome to one Worker. Ledger work assigns one ready task to one Worker and normally dispatches the next planned wave concurrently. At most one write Worker is active for a task; separate positively independent tasks may run in parallel, each in its own worktree, until root acceptance, a genuine upstream blocker, or the evidence-backed replacement decision below.

Treat the reviewed planned wave as the default route, not a reason to repeat planning. Before dispatch, perform one lightweight check only on facts that may have changed: declared dependencies are accepted, every task starts from the same current integrated base, writable and exclusive-resource boundaries remain disjoint, and no new interface, invariant, generated-source, migration, rollout, or proof coupling invalidates the wave basis. Dispatch the safe ready members up to current App capacity. Defer or serialize a member when a check is false or uncertain, preserve unaffected members, and record the adjusted wave in transient execution context or the existing ledger. Do not create another schedule artifact or reopen planning solely because current capacity, completion order, drift, or a discovered implementation overlap changes a wave; reopen the smallest upstream owner only when implementation would otherwise invent or change an accepted behavior, source of truth, ownership, test strategy, or rollout decision.

For every dispatched task, the root selects the smallest starting state that contains the accepted implementation input: omit the optional starting state when the project default already contains it; select an existing branch when that branch owns the accepted state; select the working tree only when required accepted changes are uncommitted. Before creating a new App task with `startingState: working-tree`, run `bash scripts/dev/codex-worktree-preflight.sh <selected-git-top-level>` against that selected top level. It is a read-only conservative compatibility guard: it fails closed at 32 MiB of tracked patch input or 64 MiB of total working-tree transfer input (tracked patch plus nonignored untracked input), reports the measured offending inputs, and does not create the App task. A nonzero result means use an existing commit/ref that owns the accepted input or report the exact blocker; do not omit, commit, stash, clean, ignore, or mutate user changes to pass it. Do not run this preflight for an ordinary same-task correction: continue in its existing App task and managed worktree. These are current conservative guards, not an eternal App product contract. Every member of one wave starts from the same accepted integrated base. Dispatch the wave as soon as the Goal, task briefs, and lightweight check are ready.

For every new App Worker task, the root explicitly selects and passes both the model and reasoning effort through the App's supported launch controls; never inherit an App default. Record the task identity, selected model and effort, and a short basis in transient execution context or existing ledger evidence. When the selection relies on eval evidence, the basis names the exact eval artifact and compared model/effort configuration.

Choose model and effort independently from the App's supported choices, using task difficulty, ambiguity or evidence volume, latency/cost, reversibility, and consequence of error. Per the [official Codex model guide](https://learn.chatgpt.com/docs/models#choosing-sol-terra-and-luna), select `gpt-5.6-sol` for complex, open-ended, ambiguous, difficult, or high-value coding, research, or security work needing extra analysis, judgment, or polish; `gpt-5.6-terra` for pragmatic everyday work needing strong reasoning and tool use with a capability/cost balance; and `gpt-5.6-luna` for clear, specific, repeatable, high-volume work with a known result, such as extraction, classification, transformation, or structured summaries. These are task-specific guides, not mandatory keyword routing or a Terra default. Use the lowest effort likely to produce the required result: `low` for quick, well-scoped work; `medium` for more planning and a speed/depth balance; `high` or `xhigh` for difficult work with multiple steps, sources, or tradeoffs; and `max` for the hardest single quality-first task. `Ultra` is subagent parallelism, not more single-task reasoning; this Worker cannot delegate, so do not route this single-Worker phase to Ultra. There is no fixed model/effort baseline. Security, concurrency, data, migration, and cross-service scope are risk signals, not automatic Sol or highest-effort triggers.

Evals may inform the basis when already available, but are never a prerequisite: do not pause dispatch or create/run an eval solely to justify a model or effort choice. Same-task model and effort may change when remaining work or observed Worker evidence justifies it, without an eval prerequisite. A composer selection changed after dispatch configures a future turn; it is not evidence of the active turn's effective model or effort. When that distinction matters, use only native metadata or events for the active turn.

After dispatch, follow the App's [event stream](https://developers.openai.com/codex/app-server#events): `turn/started` confirms the active turn, `item/*` carries progress and authoritative completed items, `turn/completed` supplies the terminal turn status, and `thread/status/changed` reports runtime status transitions. The root does not actively poll or narrate unchanged state. It resumes result intake, acceptance, and Git handoff/integration when the native completion or status signal arrives.

A Worker reported `inProgress` with no `turn/started` and no `item/*` progress is a recoverable execution fault, not a user blocker. When an event-driven status inspection establishes that state and one continuation request produces no new turn or item, the root stops the stale App task before any replacement, verifies and preserves its managed-worktree candidate state, records the missing progress and exact recovery base in the cumulative evidence frontier, then launches one replacement Worker for the same outcome through the existing replacement route. Do not ask the user to stop, restart, or authorize this recovery; keep the root Goal active unless no safe in-scope recovery route remains.

An ordinary implementation-owned gap returns to the same App task and managed worktree with concrete findings; the root never authors the repair. The Worker validates the finding against the accepted outcome and current evidence, fixes the violated contract at its narrowest owner, checks sibling and transitive effects, and returns a candidate intended to close the complete finding set. A root-suggested patch is a hypothesis unless an accepted decision fixes the method; the Worker may return concrete counter-evidence and a better owning route instead of mechanically applying a wrong suggestion. The root judges the result and evidence, not obedience to its proposed patch. When material findings recur from one causal class, stop issuing symptom-by-symptom patches: name the violated invariant and owning surface, group the repeated evidence, and require the next correction to use a materially different recovery route. Continue the same Worker unless the replacement rule below is also met. Tightly scoped Worker checks and criteria mapping are task-local feedback, not acceptance: the root independently judges behavior, quality, test adequacy, scope, and completeness from the report, evidence, and diff.

Do not replace the App task merely because a same-task correction remains. When the root detects evidence-backed no progress, including the stalled-Worker condition above, a repeated material correction state, oscillation, an exhausted repair hypothesis, or an invalidated worktree base after fan-in, it preserves the cumulative evidence frontier and launches a fresh replacement native App Worker for the same direct outcome or ledger task through a materially different recovery route. The replacement brief names the exact candidate state; concrete open, closed, and reopened findings; failing proof observables; attempted and falsified repair hypotheses where applicable; affected constraints and lenses; success criteria; and the different route or updated integrated base. Keep only one write Worker active for that task, retain the full replacement history, and require root acceptance. Change model or effort only when observed failure evidence justifies it.

Wave results may return in any order. Individual results are provisional; root acceptance applies only to the atomic wave's frozen combined candidate. The root reviews each bounded result, then assembles provisionally suitable deltas in a disposable wave candidate, in a controlled order, without adding unaccepted work to the authoritative integration state. No wave task reaches root acceptance while another member is unresolved. On the frozen combined candidate, map claims to their exact commands and run each identical command once; one result may prove multiple mapped claims, but commands with different arguments, environment, state preconditions, or required observables are not identical. After every member passes root review and its proof on the combined candidate, promote only the bounded accepted wave delta by committing it to one authoritative integration branch and record every task's evidence. Commit before dispatching any later task; every later App task starts from that resulting commit/ref, never an accepted uncommitted working-tree snapshot. If the repository default/main is explicitly that authoritative integration branch and the accepted wave leaves it valid, fast-forward it after acceptance. If a conflict, invalidated assumption, or combined-proof failure appears, hold the whole wave, preserve unaffected results as provisional, and repair the affected task against the current wave candidate through its owning App task when that worktree can be updated safely or through same-task replacement recovery otherwise. Reassemble and re-prove the whole wave. Reopen an upstream owner only for a genuinely missing or changed decision, not for the execution schedule or an implementation-owned integration repair.

After atomically accepting and integrating the adjusted current wave, advance to the next planned or adjusted wave and recheck only seams affected by the integrated changes. Use a fresh App task in a fresh managed worktree for each later task; never reuse an accepted task's Worker for another task.

Handoff to Local is optional. Dirty Local or a failed Handoff is an internal integration problem, not a reason to ask the user or block the root Goal when a safe in-scope route exists. Do not stash or commit unrelated user changes. Continue from the Worker worktree, a task-owned branch, or a clean integration worktree and branch; integrate only the bounded accepted Worker delta, validate that integrated state, commit it to the authoritative integration branch, record ledger evidence when applicable, and dispatch the next safe planned wave from that commit/ref.

### Worker Brief And Result Intake

Before editing, the Worker verifies that its physical current directory and Git top level are the assigned managed worktree. It treats that worktree as the only writable repository checkout. A source-checkout absolute path maps to the same repository-relative path in the managed worktree; an unmappable required path blocks instead of authorizing a write outside the worktree.

Keep the brief outcome-first and limited to:

```text
Outcome / context: <accepted direct outcome or one ready task; minimal authorities and paths>
Constraints: <task-specific editable, forbidden, permission, and role boundaries>
Evidence: <current facts, required sources or skills, and proving commands>
Success: <observable acceptance criteria>
Output: <changed/deleted files; each finding closed, disproven, or genuinely blocked with evidence; exact proof results; unmet criteria>
Stop: <genuine blocker or authority condition>
```

The constraints tell the Worker that it is not root and cannot create, continue, or complete a Goal; delegate; self-accept; update task or workflow status; start another task; or claim repository completion. Do not copy the workflow, App lifecycle, generic strictness language, or unrelated repository context into the brief.

For a correction, reuse this same brief instead of creating a correction artifact or acknowledgement turn. Make it executable as written: carry the complete set of currently known compatible findings, causal groups and violated owners established by current evidence, exact observables, required end state, affected scope, and proof to rerun; distinguish an accepted constraint from a suggested repair hypothesis. If counter-evidence disproves one finding or suggested patch, close it with that evidence and continue every remaining valid finding in the same turn. Ask only when the unresolved choice changes accepted behavior, authority, or safety.

Before returning, the Worker performs one compact self-check: reread the assigned outcome and constraints, inspect its own bounded diff for missing criteria, out-of-scope edits, and stale replaced artifacts, run the listed focused proof, and report every unmet criterion. This self-check is Worker evidence, never root acceptance.

The root records the returned App task, thread, and managed-worktree identity, assigned outcome or task, corrections, replacements and their cumulative evidence frontier, report, diff, proof, acceptance, and integration state in transient execution context or existing ledger evidence, never a second ledger. When compaction, interruption, or session handoff could lose an active wave, persist the compact `Active wave` state defined by the artifact model in `tasks.md`; update it only at material transitions and collapse it into task evidence after atomic acceptance. Replacement history does not reset. The Worker return is evidence, not authority. The root still reads the report, diff, and proof and decides acceptance; blocked reports, unmet criteria, failed operations, or ambiguous task identity cannot be accepted by prose self-identification.

Rerun relevant proof in the integration workspace when Worker evidence does not establish the integrated state. Do not rerun an unchanged command without a changed risk surface. Before an intentional stop, record the blocker and next executable task and persist active-wave state when applicable; on resume, follow the shared [Resume Order](../shared/artifact-model.md#resume-order) instead of reconstructing progress from chat.

## Review

Always inspect the final diff for:

- correctness against accepted behavior and invariants;
- error, cancellation, retry, concurrency, transaction, and resource-lifetime behavior where relevant;
- contract/generated-source drift;
- security, privacy, money, data, and rollout risk in scope;
- ownership, unnecessary abstraction/dependencies, and stale replacement surfaces;
- tests that prove behavior rather than implementation detail.

The root reviews every App Worker result and its proof before acceptance. Its first acceptance inspection is one coherent pass over the bounded outcome, all acceptance criteria, the proof, and every materially affected lens. Only evidence-backed acceptance gaps are correction findings; a style preference, equally valid implementation choice, or unproven suspicion is not. The root resolves repository-answerable uncertainty before briefing the Worker, then returns all currently detectable compatible findings in one evidence-backed batch rather than drip-feeding one finding per turn. Apply every matching review skill locally in that same inspection. A review skill supplies a method, not a subagent lane. Never launch a built-in subagent, reviewer, specialist, or re-review lane anywhere inside implementation/validation/closeout.

For changed Go, classify each changed symbol only across triggered package,
export, or method-set; context or error; resource or transaction; mutable
ownership; concurrency or lifecycle; generated-authority; and proof pressures.
Apply one matching review method per independent pressure; untriggered
categories create no work.

Every ledger task receives root acceptance review on the combined wave candidate, and no wave member is accepted while another remains unresolved. Dependent or conflicting tasks wait for atomic wave acceptance; independent members of one planned wave need not wait for one another to run. After every task is accepted, generated outputs and task evidence are current, and terminal validation is complete, the root reviews the final integrated diff against the accepted outcome and every affected lens.

Return every implementation-owned finding to the App task that owns the affected direct outcome or ledger task. Continue that task in its managed worktree and rerun affected proof. Re-review is delta-aware: verify the complete correction set, the correction delta, its proof, and every transitively invalidated lens; do not repeat inspection of unchanged surfaces without a changed risk signal. When those checks pass, accept the task without another review lane or ceremonial pass.

Classify a material finding first discovered after correction as a previously detectable omission, correction-introduced regression, recurrence of the same causal class, genuinely new affected surface, or upstream decision gap. Batch any remaining compatible findings, preserve the cumulative evidence frontier, and route only the affected class. The open frontier must shrink or the causal model or recovery route must materially change; never repeat an unchanged correction loop or impose a fixed correction-count limit. A task-local finding does not cancel unaffected wave members; stop additional members only when the finding invalidates their shared wave basis. Reopen upstream decisions rather than broadening implementation. A passing command is insufficient when its implemented fixtures or assertions do not exercise the binding proof obligation; a matching selector name alone is not evidence.

Treat edits to tests, fixtures, golden files, skip or exclusion settings, lint/build configuration, and proof commands as proof-surface changes. They require an accepted task or behavior reason; a green result obtained by weakening or removing an oracle or bypassing a triggered gate is invalid.

Validation, in-scope Worker repair, root re-inspection, revalidation, and closeout run automatically in the same root session. An implementation-owned failure never produces a next-session prompt.

## Validate

Run focused proof while implementing, then one terminal fresh evidence set for the frozen candidate. Map terminal validation commands to claims and run them on one unchanged integrated state. Do not rerun an unchanged command unless a new patch, finding, or required final bundle changes what it proves. The terminal set covers the claim with:

Apply `go-verification-before-completion` to terminal claims. Include race,
integration, generation, contract, sqlc, or migration proof only when triggered.

- targeted tests for changed behavior;
- build, type, lint, race, integration, or repository gates relevant to affected packages;
- contract, migration, generation, or mirror drift checks when their source changes;
- integrated target-environment proof across the affected deployment graph when the accepted outcome is system-wide; provider deployment status or one component's readiness alone is insufficient;
- a smoke/manual check when automated proof is unavailable or insufficient;
- targeted negative searches for identifiers and references that should be gone.

Worker output, cached results, unrelated green checks, skipped commands, and too-narrow tests do not prove the claim. When a required check cannot run, record the command, reason, narrower evidence, and unverified remainder.

Reconcile both directions: every accepted obligation and every ledger task on the current completion path maps to its implementation or an already accepted evidence-backed no-implementation disposition, and to adequate proof; every material change maps back to accepted scope. Keep this reconciliation inline unless an existing ledger owns it. Preserve unrelated pre-existing changes.

## Close Out

Mark a task complete only after its proof passes. The final response states:

- what changed;
- the most important design/behavior consequence;
- validation actually run and result;
- remaining risk, unavailable proof, or blocker;
- the exact reopen owner when unfinished.

Clamp completion language to terminal fresh evidence: use `complete`, `fixed`, `ready`, or equivalent only when that evidence supports the full claim. A blocker is a valid outcome, not successful completion.

## Stop Rule

Finish when the direct outcome or every ledger task has passed root acceptance, the root has reviewed the final integrated diff and affected lenses, the accepted completion condition is met, and relevant proof passes. Return ordinary implementation-owned gaps to their owning App task and use replacement recovery when its evidence-backed decision rule is met. Implementation non-convergence does not block the root Goal while a safe in-scope recovery route exists. A task or phase blocker reopens its owner without terminating the root Goal while another safe in-scope action can advance the accepted outcome. Stop and reopen planning, test design, technical design, specification, research, or user/external authority only when that owner must change a decision or supply unavailable evidence; mark the root Goal blocked only when no safe in-scope route remains.
