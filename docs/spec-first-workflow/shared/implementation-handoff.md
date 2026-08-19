# Implementation Handoff

Load only when another session must enter or continue Implementation, a known
Lead needs terminalization, or a canonical blocker requires an agent-owned
upstream reopen and return.

## Lead Prompt Header

Every Acceptance-Unit Lead prompt begins with this expanded header:

```text
Execution role: ACCEPTANCE_UNIT_LEAD
Skill: $acceptance-unit-lead
Role contract: docs/spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree
```

The emitted prompt is standalone; include the header text rather than referring
to its section name.

## Implementation Entry And Continuation Handoff

After Planning movement, a persisted unit transition, or an explicitly requested
different-session resume, inspect the authoritative ledger and checkout and
select only the next ready unit. The handoff basis records its unit ID, accepted
ledger revision or prerequisite receipt, integration base, attributed dirt,
current external-effect authority or locator, and verified native capabilities.
Use a commit or tree only across a checkout or integration boundary. The Ledger
Orchestrator selects neither carrier nor internal lane map; every unit enters
through an Acceptance-Unit Lead, and missing native controls are capability
blockers.

After the [Lead Prompt Header](#lead-prompt-header), emit:

```text
Lead <acceptance unit> through the assigned stage toward one canonical receipt or blocker.

- Authority: <ledger and task paths; accepted revision or receipt; external-effect envelope or durable locator>.
- Secret inputs: <durable locator or environment-variable names; none when unnecessary>.
- Native controls: <verified capabilities for Worker-backed writes, carrier/base validation, materialization, model/effort, review, recovery, and Handoff>. Goal use stays thread-local; this prompt expands no external-effect authority.
- Scope: <unit ID, accepted outcome and writable boundary; blocked dependants>.
- Dispatch scope: <ledger revision / unit ID / attempt>.
- Stage: <Local acceptance | Worktree candidate>.
- Checkout: <accepted base across a checkout; attributed dirt; evidence-selected starting state when needed>.
- Proof: <accepted proof and completion condition>.
- Stop: <fixed behavior, scope, dependency, authority, or other reopen boundary>.

Set one thread-local Goal for this stage and role and execute the Role Tree. A missing valid carrier or base is the exact blocker. Worktree returns `HANDOFF_READY` with the fixed candidate and no receipt; Local continues under a separate Goal to one receipt or blocker. Preserve unrelated work and keep dependants blocked.
```

Local ends only at its canonical `Accepted:` or `Blocked:` transition. Worktree
may return `HANDOFF_READY`; the same Lead continues in Local through the [Codex
fan-in](../../agent-harness/codex.md#worktree-fan-in). The Ledger Orchestrator
rereads the ledger transition and never routes internal lane results.

## Worktree To Local Continuation

Native Handoff passes one atomic `followUpPrompt`. After the Lead header include:

```text
Continue the same <unit ID> and <dispatch_scope>.
- Stage: Local acceptance.
- Candidate: <fixed commit/tree and Worktree identity>.
- Local precondition: <HEAD, status, and attributed dirt verified immediately before Handoff>.

Create a new Local acceptance Goal. Integrate the fixed candidate, review and prove it, route every implementation correction to its owning Worker, and apply bottom-up resolution before one canonical `Accepted:` or `Blocked:` transition. Keep the role and unit fixed and repeat no route under unchanged preconditions.
```

When `HANDOFF_READY` carries a proposed upstream blocker, replace the final
paragraph with:

```text
Create a new Local blocker-revalidation Goal. Preserve and inspect the fixed candidate, re-run bottom-up resolution against current Local artifacts, and either route a newly available implementation remedy through its Worker or persist one canonical `Blocked:` record with reopen owner and condition. Start no upstream phase or another unit.
```

`HANDOFF_READY` is routing evidence, not artifact state, and releases no
dependency before the Local transition. A proposed blocker is likewise
non-durable until the Local Lead records it.

## Known Lead Terminalization

For one known Lead without a canonical transition, send once with no model or
effort override. After the Lead header include:

```text
Terminalize the same <unit ID> and <dispatch_scope>. Re-read the ledger, native task state, and Git candidate. Persist `Accepted:` when current proof and candidate close the unit; otherwise take each evidence-changing unit-local remedy through its owning Worker and persist `Blocked:` only after none remains. Start no new unit and repeat no valid proof, review, or remedy under unchanged preconditions.
```

## Upstream Reopen And Implementation Return

For a canonical agent-owned upstream blocker, keep the blocked Lead, Goal, and
candidate pinned and create one fresh Local task with the Orchestrator-selected
model and effort:

```text
Execution role: UPSTREAM_REOPEN_LEAD
Skill: $upstream-reopen-lead
Role contract: docs/spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree

Reopen <one macro phase> only to close <condition> for <blocked unit>.
- Authority: <blocker, phase artifacts, revisions, native capabilities>.
- Preserved implementation: <Lead, Goal, candidate or bounded diff, attributed dirt>.
- Phase boundary: <owner, writable artifacts, unchanged decisions>.
- Proof: <stop rule, review, movement evidence>.
- Stop: <review-cleared result and next owner, or exact boundary>.

Load the router, named phase, and triggered owners. Close that phase and its review loop without an Implementation Goal, return its result and next owner to the Ledger Orchestrator, and stop before another phase or Implementation.
```

After closure, the [Codex adapter](../../agent-harness/codex.md#upstream-reopen-and-implementation-return)
selects native resume or its narrowly allowed replacement. For resume, send the
same Lead header followed by:

```text
Resume the same <unit ID> and <dispatch_scope> after its upstream reopen.
- Closure: <review-cleared revisions, prerequisite receipts, changed condition>.
- Candidate: <preserved candidate or bounded diff and checkout precondition>.
- Proof: <still-valid proof and invalidated proof to rerun>.

Resume the existing Goal, revalidate the unit and candidate, and continue to one canonical receipt or blocker. A rejected native resume makes no unit edit and returns its evidence to the Ledger Orchestrator.
```

When replacement is required, reuse the Implementation Entry prompt with its
predecessor and native failure evidence; this branch adds no second recovery
policy.
