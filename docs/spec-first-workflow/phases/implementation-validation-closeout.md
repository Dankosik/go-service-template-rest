# Implementation / Validation / Closeout

## Acceptance Posture

Act as a contract closer. Seek the earliest proven state that satisfies the
accepted outcome while preserving already-valid behavior, decisions, candidate
state, and proof. Spend an edit only on an accepted criterion or current open
finding; all other discoveries retain observation status. Acceptance is
terminal: close out when the criteria and mapped proof pass.

Reason backward from the observable outcome: identify what must be true, its
narrowest implementation owner, the wiring that exposes it on the real path,
and the check that would fail if it were absent or wrong. Task status, file or
symbol presence, an unrelated passing check, and Worker prose are not
completion evidence.

## Read When

- A request authorizes change, build, or fix.
- Direct work has a clear outcome and bounded proof, or structured work has the inputs it actually needs.
- Existing implementation needs repair, validation, or closeout.

## Inputs

- Accepted inline outcome or the smallest relevant durable artifact.
- Current repository state, including unrelated user changes.
- Canonical sources and repository-native proof commands for changed surfaces.

## Outputs

- The bounded code, tests, configuration, generated output, or documentation change.
- Claim-scoped review and proof evidence.
- An honest completion, partial-verification, or blocker statement.

## Implement

Inspect the owning code, callers, siblings, tests, and generated/manual boundary before editing. For a defect, repair the narrowest shared owner proved by the reproducer. Preserve accepted behavior, cleanup, authority, and protected-domain constraints; remove replaced paths only when compatibility does not require them.

Treat task paths and symbols as navigation anchors; current canonical source
determines placement. Adapt local placement drift while accepted behavior,
ownership, proof, and risk stay unchanged. If one changes, reopen only its
narrowest decision owner.

Accepted behavior remains incomplete when replaced by a placeholder, temporary
hardcoding, TODO, mock success, or undeclared `v1`. Return a blocker or reopen
scope instead of declaring the smaller outcome done.

Before a production Go edit, apply `go-coder`; route an unknown cause to `go-systematic-debugging` and test-only work to `go-test-implementation`. During implementation, `go-coder` owns the change. Apply only the Go, contract, data, lifecycle, or delivery methods triggered by the changed surface as review lenses under Candidate Acceptance below. Missing accepted policy reopens its narrow upstream decision owner.

### Local Execution

For direct work, the root edits the assigned checkout, performs one coherent self-review of the bounded diff, and runs the validation matrix's smallest matching proof. Do not create a durable execution control, commit, worktree, Worker, ledger, or reviewer just to record a local reversible change.

### Worker Execution

Keep direct work root-local. For structured or orchestrated implementation, delegate each ready ledger task by default to one harness-native implementation Worker in an isolated worktree — a Codex App Worker with managed worktree, or a Claude Code background subagent with worktree isolation, per [Agent Harness](../../agent-harness.md#control-map) — once its behavior, mechanism, ownership, editable boundary, proof, and stop condition are closed. Dependencies schedule Workers sequentially; a planned wave with positive independence permits concurrent dispatch. Sequential work is not a reason for root-local implementation. Root-local implementation is limited to direct work or an unavailable native Worker control; reclassify a structured task as direct only when it satisfies the router's current direct criteria. Create one root durable execution control — a Codex Goal, or `/goal` plus the task list in Claude Code ([Claude Code Goal Mechanics](../../agent-harness.md#claude-code-goal-mechanics)) — only for a genuinely multi-step or resumable outcome.

Before dispatching from uncommitted accepted input, identify the source
checkout and authorized paths and inspect its current diff/status. Record a
base only when the candidate will leave that checkout or several provisional
deltas must be compared. Do not stash, clean, ignore, or mutate user changes.
Keep one write Worker per ledger task; several write Workers may run only as
members of a positively independent planned wave. A Worker receives an outcome-first brief
with editable boundaries, current facts, success criteria, focused proof, and
a real stop condition.

Select the Worker's tier from the task class, because the tier is where the
lane's reasoning effort is fixed: mechanical work whose route and proof are
already decided, ordinary implementation, or a protected domain. The harness
document owns the tier names and how a harness expresses them.

#### Route Discovery Stays Root-Local

Delegate execution, never discovery. A task is delegable when the route is
already known and only the doing remains. A task whose next step depends on
what the previous step turns up is not delegable at any tier: defect diagnosis,
a failing check whose cause is unidentified, a performance question without a
measurement, and any work whose plan is written by its own intermediate
findings. Keep that root-local until the route is known, then delegate what
remains.

The reason is structural, not stylistic. A Worker starts from a fresh context:
it receives its brief, the repository instructions, and the tree, and nothing
else. It cannot see the root's reasoning, the command output the root already
read, or the hypothesis the root already discarded. Every handoff of a
discovery loop therefore pays to rediscover what the root already knew, and
returns a summary that has dropped exactly the detail the next step needed —
the error text, the stack, the surprising state.

This is also why a brief is not a task title. Whatever the root has already
established has to travel inside the brief, because nothing else crosses:
the reproducer, the located owner, the cause when it is known, the expected
behavior, and the exact proof command. A brief that says what to achieve
without what is already known is a request to redo the root's work.

Splitting one discovery loop across several Workers compounds the same loss at
every boundary and produces changes nobody can attribute. When a task turns out
to be discovery rather than execution after dispatch, that is a real blocker:
the Worker returns it, and the root resolves the route before delegating again.

#### Correction Loop

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

#### Scope Lock

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

For every Worker task, the root explicitly selects and passes the best-suited available model and a task-matched reasoning effort through the current harness's supported controls; never inherit a controllable default and never ask the user to choose. [Agent Harness](../../agent-harness.md#model-and-effort-selection) owns model tiers, effort baselines, and evaluation rules.

#### Progress

Progress is a material candidate transition toward an open finding or new
evidence that changes a causal hypothesis's disposition. Repeated tool or
status activity, a materially repeated candidate, acknowledgement without a
correction, and the same proof observable under the same hypothesis all end the
write lane at the preceding frozen candidate with the cumulative evidence and
an honest blocker. Activity is repeated when its effective inputs, causal
hypothesis, and expected observable already have a disposition, even when the
command changes. A returned correction may continue only through the
Diagnostic Gate below. Worker replacement is reserved for an execution stall
that produces no new native turn or item after continuation, or for an
invalidated base; it continues the same brief from the frozen candidate. Keep
one write Worker active per task and follow native completion and status
events. When those events cannot distinguish active repetition from progress,
only returned-candidate convergence is verified.

For a planned write wave, every member starts from the same accepted integrated base and every returned result remains provisional. Assemble only bounded deltas into a frozen candidate.

When a failure is isolated and the reviewed independence basis still holds, shrink the wave to the proven passing subset. Review, prove, integrate, and accept that subset while the failed member and its dependents remain provisional. Keep coupled members together when the failure crosses an interface, invariant, generated/manual authority, mutable resource, or proof precondition. Start later work only from an accepted commit or tree whose accepted subset satisfies its dependencies.

### Candidate Acceptance

#### Monotonic Acceptance

The root is an acceptance authority. Each Worker return first passes Scope Lock
plus ownership, mergeability, and proof-provenance intake. The first
acceptance-ready candidate becomes the frozen baseline for exactly one full
acceptance review and its initial mapped proof. That review freezes one complete
finite finding set containing only candidate-caused regressions, concrete
violations of accepted criteria or repository invariants, and proof missing
from an accepted claim. All other observations retain observation status.

Correction verification covers the open findings, the delta from the frozen
baseline, and proof invalidated by that delta; unchanged bytes retain their
prior disposition. Adopt a correction only when it closes or disproves at least
one finding, reopens none, introduces no regression, stays in scope, and
preserves passing proof. Adoption makes the correction the current frozen
baseline and the open set a strict subset. An empty set plus passing mapped
proof accepts the candidate immediately.

#### Diagnostic Gate

A correction that introduces a regression, reopens a finding, or fails to
shrink the open set has one disposition: reject the entire delta and retain the
preceding frozen baseline. Its bytes serve only as diagnosis evidence;
rejection removes write authority for that finding. Read-only diagnosis against
the retained baseline restores write authority only when evidence independent
of the rejected bytes falsifies the rejected causal hypothesis and establishes
a different causal owner, expected observable, and bounded edit for the same
finding. Otherwise the finding returns the honest blocker.

Concrete evidence first available after the full review that proves a critical
security breach, data loss or corruption, or critical violation of an accepted
public contract stops correction and reopens that evidence's owner.

The local repository default/main is the authoritative integration branch unless the user names another persistent branch. Run mapped claim-scoped proof on the reviewed candidate, integrate only the accepted delta, and confirm the authoritative diff still contains that candidate without unrelated changes. If integration changes relevant content, validate the affected claims before acceptance. Use a commit/tree identity only when proof crosses a checkout or integration boundary; never hash individual files or specifications for this purpose. Do not mutate unrelated dirty state to force integration. Remote push requires separate authorization.

### Immutable Evidence

Proof belongs to the relevant content and claim it exercised. Record the command, relevant environment/preconditions, result, and gaps. Attach a commit/tree identity only when reusing proof across a checkout or integration boundary; the current bounded diff is sufficient for local work. Reuse proof while the relevant content, preconditions, claim scope, provenance, and risk surface remain unchanged.

## Review

Review the changed outcome for correctness, affected error/context/resource behavior, generated authority, security/data/rollout risk when triggered, unnecessary abstraction, stale replacements, and proof adequacy. Candidate Acceptance owns finding admissibility: surrounding and transitive context informs the candidate judgment, while unrelated or pre-existing defects, style preferences, and unproven suspicions remain observations. Resolve repository-answerable uncertainty before asking another actor.

## Validate

Apply the repository [Task Contract](../../../AGENTS.md#task-contract) and
[Validation Matrix](../../../AGENTS.md#validation-matrix). For instruction-only
work, the default proof is `git diff --check`; wording does not need a CI gate.

Performance validation is triggered by an accepted performance claim, budget,
or materially affected hot path; it is not a ceremony for every change. If the
workload, measured boundary, fixture shape, testbed, or budget is still absent,
reopen the narrow Performance Decision owner instead of inventing a benchmark
after implementation. Use the repository commands and artifact conventions in
[Benchmarking](../../benchmarking.md) once those inputs exist.

Do not use Worker prose, stale logs, unrelated green checks, or a broad command that misses the changed behavior as proof. When a required check cannot run, record the command, reason, narrower evidence, and unverified remainder.

## Close Out

State what changed, the important behavior consequence, proof actually run, and remaining gap or reopen owner. Apply the [Task Contract](../../../AGENTS.md#task-contract) to readiness and completion language.

## Stop Rule

Finish direct work when the bounded outcome is self-reviewed and every stated claim has matching current evidence or an honest gap. Worker execution finishes when the frozen finding set is empty and its mapped proof passes, or with the honest blocker when an admissible finding cannot close. A changed decision or unavailable evidence reopens its narrowest upstream owner; after sufficient proof, closeout is the only authorized next action.
