# Implementation Worker Execution

## Execution Role Tree

The user supplies a ready Implementation ledger once and explicitly launches
its orchestration. The Ledger Orchestrator then routes autonomously until that
ledger is exhausted or one terminal condition in the role table holds. It does
not perform Intake, Research, Specification, Design, Test Design,
Planning, or unit work itself. It may suspend Implementation and route a
canonical agent-owned upstream reopen through fresh phase tasks, and never asks
the user to choose a carrier, lane, model, effort, review, proof, correction,
or reopen route.

Optimize wall-clock time without weakening acceptance quality: plan the
[smallest coherent acceptance units](planning.md#acceptance-unit-contract), run
only positively independent units or slices concurrently, keep one Lead as the
acceptance owner, integrate serially, and reuse proof while its preconditions
remain valid.

Every top-level task and implementation leaf binds exactly one execution role
before its first governed action. A native task or subagent is a carrier, not a
role; each such dispatch starts with `Execution role: <ROLE>` and links to this
tree. Built-in read-only lanes opened inside an upstream non-implementation
phase follow that phase and Subagents And Review rather than binding an
Implementation role. A missing or unknown required role makes the handoff
invalid rather than granting broader authority. [Agent Harness's Codex App Selection
Tree](../../agent-harness.md#codex-app-selection-tree) shows which actor selects
the model and reasoning effort for each direct child.

```mermaid
flowchart TD
    user["User<br/>ready Implementation ledger<br/>one orchestration launch"]
    orchestrator["LEDGER_ORCHESTRATOR<br/>routing only"]
    lead["ACCEPTANCE_UNIT_LEAD<br/>one fresh native task per ready unit"]
    reopen["UPSTREAM_REOPEN_LEAD<br/>one fresh task for one macro phase"]
    strategy{"Lead chooses the unit strategy"}
    serial["Lead executes serially"]
    specialist["READ_ONLY_SPECIALIST<br/>optional leaf"]
    worker["IMPLEMENTATION_WORKER<br/>optional isolated write leaf"]
    reviewer["ACCEPTANCE_REVIEWER<br/>triggered fresh read-only leaf"]
    fanin["Lead serial fan-in<br/>integration · self-review · proof"]
    review{"Independent review triggered?"}
    acceptance["Lead corrections and acceptance"]
    result["One canonical receipt or blocker"]
    done["Ledger exhausted"]
    stopped["Exact user/external boundary,<br/>unrecoverable native blocker,<br/>or canonical blocker with no ready/recovery"]

    user --> orchestrator
    orchestrator -->|"one ready unit; several only in a proved wave"| lead
    orchestrator -->|"agent-owned canonical reopen"| reopen
    reopen -->|"review-cleared phase result"| orchestrator
    orchestrator -->|"resume blocked unit after closure"| lead
    lead --> strategy
    strategy -->|"no useful independent work"| serial
    strategy -->|"one independent question"| specialist
    strategy -->|"independent write slices"| worker
    serial --> fanin
    specialist --> fanin
    worker --> fanin
    fanin --> review
    review -->|"yes"| reviewer
    review -->|"no"| acceptance
    reviewer --> acceptance
    acceptance --> result
    result -->|"re-read ledger and route again"| orchestrator
    orchestrator -->|"no ready or pending units"| done
    orchestrator -->|"stop condition reached"| stopped
```

| Role | Receives from | Owns | May dispatch | Upward result |
| --- | --- | --- | --- | --- |
| `LEDGER_ORCHESTRATOR` — Ledger Orchestrator | Ready canonical ledger and explicit task-creation authority | Ready-unit selection, agent-owned upstream-reopen routing, native task lifecycle, and terminal routing | One `ACCEPTANCE_UNIT_LEAD` per ready unit; several only for a ledger-proven planned wave; one `UPSTREAM_REOPEN_LEAD` at a time | Ledger exhausted; an AGENTS-owned user decision or external confirmation; an unrecoverable native blocker; or a canonical blocker with neither ready work nor authorized recovery |
| `ACCEPTANCE_UNIT_LEAD` — Acceptance-Unit Lead | Ledger Orchestrator, Planning handoff, or direct Implementation entry | Exactly one unit through strategy, implementation, serial integration, review, proof, correction, acceptance, and receipt | Optional leaf specialists, Workers, and a triggered reviewer | `HANDOFF_READY` for a fixed Worktree candidate, then one canonical `Accepted:` receipt or `Blocked:` record |
| `UPSTREAM_REOPEN_LEAD` — Upstream Reopen Lead | Ledger Orchestrator and one canonical unit blocker | Exactly one named non-implementation macro phase through its phase stop rule, triggered review, repair, and focused re-review | Phase-eligible read-only lanes and triggered reviewers under Subagents And Review | Review-cleared phase result and next owner, or the exact user/external/native boundary that prevents closure |
| `READ_ONLY_SPECIALIST` — Read-Only Specialist | Acceptance-Unit Lead | One independently checkable question | Nothing; it is a leaf | `DONE` with evidence, or `NEEDS_PARENT` |
| `IMPLEMENTATION_WORKER` — Implementation Worker | Acceptance-Unit Lead | One exact write slice and its focused proof | Nothing; it is a leaf | `DONE` with a frozen candidate, or `NEEDS_PARENT` |
| `ACCEPTANCE_REVIEWER` — Acceptance Reviewer | Acceptance-Unit Lead | Independent falsification of one fixed unit | Nothing; it is a fresh one-shot leaf | `PASS`, `FAIL`, or `NEEDS_PARENT` to the Lead |

The bound role does not change during the session. A Lead that implements
serially remains the Lead; it does not relabel itself as a Worker. A child gets
only its row's authority, not its parent's. A correction resumes the same actor
under the same role. Acceptance-unit roles never revise behavior, unit scope,
or ledger dependencies. An Upstream Reopen Lead may revise only its named
phase-owned artifacts; it returns at that macro-phase boundary and never enters
Implementation or the next phase. Planning may change the ledger only in a
separately routed Planning reopen.

### Bottom-Up Obstacle Resolution

| Current actor | Direct parent | Obstacle result |
| --- | --- | --- |
| `READ_ONLY_SPECIALIST`, `IMPLEMENTATION_WORKER`, or `ACCEPTANCE_REVIEWER` | `ACCEPTANCE_UNIT_LEAD` | `NEEDS_PARENT` |
| `ACCEPTANCE_UNIT_LEAD` | `LEDGER_ORCHESTRATOR` | Canonical unit `Blocked:` record |
| `UPSTREAM_REOPEN_LEAD` | `LEDGER_ORCHESTRATOR` | Review-cleared phase result and next owner, or exact user/external/native boundary |
| `LEDGER_ORCHESTRATOR` | User | Only an AGENTS-owned user decision, irreversible external-effect confirmation, unrecoverable native blocker, or canonical blocker with no ready work or authorized recovery |

An obstacle is not a canonical blocker while a safe resolution remains inside
the current actor's authority. The actor first diagnoses it, uses its available
tools and evidence, and attempts the narrowest in-scope remedy. An attempt is
eligible only when evidence, inputs, the causal hypothesis, or the expected
observable changed; never repeat a route under the same preconditions. If no
eligible remedy remains, a Specialist, Worker, or Reviewer returns
`NEEDS_PARENT` exactly one level with the observed evidence, attempted actions
and results, the boundary that cannot be crossed, and one requested
parent-owned action. Never skip the direct parent or ask the user directly.

A Specialist, Worker, or Reviewer returns its obstacle only to the
Acceptance-Unit Lead. `NEEDS_PARENT` is a message, not artifact state, partial
acceptance, or dependency release. The Lead re-diagnoses instead of
copying that result into the ledger. It uses unit-level authority to close
technical decisions, revise the internal execution strategy, fall back from
fan-out to serial work, obtain missing evidence, integrate, or route a valid
same-actor correction. Review findings always return to the Lead for this
disposition.

The Lead records the unit's one canonical blocker only after every safe
unit-local route is exhausted, or when resolution requires a change to accepted
behavior, unit scope, ledger dependencies, unavailable external authority, or
an upstream owner. The blocker names the evidence, attempted routes, exact
boundary, reopen owner and condition, and preserved candidate. It is not a
transcript of attempts.

The Ledger Orchestrator consumes only that canonical transition or a native
lifecycle obstacle. It resolves native identity, wait, resume, Handoff, and
routing issues within its authority and continues unrelated ready units. If it
receives a misrouted `NEEDS_PARENT`, it returns the message unchanged to its
owning Lead instead of consuming it. The Orchestrator never enters the unit or
an upstream phase to implement, author, review, prove, or correct it. An
outcome-level stop occurs only under its table row's terminal conditions.

### Upstream Reopen Recovery

A Worktree Lead that reaches an agent-owned upstream boundary first freezes its
current candidate and returns `HANDOFF_READY` with the proposed blocker,
evidence, reopen owner, and condition. It does not persist `Blocked:` from the
Worktree. The Orchestrator applies the ordinary native Handoff to move that same
Lead and candidate into Local; the Lead's Local Goal revalidates the boundary
and either takes a newly available unit-local remedy or persists the one
canonical blocker there. Failed or ambiguous Handoff keeps the candidate in its
original carrier and enters ordinary Handoff recovery.

A canonical blocker whose recorded reopen owner is agent-owned is an authorized
recovery route, not an outcome-level stop. Keep the blocked Lead, its native
identity, Goal, candidate, and blocker pinned. Dispatch one fresh Local
`UPSTREAM_REOPEN_LEAD` for the smallest named macro phase using [Upstream Reopen
And Implementation Return](../shared/resume-and-handoff.md#upstream-reopen-and-implementation-return),
then wait for that phase's movement-allowing result. The Reopen Lead applies the
phase's ordinary delegation and review contract through repair and focused
re-review; it starts neither another macro phase nor Implementation.

Re-read the canonical artifacts after every reopened phase. Route another fresh
Reopen Lead only when the completed change invalidated that downstream phase's
inputs; otherwise preserve its prior disposition. If Planning must repair unit
scope or dependencies, route Planning separately. A new or reopened prerequisite
repair unit is accepted before the blocked unit continues; never widen the
blocked unit silently.

Once Implementation inputs are closed and the recorded reopen condition has
changed, inspect the installed native controls. When they expose documented
Goal resume, resume the original Lead and its Goal first; an ordinary message is
not Goal-resume proof. If current native-schema inspection or a recorded
rejection proves that the Goal cannot resume, the Orchestrator may create one
fresh Local replacement `ACCEPTANCE_UNIT_LEAD` for the same unit from the
preserved candidate and current artifact revisions. The replacement uses a new
attempt in `dispatch_scope`, revalidates the candidate and checkout before work,
and becomes the sole acceptance owner. An unknown task, Goal, create, candidate,
or Handoff outcome never qualifies for replacement.

An explicitly requested Ledger Orchestrator dispatches one ready
[acceptance unit](planning.md#outputs), or each currently ready member of a
ledger-proven independent planned wave, to a separate fresh native task hosting
an Acceptance-Unit Lead. A serial unit starts in Local; each planned-wave member
starts in its own Worktree from the recorded base. The Orchestrator selects no
internal carrier or lane. Outside global orchestration, the current
Implementation root binds `ACCEPTANCE_UNIT_LEAD` when it selects a unit.

The Acceptance-Unit Lead inspects the fixed unit, current repository, relevant
dirt, dependencies, generated/manual authority, mutable resources, and proof
preconditions before choosing serial execution or bounded one-level fan-out.
Internal decomposition may realize only the accepted unit outcome: it never
changes accepted behavior, splits the ledger unit, revises dependencies, or
starts another acceptance unit. New evidence that requires one of those changes
blocks the unit and reopens its owner.

The unit's ledger entries are the brief body. Every different-session dispatch
uses [Implementation Entry And Continuation
Handoff](../shared/resume-and-handoff.md#implementation-entry-and-continuation-handoff)
to carry exactly one unit ID, its accepted revision or receipt, recorded base
when a candidate crosses a checkout, relevant dirt, current external-effect
authority, verbatim native-control authority, proof, stop condition, and unique
dispatch scope. The Ledger
Orchestrator supplies no lane map.

A Worktree Lead completes its Worktree Goal before returning a fixed candidate
as `HANDOFF_READY`; when an upstream boundary caused the stop, that return also
carries the proposed canonical blocker for Local revalidation. `HANDOFF_READY`
is not an artifact state, receipt, acceptance, or dependency release. [Agent Harness's native Worktree
fan-in](../../agent-harness.md#worktree-fan-in) solely owns the Handoff mechanics
that move the same task into Local, one Lead at a time. After successful
Handoff, the same Lead creates a separate Local Goal, then integrates, reviews,
proves, corrects, and records the one canonical receipt or blocker. Moving the
carrier is Orchestrator lifecycle routing; candidate judgment and integration
remain Lead-owned.

When the Lead works serially, it implements, reviews, proves, corrects, and
records the unit receipt or blocker itself. When it selects fan-out, it retains
the same responsibilities and integrates every returned lane serially. A
routing-only Ledger Orchestrator that lacks fresh top-level task creation stops
blocked instead of implementing. A missing inner fan-out control returns
execution to the same Lead serially.

## Acceptance-Unit Lead Fan-Out

Open only the lanes that current unit evidence makes useful:

- use a read-only research, evidence, or review lane for one independently
  checkable question whose result can change this unit's implementation or
  acceptance;
- use an isolated write lane only when two or more implementation slices can
  start from the same fixed unit contract and accepted base, with exact
  pairwise-disjoint writable paths and one writer per path;
- keep every shared interface, schema, generated authority, mutable resource,
  integration, ledger, formatting, aggregate-proof, review, and receipt surface
  with the Lead; and
- require focused lane checks that do not depend on another lane's provisional
  output or a shared broad or Docker gate.

Lane count is the useful independent width, bounded by harness and proof
capacity. Do not create lanes to fill capacity. Dispatch isolated writers from
one accepted base and explicitly select each lane's model and reasoning effort
under [Agent Harness](../../agent-harness.md#model-and-effort-selection).
Read-only lanes write nothing. Every internal lane remains a leaf; in
particular, a discovered child write dependency returns to the Lead instead
of being dispatched by a write lane.

Each lane brief begins:

```text
Execution role: <READ_ONLY_SPECIALIST | IMPLEMENTATION_WORKER>
Role contract: docs/spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree
```

It then names the unit, lane outcome, exact writable or read-only surface,
Lead-reserved surfaces, fixed authorities, focused proof, and stop condition. A
write lane may return a fixed commit, tree, or bounded diff from its Worktree;
it never edits the ledger, integrates, rebases, stashes, deploys, or mutates the
integration checkout. Every lane preserves unrelated work and stops before
crossing another owner, changing accepted behavior or dependencies, or widening
the unit.

Every lane returns `DONE` or `NEEDS_PARENT`, source or changed paths, focused
commands and results, candidate identity when applicable, and the one-level
escalation envelope above. The Lead waits for all current lanes, freezes their
output, verifies scope and ownership, and performs candidate intake and
integration serially. It alone
reconciles and formats the combined change, runs focused and aggregate proof,
performs Lead self-review and triggered independent review, routes same-lane
corrections when useful, and writes the one unit receipt or blocker. No internal subset
creates a receipt or releases a ledger dependency.

## Execution-Ready Dispatch

Dispatch only when the route is closed: the reproducer or current facts,
narrowest owner, known cause or deterministic mechanism, expected behavior,
editable boundary, and exact proof live in the accepted ledger entry or its
live delta. If an intermediate finding still determines the next step, keep
discovery Lead-local.

If a dispatched task exposes an open route, the Worker returns the frozen
candidate and missing decision. The Lead closes that route before
dispatching implementation again. [Agent Harness](../../agent-harness.md#what-crosses-into-a-worker)
owns what context crosses into a Worker.

## Observe And Freeze

Treat a Worker worktree as mutable until the Worker returns. Observe native
completion events and stable status only; inspect or review the candidate after
the return identifies a fixed commit, tree, or bounded diff. Before return,
message the Worker only to stop unsafe work or when new evidence invalidates an
accepted input; ordinary findings wait for the frozen candidate.

Wait on all currently relevant Workers together when the harness supports it,
using the latest delivered cursor or equivalent. An unchanged timeout carries
no new evidence: preserve the current disposition, emit no correction, and
either continue independent Lead work or wait again. Never convert partial
progress or a mutable diff into a review finding.

## Correction Loop

The Acceptance-Unit Lead owns every correction through its terminal receipt
or blocker; the Ledger Orchestrator remains waiting and makes no correction
decision.

A returned candidate that misses an accepted criterion goes back to **the same
Worker**, with its context intact, through the harness's own correction channel.
Spawning a fresh Worker for the same task instead is a defect: it throws away
the reasoning that made the second attempt cheaper than the first, and it
re-opens questions the first attempt already closed.

Review one frozen candidate, disposition all supported findings, and send one
batched correction brief. It names each finding, the criterion it violates, and
the proof that must change — not a restatement of the unit. Route it through the
[Diagnostic Gate](#diagnostic-gate) so a correction that cannot name a
candidate-caused regression, a violated accepted criterion or repository-owned
invariant, or missing proof is recorded as an observation rather than
re-entering the write lane.

Before sending a third correction batch for the same acceptance unit, audit the
route: cause, owner, accepted input, unit boundary, and proof strategy. Resume
the same Worker only when new evidence closes the diagnosed route defect;
otherwise reopen the narrowest upstream owner or return the honest blocker.

Replace a Worker only for an execution stall that produces no new turn, or for
an invalidated base, and then continue the same exact brief from the frozen
candidate. Worker replacement resets context; it is a recovery action, never a
correction technique.

## Scope Lock

A candidate is scope-valid only when every changed path is authorized by an
explicit editable path or by the deterministic placement rule in its bounded
discovery boundary, and every retained change maps to an accepted criterion or
required proof. Before Lead intake review, derive paths
with `git diff --name-only <recorded-base> <candidate-tree>` or the
harness-native equivalent. For an uncommitted checkout, combine
`git diff --name-only <recorded-base>` with
`git ls-files --others --exclude-standard` so the check includes untracked
paths. A scope-invalid lane candidate has one disposition: reject it in full
from the recorded base while other provisional unit lanes remain unaffected. A
required boundary expansion reopens the scope owner before implementation.

## Progress

Continue the write lane only when a returned candidate materially advances an
open finding or new evidence changes the current causal hypothesis. When the
effective inputs, hypothesis, and expected observable already have a
disposition, retain the preceding frozen candidate and return the honest
blocker. A correction re-enters the write lane only through the Diagnostic
Gate. Judge convergence from returned candidates, not elapsed time, wait
timeouts, message count, or intermediate Worker activity.

For internal write fan-out, every lane starts from the same accepted unit base
and every return remains provisional. Assemble only bounded deltas into the
unit's frozen candidate. A passing lane may be retained after another fails,
but no subset is accepted independently: the Lead completes or blocks the
whole unit. Start later work only through the phase-owned [Acceptance-Unit
Closure](implementation-validation-closeout.md#acceptance-unit-closure).

## Candidate Intake And Correction

### Monotonic Intake

The Acceptance-Unit Lead owns candidate intake, correction routing,
acceptance, and integration.
Each Worker return first passes Scope Lock plus ownership, mergeability, and
proof-provenance intake. The first intake-valid candidate becomes the frozen
baseline for the phase-owned bounded
[review](implementation-validation-closeout.md#review) and mapped proof. That
review creates the finite finding set supported by the evidence then available,
containing only candidate-caused regressions, concrete violations of accepted
criteria or repository invariants, and proof missing from an accepted claim.
All other observations retain observation status.

Correction verification covers the open findings, the delta from the frozen
baseline, and proof invalidated by that delta; unchanged bytes retain their
prior disposition. Adopt a correction only when it closes or disproves at least
one finding, reopens none, introduces no regression, stays in scope, and
preserves passing proof. Adoption makes the correction the current frozen
baseline and the open set a strict subset. An empty set plus passing mapped
proof, plus a passing independent review when triggered, accepts the candidate.

### Diagnostic Gate

A correction that introduces a regression, reopens a dispositioned finding, or
fails to shrink the current finding set has one disposition: reject the delta
and retain the preceding frozen baseline. Its bytes are diagnosis evidence
only, never accepted change or proof. Re-enter correction only when
[Progress](#progress) has new evidence that changes the causal hypothesis or
expected observable; otherwise return the honest blocker.

Current evidence outranks the finding set. Evidence first available after the
full review adds or reopens a finding only when it proves a candidate-caused
regression, a violation of an accepted criterion or repository invariant, or
missing proof for an accepted claim. Reopen an upstream owner only when that
evidence invalidates an accepted decision rather than the candidate.

The local repository default/main is the authoritative integration branch
unless the user names another persistent branch. Integrate only the accepted
unit delta and confirm the authoritative diff still contains that candidate
without unrelated changes. If integration changes relevant content, validate
only the affected claims. When integration is commit-backed, land the final
accepted delta and its ledger transition as one acceptance commit per unit;
intermediate Worker and correction commits remain candidate history rather than
integration history. Use a commit/tree identity only when proof crosses a
checkout or integration boundary; never hash individual files or specifications
for this purpose. Preserve unrelated dirty state during integration; publication
remains governed by [AGENTS.md Authorization And
Boundaries](../../../AGENTS.md#authorization-and-boundaries).
