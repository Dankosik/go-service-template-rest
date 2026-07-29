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

Inspect current diff/status and trace the accepted observable through its
production entry point and wiring, owning code, every relevant caller and
sibling that can reach the same behavior, tests, and generated/manual boundary
before editing. For a defect, use the reproducer to identify the violated
contract and fix the narrowest shared causal owner reached by every affected
path; keep a leaf-specific fix only when evidence proves the cause is
leaf-specific.

Preserve unrelated user work, accepted unchanged behavior, authoritative source
ownership, and protected-domain constraints. When a canonical source owns
derived output, change it first and regenerate. When behavior or wiring is
replaced, remove its superseded code, wiring, tests, configuration, generated
output, and documentation unless a named compatibility requirement still
exercises them.

Treat task paths and symbols as navigation anchors; current canonical source
determines placement. Adapt local placement drift while accepted behavior,
ownership, proof, and risk stay unchanged. If one changes, reopen only its
narrowest decision owner.

Accepted behavior is complete only on the real production path; a placeholder,
temporary hardcoding, TODO, mock success, or undeclared `v1` is a blocker unless
the accepted outcome explicitly permits it.

Before a production Go edit, apply `go-coder`; route an unknown cause to
`go-systematic-debugging` and test-only work to `go-test-implementation`. These
skills supply methods under this phase and do not own completion. Apply only the
Go, contract, data, lifecycle, or delivery methods triggered by the changed
surface as review lenses under [Review](#review) below. Missing accepted policy
reopens its narrow upstream decision owner.

### Local Execution

Root-local execution covers direct work and ready ledger tasks routed here by
[Worker Execution](implementation-worker-execution.md). The root edits the
assigned checkout, performs one coherent self-review of the bounded diff, and
runs the Validation Matrix's smallest matching proof. The bounded working-tree
diff is the complete execution record for direct work; ledger work also retains
its accepted task entry. Reopen the path only when evidence changes risk,
ownership, reversibility, or proof.

### Worker Execution

For ledger work, read
[Implementation Worker Execution](implementation-worker-execution.md) before
selecting or operating a Worker. That branch defines the execution-need trigger
and owns carrier-specific dispatch, dirty-state protection, Scope Lock, planned
waves, correction continuity, rejected-delta handling, candidate intake, and
candidate handoff. This phase retains completion, acceptance, and integration
authority.

### Immutable Evidence

Proof belongs to the relevant content and claim it exercised. Record the command, relevant environment/preconditions, result, and gaps. Attach a commit/tree identity only when reusing proof across a checkout or integration boundary; the current bounded diff is sufficient for local work. Reuse proof while the relevant content, preconditions, claim scope, provenance, and risk surface remain unchanged.

## Review

Review the bounded diff and resulting production behavior for correctness;
affected error, context, ownership, concurrency, resource, and lifecycle
behavior; canonical/generated authority; triggered security, data, and rollout
risk; unnecessary machinery; stale replacement surfaces; and proof adequacy.
Surrounding and transitive context informs the bounded judgment, while unrelated
or pre-existing defects, style preferences, and unproven suspicions remain
observations. Resolve repository-answerable uncertainty before asking another
actor.

## Validate

Reconcile the accepted outcome in both directions: each accepted changed or
deliberately unchanged behavior maps to the retained delta or current
authoritative evidence that no implementation change is required, and to
current proof or an explicit unverified remainder; every retained change and
completion claim maps back to an accepted criterion. Use the smallest matching
[Validation Matrix](../../../AGENTS.md#validation-matrix) proof.

Apply the [Stop Rule](#stop-rule)'s production-path proof criterion before
treating implementation evidence as completion.

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

Close the outcome only when every accepted changed or deliberately unchanged
behavior has a terminal disposition; the bounded accepted outcome—including
any required real production wiring, shared-cause repair, replacement cleanup,
and canonical/generated synchronization—is complete; the retained task delta
contains no unrelated change and has been reviewed against every triggered
risk; and each completion claim has current proof of equal scope. A
production-path claim must exercise the relevant path or separately prove both
its owning behavior and production wiring.

If implementation is complete but required proof is unavailable, stop as
`implementation complete; verification incomplete`. Name the unverified claim,
the narrower evidence obtained, and the next proof or reopen owner; do not claim
outcome completion or readiness.

Worker execution uses the same criterion. Accept a candidate immediately when
its current admissible finding set is empty and mapped proof passes. Proof
unavailable for an external or environmental reason yields partial
verification, not another correction loop. A changed decision reopens its
narrowest upstream owner; after sufficient proof, closeout is the only
authorized next action.
