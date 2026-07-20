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

Before a production Go edit, apply `go-coder`; route an unknown cause to `go-systematic-debugging` and test-only work to `go-test-implementation`. Apply only the Go, contract, data, lifecycle, or delivery methods triggered by the changed surface.

### Local Execution

For direct work, the root edits the assigned checkout, performs one coherent self-review of the bounded diff, and runs the validation matrix's smallest matching proof. Do not create a Goal, commit, worktree, Worker, ledger, or reviewer just to record a local reversible change.

### Optional Worker Execution

Use a native Codex App Worker and managed worktree only when the work is long-running, resumable, isolated from a dirty checkout, independently parallelizable, needs separate context, or the user explicitly delegates it. Create one root Goal only for that genuinely multi-step or resumable outcome.

Before dispatching from uncommitted accepted input, run `bash scripts/dev/codex-worktree-preflight.sh <selected-git-top-level>` against the selected checkout. It is read-only and fails closed on oversized transfer input; do not stash, clean, ignore, or mutate user changes to pass it. Keep one write Worker per outcome. A Worker receives an outcome-first brief with editable boundaries, current facts, success criteria, focused proof, and a real stop condition.

The root inspects every delegated diff and proof before accepting it. Return all currently detectable evidence-backed compatible findings together. Re-review only the correction and affected surfaces; do not repeat an unchanged correction loop or launch a ceremonial reviewer.

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
