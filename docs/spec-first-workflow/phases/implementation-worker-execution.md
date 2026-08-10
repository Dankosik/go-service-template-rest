# Implementation Worker Execution

Choose the execution carrier from the acceptance unit's execution need, not its
workflow label. Use one isolated harness-native implementation Worker for a
ready acceptance unit only when the user explicitly requests an isolated Worker
or Worktree, root-local execution would materially risk unrelated dirty state
and an isolated checkout is the narrowest safe boundary, the root must continue
coordinating another owner while this closed unit executes in a bounded separate
context, or the unit belongs to a positively independent planned wave. Otherwise
execute the closed acceptance unit root-locally under [Local
Execution](implementation-validation-closeout.md#local-execution) and the same
acceptance contract.

When a required native Worker or shared-lane control is unavailable, execute
the same closed acceptance unit root-locally and state the missing control; this
changes the carrier, not the workflow path or acceptance contract. Create one harness-native root
durable execution control only for a genuinely multi-step or resumable outcome;
[Agent Harness](../../agent-harness.md#goal-mechanics) owns the mechanism.

Before isolated Worker dispatch, put the exact accepted `tasks.md` revision and
every cited durable authority in the accepted integration base visible to the
Worker. An untracked or working-tree-only ledger routes to root-local execution
until that input becomes part of the base. Inspect the source checkout,
authorized paths, and current diff/status; record the base because the candidate
will cross a checkout boundary. Preserve unrelated user changes.

Keep one isolated write Worker per [acceptance
unit](planning.md#outputs); several isolated write Workers may run only as
members of a positively independent planned wave. One fixed acceptance unit may
instead use the shared-checkout implementation lanes below; those lanes do not
split its ledger, review, proof, receipt, or transition. The unit's ledger
entries are the brief body: dispatch only the ledger index path, the unit's
task-file paths when the ledger is split, unit or task IDs, and live facts the
ledger cannot contain. A repeated task summary is a dispatch defect because it
creates a second, drift-prone instruction source.

Explicitly and independently select and pass the best-suited available model
and unit-matched reasoning effort for every Worker. [Agent
Harness](../../agent-harness.md#model-and-effort-selection) owns tier mappings,
stronger-model conditions, and supported controls.

## Shared-Checkout Implementation Lanes

At selection of a ready acceptance unit, inspect its current code, exact
writable paths, dependency direction, generated/manual authority, mutable
resources, and proof preconditions. Use concurrent shared-checkout
implementation lanes only when all of these hold:

- two or more useful implementation slices can start immediately from the same
  fixed accepted contract and current checkout;
- every writable path is exact and pairwise disjoint, with one active writer
  per file;
- no lane must choose or revise an interface, schema, generated authority,
  behavior, ownership, or proof precondition consumed by another lane;
- every cross-lane, integration, ledger, generated, formatting, aggregate-proof,
  review, and receipt surface is reserved to the root; and
- focused lane checks can run without a shared mutable resource or a broad Go
  or Docker gate.

Do not create lanes merely to fill available capacity. Coupled files stay with
one writer; dependent acceptance units remain dependency-ordered even when
parts of their future code could be written early. Lane count is the number of
eligible useful slices, capped by current harness and focused-proof capacity.

Dispatch every eligible lane concurrently through the current harness's shared-
checkout implementation carrier. Each brief names the acceptance unit, the
lane-specific postcondition, exact writable paths, root-reserved paths, fixed
contract sources, allowed focused proof, stop condition, and return shape.
Workers may inspect repository sources but write only their owned paths. Every
brief includes these constraints:

- preserve all existing and unrelated dirt;
- do not edit the task ledger, commit, rebase, stash, deploy, or otherwise
  mutate Git state;
- do not edit another lane's or the root's paths; and
- stop and return the issue to the root when another owner's file, a shared
  mutable resource, an unrecorded contract choice, or a scope expansion becomes
  necessary.

Every lane returns `DONE` or `BLOCKED`, its changed paths, focused commands and
results, unresolved issue, and `provisional edits: present|none`. Shared-checkout
edits remain provisional until root intake. A blocked lane leaves any partial
edits in its owned paths and never resets, checks out, or stashes them. The root
preserves that diff; it may return a lane-local correction to the same idle lane
but owns cross-lane reconciliation only after all writers stop.

While lanes run, the root writes only root-reserved paths and waits to inspect
or reconcile lane-owned content until the relevant lane returns. A local
correction returns to that same lane. After every lane returns, keep all lanes
stopped while the root verifies path scope, reconciles the combined change,
formats it, runs focused checks, and then runs the unit's exact broad or Docker
proof serially. The root alone owns final review, any triggered independent
review, acceptance, receipt, and the persisted ledger transition.

## Execution-Ready Dispatch

Dispatch only when the route is closed: the reproducer or current facts,
narrowest owner, known cause or deterministic mechanism, expected behavior,
editable boundary, and exact proof live in the accepted ledger entry or its
live delta. If an intermediate finding still determines the next step, keep
discovery root-local.

If a dispatched task exposes an open route, the Worker returns the frozen
candidate and missing decision. The root closes that route before dispatching
implementation again. [Agent Harness](../../agent-harness.md#what-crosses-into-a-worker)
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
either continue independent root work or wait again. Never convert partial
progress or a mutable diff into a review finding.

## Correction Loop

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
required proof. Before root intake review, derive paths
with `git diff --name-only <recorded-base> <candidate-tree>` or the
harness-native equivalent. For an uncommitted checkout, combine
`git diff --name-only <recorded-base>` with
`git ls-files --others --exclude-standard` so the check includes untracked
paths. A scope-invalid candidate has one disposition: reject it in full from
the recorded base while other provisional wave members remain unaffected. A
required boundary expansion reopens the scope owner before implementation.

## Progress

Continue the write lane only when a returned candidate materially advances an
open finding or new evidence changes the current causal hypothesis. When the
effective inputs, hypothesis, and expected observable already have a
disposition, retain the preceding frozen candidate and return the honest
blocker. A correction re-enters the write lane only through the Diagnostic
Gate. Judge convergence from returned candidates, not elapsed time, wait
timeouts, message count, or intermediate Worker activity.

For a planned write wave, every member starts from the same accepted integrated base and every returned result remains provisional. Assemble only bounded deltas into a frozen candidate.

When a failure is isolated and the reviewed independence basis still holds,
shrink the wave to the proven passing subset. Review, prove, and integrate that
subset while the failed member and its dependents remain provisional. Keep
coupled members together when the failure crosses an interface, invariant,
generated/manual authority, mutable resource, or proof precondition. Start
later work only through the phase-owned [Acceptance-Unit
Closure](implementation-validation-closeout.md#acceptance-unit-closure).

## Candidate Intake And Correction

### Monotonic Intake

The root owns candidate intake, correction routing, acceptance, and integration.
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
