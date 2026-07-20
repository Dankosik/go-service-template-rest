# Implementation / Validation / Closeout

Implement the accepted outcome at its narrowest owner, then make only the claim the current evidence supports. Direct work is local by default; delegation is an optional execution tool.

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

Before a production Go edit, apply `go-coder`; route an unknown cause to `go-systematic-debugging` and test-only work to `go-test-implementation`. During implementation, `go-coder` owns the change. Apply only the Go, contract, data, lifecycle, or delivery methods triggered by the changed surface as review lenses under Candidate Acceptance below. Missing accepted policy reopens its narrow upstream decision owner.

### Local Execution

For direct work, the root edits the assigned checkout, performs one coherent self-review of the bounded diff, and runs the validation matrix's smallest matching proof. Do not create a Goal, commit, worktree, Worker, ledger, or reviewer just to record a local reversible change.

### Optional Worker Execution

Keep a clear sequential outcome root-local even when it is long-running or spans multiple files. Use a native Codex App Worker and managed worktree only when resumability, dirty-checkout isolation, safe parallelism, or materially separate context outweighs dispatch, worktree, fan-in, and acceptance cost, or when the user explicitly delegates implementation. Create one root Goal only for a genuinely multi-step or resumable outcome.

Before dispatching from uncommitted accepted input, run `bash scripts/dev/codex-worktree-preflight.sh <selected-git-top-level>` against the selected checkout. It is read-only and fails closed on oversized transfer input; do not stash, clean, ignore, or mutate user changes to pass it. Keep one write Worker per outcome. A Worker receives an outcome-first brief with editable boundaries, current facts, success criteria, focused proof, and a real stop condition.

For every Worker task, the root explicitly selects and passes the best-suited available model and reasoning effort through supported App controls; never inherit an App default or ask the user to choose when those controls exist. This is the user's standing request. Choose model and effort independently: Luna for clear mechanical work, Terra for ordinary implementation, and Sol for complex or high-consequence work, using the lowest effort likely to succeed. Existing eval evidence may inform the choice but is never a dispatch prerequisite.

Ordinary implementation-owned findings return together to the same Worker and managed worktree. Each correction brief contains the complete current finding set and requires every finding to return closed, disproved, or genuinely blocked with evidence. A repeated material candidate, an acknowledgement-only correction, or the same proof observable failing after one correction under the same causal hypothesis establishes evidence-backed no progress: preserve the candidate and cumulative evidence, then replace the Worker or switch to a materially different repair route. Treat an `inProgress` task that produces no new turn or item after one continuation as stalled. Oscillation, an exhausted repair route, or an invalidated base likewise requires replacement or rerouting. Keep only one write Worker active for the outcome and follow native completion and status events instead of actively polling or narrating unchanged state.

For a planned write wave, every member starts from the same accepted integrated base and every returned result remains provisional. Assemble only bounded deltas into a frozen candidate.

When a failure is isolated and the reviewed independence basis still holds, shrink the wave to the proven passing subset. Review, prove, integrate, and accept that subset while the failed member and its dependents remain provisional. Keep coupled members together when the failure crosses an interface, invariant, generated/manual authority, mutable resource, or proof precondition. Start later work only from an accepted commit or tree whose accepted subset satisfies its dependencies.

### Candidate Acceptance

On each Worker return, the root performs only boundary intake for scope, ownership, mergeability, and proof provenance. The root performs one full acceptance review on the frozen candidate across all triggered lenses, returns one combined set of currently detectable evidence-backed compatible findings, and maps claims with the same preconditions to one exact proof command. Re-review only the correction and affected surfaces; an unchanged surface keeps its prior disposition unless new evidence changes its risk.

The local repository default/main is the authoritative integration branch unless the user names another persistent branch. Freeze the reviewed candidate as a commit or tree and run every mapped claim-scoped proof command once on that exact state. Integrate only the accepted delta, prefer a byte-identical fast-forward when valid, verify the resulting authoritative tree identity, and then accept it or dispatch later work. If integration changes the tree, treat it as a new candidate and validate the affected claims before acceptance. Do not mutate unrelated dirty state to force integration. Remote push is outside this integration rule and requires separate authorization.

### Immutable Evidence

Proof belongs to the exact tree and claim it exercised. Record the commit or tree identity, command, relevant environment/preconditions, result, and gaps. Exact successful Worker proof remains valid after a byte-identical fast-forward or integration. Rerun only when the tree, relevant environment or precondition, claim scope, provenance, or risk surface changed. A local direct change does not need a commit before proof.

## Review

Review the changed outcome for correctness, affected error/context/resource behavior, generated authority, security/data/rollout risk when triggered, unnecessary abstraction, stale replacements, and proof adequacy. A style preference or unproven suspicion is not a correction finding. Resolve repository-answerable uncertainty before asking another actor.

## Validate

Map each claim to current proof of equal scope. Use the smallest matching validation:

| Surface | Proof |
| --- | --- |
| Docs/instructions | `git diff --check` and the relevant instruction gate |
| Local Go behavior | Focused package or regression proof; changed-code lint when useful |
| Concurrency/lifecycle | Focused behavior plus race/liveness proof |
| API, sqlc, migrations, generated source | Canonical generation/drift plus affected runtime proof |
| Security, deployment, cross-service | Matching protected-domain and integrated proof |
| Publication, CI parity, broad cross-cutting work | `check-full`, `ci-local`, `pr-check`, container, or security suites only when that claim requires them |

Do not use Worker prose, stale logs, unrelated green checks, or a broad command that misses the changed behavior as proof. When a required check cannot run, record the command, reason, narrower evidence, and unverified remainder.

## Close Out

State what changed, the important behavior consequence, proof actually run, and remaining gap or reopen owner. Use complete/fixed/ready only when the matching current evidence supports the full claim.

## Stop Rule

Finish when the bounded outcome is reviewed, every stated claim has matching current evidence or an honest gap, and no unowned implementation decision remains. Reopen the narrowest upstream owner only when a decision or unavailable evidence must change; do not manufacture workflow work after proof is sufficient.
