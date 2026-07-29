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
symbol presence, an unrelated passing check, and an implementation summary are
not completion evidence.

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

Accepted behavior is complete only on the real production path; a placeholder,
temporary hardcoding, TODO, mock success, or undeclared `v1` is a blocker unless
the accepted outcome explicitly permits it.

Before a production Go edit, apply `go-coder`; route an unknown cause to `go-systematic-debugging` and test-only work to `go-test-implementation`. During implementation, `go-coder` owns the change. Apply only the Go, contract, data, lifecycle, or delivery methods triggered by the changed surface as review lenses under [Review](#review) below. Missing accepted policy reopens its narrow upstream decision owner.

### Local Execution

For direct work, the root edits the assigned checkout, performs one coherent
self-review of the bounded diff, and runs the Validation Matrix's smallest
matching proof. The bounded working-tree diff is its complete execution record
unless current evidence invalidates the router's direct criteria.

### Worker Execution

For every structured or orchestrated ledger task, read
[Implementation Worker Execution](implementation-worker-execution.md) before
choosing the Worker route or its unavailable-control fallback, dispatching,
correcting, accepting, or integrating a candidate. That branch owns execution
readiness, unrelated dirty-state protection, model and effort selection, Scope
Lock, planned waves, monotonic acceptance, rejected-delta handling, and local
integration.

### Immutable Evidence

Proof belongs to the relevant content and claim it exercised. Record the command, relevant environment/preconditions, result, and gaps. Attach a commit/tree identity only when reusing proof across a checkout or integration boundary; the current bounded diff is sufficient for local work. Reuse proof while the relevant content, preconditions, claim scope, provenance, and risk surface remain unchanged.

## Review

Review the changed outcome for correctness, affected error/context/resource behavior, generated authority, security/data/rollout risk when triggered, unnecessary abstraction, stale replacements, and proof adequacy. Surrounding and transitive context informs the bounded judgment, while unrelated or pre-existing defects, style preferences, and unproven suspicions remain observations. Resolve repository-answerable uncertainty before asking another actor.

## Validate

Validate every stated claim against the repository
[Task Contract](../../../AGENTS.md#task-contract) and the smallest matching
[Validation Matrix](../../../AGENTS.md#validation-matrix) proof.

Performance validation is triggered by an accepted performance claim, budget,
or materially affected hot path; it is not a ceremony for every change. If the
workload, measured boundary, fixture shape, testbed, or budget is still absent,
reopen the narrow Performance Decision owner instead of inventing a benchmark
after implementation. Use the repository commands and artifact conventions in
[Benchmarking](../../benchmarking.md) once those inputs exist.

Qualifying proof is current, claim-scoped, and matched to the changed surface;
summaries, stale evidence, and checks that miss the stated claim do not qualify.
When a required check cannot run, record the command, reason, narrower evidence,
and unverified remainder.

## Close Out

State what changed, the important behavior consequence, proof actually run, and remaining gap or reopen owner. Apply the [Task Contract](../../../AGENTS.md#task-contract) to readiness and completion language.

## Stop Rule

Finish direct work when the bounded outcome is self-reviewed and every stated claim has matching current evidence or an honest gap. Worker execution finishes when the frozen finding set is empty and its mapped proof passes, or with the honest blocker when an admissible finding cannot close. A changed decision or unavailable evidence reopens its narrowest upstream owner; after sufficient proof, closeout is the only authorized next action.
