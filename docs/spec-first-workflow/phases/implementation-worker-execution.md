# Implementation Worker Execution

Choose the execution carrier from the task's execution need, not its workflow
label. Use one harness-native implementation Worker for a ready ledger task only
when the user explicitly requests a Worker, its isolated checkout protects
unrelated state, a bounded separate context is needed while the root coordinates
other owners, or it belongs to a positively independent planned wave. Otherwise
execute the closed ledger task root-locally under [Local
Execution](implementation-validation-closeout.md#local-execution) and the same
acceptance contract.

When a required native Worker control is unavailable, execute the same closed
ledger task root-locally and state the missing control; this changes the carrier,
not the workflow path or acceptance contract. Create one harness-native root
durable execution control only for a genuinely multi-step or resumable outcome;
[Agent Harness](../../agent-harness.md#goal-mechanics) owns the mechanism.

Before dispatching from uncommitted accepted input, identify the source
checkout and authorized paths and inspect its current diff/status. Record a
base only when the candidate will leave that checkout or several provisional
deltas must be compared. Do not stash, clean, ignore, or mutate user changes.
Keep one write Worker per ledger task; several write Workers may run only as
members of a positively independent planned wave. When the Worker can read the
exact accepted `tasks.md` revision, its task entry is the brief body: dispatch
only the artifact path, task ID, and current facts that the ledger could not
contain. Do not paste or restate the task. Send the full outcome-first brief
inline only when that exact ledger is unavailable to the Worker.

Explicitly and independently select and pass the best-suited available model
and task-matched reasoning effort for every Worker. [Agent
Harness](../../agent-harness.md#model-and-effort-selection) owns tier mappings,
stronger-model conditions, and supported controls.

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

## Correction Loop

A returned candidate that misses an accepted criterion goes back to **the same
Worker**, with its context intact, through the harness's own correction channel.
Spawning a fresh Worker for the same task instead is a defect: it throws away
the reasoning that made the second attempt cheaper than the first, and it
re-opens questions the first attempt already closed.

A correction brief names the finding, the criterion it violates, and the proof
that must change — not a restatement of the whole task. Route it through the
[Diagnostic Gate](#diagnostic-gate) so a correction that cannot name a
candidate-caused regression, a violated accepted criterion or repository-owned
invariant, or missing proof is recorded as an observation rather than
re-entering the write lane.

Replace a Worker only for an execution stall that produces no new turn, or for
an invalidated base, and then continue the same exact brief from the frozen
candidate. Worker replacement resets context; it is a recovery action, never a
correction technique.

## Scope Lock

A candidate is scope-valid only when every changed path is authorized by an
explicit editable path or by the deterministic placement rule in its bounded
discovery boundary, and every retained change maps to an accepted criterion or
required proof. Before acceptance review, derive paths
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
Gate. Follow native completion events; when they expose no intermediate
progress, judge convergence from returned candidates.

For a planned write wave, every member starts from the same accepted integrated base and every returned result remains provisional. Assemble only bounded deltas into a frozen candidate.

When a failure is isolated and the reviewed independence basis still holds, shrink the wave to the proven passing subset. Review, prove, integrate, and accept that subset while the failed member and its dependents remain provisional. Keep coupled members together when the failure crosses an interface, invariant, generated/manual authority, mutable resource, or proof precondition. Start later work only from an accepted commit or tree whose accepted subset satisfies its dependencies.

## Candidate Acceptance

### Monotonic Acceptance

The root is the acceptance authority. Each Worker return first passes Scope
Lock plus ownership, mergeability, and proof-provenance intake. The first
acceptance-ready candidate becomes the frozen baseline for one full acceptance
review and its initial mapped proof. That review creates the finite finding set
supported by the evidence then available, containing only candidate-caused
regressions, concrete violations of accepted criteria or repository
invariants, and proof missing from an accepted claim. All other observations
retain observation status.

Correction verification covers the open findings, the delta from the frozen
baseline, and proof invalidated by that delta; unchanged bytes retain their
prior disposition. Adopt a correction only when it closes or disproves at least
one finding, reopens none, introduces no regression, stays in scope, and
preserves passing proof. Adoption makes the correction the current frozen
baseline and the open set a strict subset. An empty set plus passing mapped
proof accepts the candidate immediately.

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

The local repository default/main is the authoritative integration branch unless the user names another persistent branch. Run mapped claim-scoped proof on the reviewed candidate, integrate only the accepted delta, and confirm the authoritative diff still contains that candidate without unrelated changes. If integration changes relevant content, validate the affected claims before acceptance. Use a commit/tree identity only when proof crosses a checkout or integration boundary; never hash individual files or specifications for this purpose. Preserve unrelated dirty state during integration; publication remains governed by [AGENTS.md Authorization And Boundaries](../../../AGENTS.md#authorization-and-boundaries).
