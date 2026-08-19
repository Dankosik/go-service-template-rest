# Resume And Macro-Phase Handoff

Resume from durable authority and hand off only across a real actor or
macro-phase boundary.

## Read When

- Resuming after compaction, interruption, or a different session.
- The current actor or session cannot continue and another owner must act.
- An orchestrated acceptance unit has a canonical agent-owned upstream reopen.
- An implementation session stops at an accepted unit boundary with later
  implementation work remaining.
- A ready macro phase has reached its user-started boundary.

## Resume

Apply the [Artifact Model Resume Order](artifact-model.md#resume-order). If the
authoritative sources disagree, reopen the narrowest owner instead of merging
the conflict from chat.

When completed coordination is larger than the live decision state, refresh
from the canonical ledger, native task status, and Git candidate identities in
a fresh root context when the harness supports it. Carry no transcript replay
or duplicate task-lifecycle record.

## Implementation Entry And Continuation Handoff

When the current actor or session hands a ready unit to a different session that
will enter or continue Implementation — after Planning movement, a persisted
acceptance-unit transition, or an explicitly requested different-session resume
— return a standalone `Next Session Prompt` without waiting for the user to ask.
Inspect the authoritative ledger and current checkout to select only the next
ready acceptance unit. Record its accepted ledger revision or prerequisite
receipt, accepted integration base, relevant pre-existing dirty paths plus their
owner, current external-effect authority or durable locator, the initiating
user’s verbatim native-control envelope, and exact unit ID as the handoff basis.
Use a commit or tree identity only when the candidate crosses a checkout or
integration boundary. The dispatching Ledger Orchestrator does not choose or
record the unit's carrier or internal lane map. Emit an Acceptance-Unit Lead
handoff when the envelope authorizes the available native controls; the Lead
classifies the surface, writes bounded small work directly, and attempts Workers
only when the Direct-Write Gate requires them.

When eligible, emit a copy-pastable prompt in this shape:

```text
Execution role: ACCEPTANCE_UNIT_LEAD
Role contract: docs/spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree

Lead <acceptance unit> through the assigned stage toward one canonical receipt
or blocker.

- Authority: <ledger and task paths; accepted revision or receipt; current external-effect envelope or durable locator>.
- Secret inputs: <durable locator or environment-variable names under AGENTS.md; none when no secret is required>.
- Native controls: <verbatim initiating authority for the Role Tree's adaptive Implementation Write Boundary, current-harness Worker controls when required, Slice DAG scheduler, frozen-base materialization, model/effort selection, review, recovery, direct fallback, and Handoff>. Goal use stays thread-local; this prompt expands no authority.
- Scope: <exact unit ID, accepted outcome and writable boundary; dependent work that remains blocked>.
- Dispatch scope: <ledger revision / unit ID / attempt>.
- Stage: <Local acceptance | Worktree candidate>.
- Checkout: <accepted base when crossing a checkout; relevant pre-existing dirt and owner, or none; exact user-named starting state, or omit startingState>.
- Proof: <accepted unit proof and completion condition>.
- Stop: <accepted behavior, unit scope, ledger dependency, external-effect authority, or another fixed authority that would have to change>.

Set the thread-local Goal for this stage and role, then execute the Role Tree's
Direct-Write Gate and, when required, its Execution Map. If a required Worker
carrier or base cannot be established after safe recovery, retain the same unit
and use the evidenced direct fallback; block only when candidate, authority,
scope, or proof cannot be preserved. A Worktree stage returns
`HANDOFF_READY` with its fixed candidate and no receipt; Local continues under a
separate Goal to the receipt or blocker. Preserve unrelated work and keep
dependent units blocked.
```

A Local Acceptance-Unit Lead returns only after it records the unit's canonical
`Accepted:` receipt or `Blocked:` record. A Worktree Lead may first return
`HANDOFF_READY` with a fixed candidate for native Handoff; an agent-owned
upstream boundary also carries its proposed blocker for Local revalidation. The
same task and role then continue in Local to the canonical receipt or blocker under [Agent
Harness](../../agent-harness.md#worktree-fan-in). A Ledger Orchestrator rereads
that ledger transition before selecting another ready dependent unit; it never
consumes or routes internal lane results.

### Worktree To Local Continuation

The Agent Harness passes this compact message atomically as native Handoff's
`followUpPrompt`; it is never a later standalone send:

```text
Execution role: ACCEPTANCE_UNIT_LEAD
Role contract: docs/spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree

Continue the same <unit ID> and <dispatch_scope>.
- Stage: Local acceptance.
- Candidate: <fixed commit/tree and Worktree identity>.
- Local precondition: <HEAD, status, and attributed dirt verified immediately before Handoff>.

Create a new thread-local Goal for Local acceptance. Integrate the fixed candidate, review, prove, diagnose findings, route every implementation correction to its owning Worker, and apply the Role Tree's bottom-up resolution ladder before recording the one canonical `Accepted:` receipt or `Blocked:` record. Do not rediscover the unit, repeat a route under unchanged preconditions, change role, or start another unit.
```

When `HANDOFF_READY` carries a proposed upstream blocker, replace the final
sentence above with:

```text
Create a new thread-local Goal for Local blocker revalidation. Preserve and inspect the fixed candidate, re-run the bottom-up resolution ladder against current Local artifacts, and either route a newly available implementation remedy through the Role Tree's Worker boundary or persist the one canonical `Blocked:` record with its reopen owner and condition. Start no upstream phase or another unit.
```

### Known Lead Terminalization

When one known Lead stops or requests attention without its canonical
transition, send this once to that same task with no model or effort override:

```text
Execution role: ACCEPTANCE_UNIT_LEAD
Role contract: docs/spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree

Terminalize the same <unit ID> and <dispatch_scope>. Re-read the canonical ledger, native task state, and Git candidate. If the accepted proof and candidate already close the unit, persist its one `Accepted:` receipt. Otherwise apply the Role Tree's bottom-up resolution ladder: take an evidence-changing unit-local remedy when one remains and route every implementation correction through its owning Worker; persist the exact `Blocked:` record and reopen owner only after none remains. Start no new unit and repeat no valid proof, review, or remedy under unchanged preconditions.
```

## Upstream Reopen And Implementation Return

When a canonical unit blocker names an agent-owned upstream phase, the Ledger
Orchestrator keeps the blocked Lead and candidate pinned and creates one fresh
Local task through the autonomous native dispatch protocol with the model and
effort selected by the Orchestrator from this phase's fixed brief.

```text
Execution role: UPSTREAM_REOPEN_LEAD
Role contract: docs/spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree

Reopen <one macro phase> only to close <recorded condition> for <blocked unit>.
- Authority: <canonical blocker, phase artifacts, current revisions, and initiating native-control envelope>.
- Preserved implementation: <blocked Lead identity, Goal state, candidate identity or bounded Local diff, and attributed dirt>.
- Phase boundary: <smallest owning phase, allowed artifact paths, and decisions that remain unchanged>.
- Proof: <phase stop rule, triggered review, and movement evidence>.
- Stop: <review-cleared phase result and next owner, or exact user/external/native boundary>.

Load the workflow router, the named phase, and its triggered conditional owners.
Do not set an Implementation Goal. Close this phase through its review, repair,
and focused re-review loop, return the phase result and next owner to the Ledger
Orchestrator, and stop without entering another phase or Implementation.
```

The Orchestrator verifies the returned artifact revisions and review disposition,
then routes another fresh Reopen Lead only for a downstream macro phase whose
inputs changed. When Planning makes the ledger executable again, any new or
reopened prerequisite repair unit runs before the blocked unit.

After the reopen chain and prerequisites close, inspect the installed native
controls. When they expose documented Goal resume, send this continuation to
the same blocked Lead with no model or effort override and resume its Goal:

```text
Execution role: ACCEPTANCE_UNIT_LEAD
Role contract: docs/spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree

Resume the same <unit ID> and <dispatch_scope> after its upstream reopen.
- Closure: <review-cleared artifact revisions, prerequisite receipts, and changed reopen condition>.
- Candidate: <preserved candidate identity or bounded Local diff and current checkout precondition>.
- Proof: <still-valid proof and invalidated proof to rerun>.

Resume the existing thread-local Goal, revalidate the unit and candidate, and
continue to one canonical receipt or blocker. If the native resume rejects,
make no unit edit and return that evidence to the Ledger Orchestrator.
```

Only after native-schema inspection or a recorded rejection proves that known
Goal non-resumable may the Orchestrator create one replacement Local Lead. Use
the Implementation Entry prompt above with the
same unit, current ledger revision, preserved candidate, predecessor task and
native failure evidence, plus a new attempt in `dispatch_scope`; apply the
Orchestrator-selected model and effort through the autonomous native dispatch
protocol. The replacement starts a new Local acceptance Goal,
revalidates the candidate before editing, and becomes the sole acceptance owner.
Unknown task, Goal, create, candidate, or Handoff state remains a blocker and
never authorizes replacement.

## Macro-Phase Handoff

Treat handoff as a chain of custody. At a ready macro-phase boundary, return the
phase result and a short standalone `Next Session Prompt`, then stop. Emit that
prompt only when the current macro phase is complete, every triggered review has
a movement-allowing disposition, and a different macro phase is the next owner.
An `UPSTREAM_REOPEN_LEAD` instead returns through [Upstream Reopen And
Implementation Return](#upstream-reopen-and-implementation-return); it emits no
user-visible prompt and leaves the Ledger Orchestrator to route the next owner.
When that next owner is Implementation, use the Implementation Entry And
Continuation Handoff above instead of the generic prompt below.
An incomplete or blocked phase, same-phase reopen, or context rollover preserves
or reports resume state without a user-visible prompt unless the user explicitly
asks for one or the Implementation Entry And Continuation Handoff above applies.
The result tells the user
what was completed, the decisions and authority that now hold, the movement
evidence, and any open proof or risk. The prompt carries only the facts and
evidence-backed direction needed to start the target macro phase from durable
sources without replaying chat. Give the target session the strongest current
continuation hypothesis, why it deserves attention, and the constraint or
evidence that could overturn it. This steers the next phase without turning a
candidate direction into accepted authority: the target session tests it first, then
keeps, revises, or rejects it from current evidence and records why. When the
completed phase supports no candidate solution, carry its most discriminating
next decision question and evidence basis instead of inventing one.

```text
Continue with <target macro phase> only. <Prior macro phase> is ready: <movement evidence>. Start from <minimal authoritative paths or external sources>. Use <current candidate direction> as the leading hypothesis because <decision-relevant evidence>; test it first against <named constraints or falsifier>, and keep, revise, or reject it from current evidence rather than treating it as settled. First, <one concrete action>. Preserve <critical authority boundary>; stop or reopen at <exact condition and owner>. Do not enter the following macro phase in this session.
```

A risk-triggered research-synthesis challenge and triggered specification,
technical-design, test-design, and task-readiness reviews are internal
checkpoints. The root launches the applicable read-only lane and continues the
review, repair, and focused re-review loop in the same session when possible.
They do not create a macro-phase handoff. When current-phase evidence invalidates
a narrow upstream artifact, the root suspends the active phase, closes that
upstream repair and its triggered fresh review, and resumes the active phase in
the same session.

An explicitly requested standalone review remains read-only: return its complete
result and stop at that review boundary. It gains no repair, implementation, or
workflow-handoff authority unless the user separately grants it.

An explicitly requested review of completed implementation may run inside
implementation when the request makes it an acceptance condition; otherwise it
begins after Implementation / Validation / Closeout and never retroactively
becomes an acceptance or closeout gate.

## Stop Rule

The target session can continue from the named sources without reconstructing chat,
and the phase result plus prompt preserve the target owner, authority boundary,
proof obligation, evidence-backed leading hypothesis or decision question and
its falsifier, next executable action, and exact stop or reopen condition. The
prompt neither discards the prior phase's decision implications nor presents a
candidate direction as closed. The current session does not begin the target
macro phase.
