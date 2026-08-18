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
- A closeout that claims outcome completion only with qualifying proof;
  otherwise an honest named blocker. When implementation is complete and only
  required proof is unavailable, report that blocker as `implementation
  complete; verification incomplete`.

## Acceptance-Unit Closure

For orchestrated ledger work, the bound Acceptance-Unit Lead is the acceptance
owner and current phase root. For direct work, the current root is the
acceptance owner.

For ledger work, select one ready acceptance unit from the authoritative ledger
and keep its recorded boundary fixed through implementation, bounded
acceptance-owner review, mapped validation, any triggered fresh one-shot review,
acceptance, and the immediate persisted transition defined by [Artifact
Model](../shared/artifact-model.md#minimal-status). That transition is the
unit's completion criterion.

Only after that transition may task selection re-evaluate `Depends on` and
start a dependent unit or new planned wave. Members already running inside the
same accepted planned wave remain provisional and may continue. When the
transition cannot occur, the current unit remains selected for the phase-owned
correction, reopen, or blocker disposition.

When the current session stops at this transition and later implementation work
remains, apply the shared [Implementation Entry And Continuation
Handoff](../shared/resume-and-handoff.md#implementation-entry-and-continuation-handoff)
so the next prompt carries the next ready unit and the receiving session chooses
its execution strategy from current evidence.

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

Treat task paths, symbols, and an accepted Go Ownership file map as
evidence-backed starting points, not immutable prescriptions; current canonical
source and the code being written determine final placement. When the real code
gives one file independently changing responsibilities, obscures an ownership
or lifecycle transition, or exposes a materially clearer smaller placement,
revise the file map and make the smallest behavior-preserving move, split,
merge, or deletion in this acceptance unit. File length is a signal to inspect
those pressures, not a split criterion. Keep that adaptation in Implementation
when accepted behavior, semantic owner, dependency direction, generated/manual
authority, exported surface, proof strategy, and risk stay unchanged. If one
changes, reopen only its narrowest decision owner. Do not retain a known-poor
shape merely because it matches the pre-code plan, and do not use adaptation to
widen into unrelated cleanup.

A fixed ledger or file map does not suppress current evidence that invalidates
its dependency or implementation-source disposition. When the real code or
current dependency evidence exposes a viable reuse or generated-source option
that could replace a planned non-trivial custom mechanism, reopen Go Ownership
instead of preserving the custom implementation merely because Planning pinned
it.

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

Local execution covers direct work and ready acceptance units routed here by
[Worker Execution](implementation-worker-execution.md). A direct-work
acceptance owner edits the assigned checkout. A ledger Acceptance-Unit Lead
follows the Role Tree's [Implementation Write
Boundary](implementation-worker-execution.md#implementation-write-boundary),
performs one coherent self-review of the bounded integrated diff, and runs the [Validation
Matrix](#validation-matrix)'s smallest matching proof. The bounded working-tree
diff is the complete execution record for direct work; ledger work also retains
its accepted task entry. The acceptance owner
accepts the fixed candidate when the Stop Rule passes and any triggered
[independent implementation review](../shared/implementation-review.md) has
returned `PASS`.
Reopen the path only when evidence changes risk, ownership, reversibility, or
proof.

### Worker Execution

For ledger work, read
[Implementation Worker Execution](implementation-worker-execution.md) and bind
the current session's role from its [Execution Role
Tree](implementation-worker-execution.md#execution-role-tree) before selecting
or operating an internal Worker lane. That branch owns the Acceptance-Unit
Lead's Slice DAG scheduling and delegation, isolated lane eligibility, dirty-state
protection, Scope Lock, correction continuity, rejected-delta handling,
candidate intake, and handoff. The Lead applies this phase and retains decision,
correction-routing, acceptance, and integration authority.

### Immutable Evidence

Proof belongs to the relevant content and claim it exercised. Record the command, relevant environment/preconditions, result, and gaps. Attach a commit/tree identity only when reusing proof across a checkout or integration boundary; the current bounded diff is sufficient for local work. Reuse proof while the relevant content, preconditions, claim scope, provenance, and risk surface remain unchanged.

### Proof Ownership

Assign one final owner to each deterministic gate for an acceptance unit. A
Worker owns iterative focused checks while changing its candidate. The final
owner runs the gate once on the exact accepted tree: use the Worker's receipt
when its tree and preconditions cross into integration unchanged; otherwise the
acceptance owner runs the gate after integration. The acceptance owner validates
the receipt, tree identity, preconditions, scope, and claimed observable instead
of automatically repeating the command. A reviewer runs only a missing or adversarial falsifier
needed for its independent question. A deterministic result retains its
disposition while tree, preconditions, command, and claim are unchanged.

## Review

Review the bounded diff and resulting production behavior for correctness;
affected error, context, ownership, concurrency, resource, and lifecycle
behavior; canonical/generated authority; triggered security, data, and rollout
risk; unnecessary machinery; duplicated ownership of one live policy; stale
replacement surfaces; and proof adequacy.
For a substantial Go change, compare every added or materially changed file
with its design's inverse file map and the repository's [file-granularity
contract](../../project-structure-and-module-organization.md#4-file-naming-and-granularity),
then re-evaluate that map against the actual code. A justified local adaptation
becomes the candidate's effective file map; literal conformity to a stale plan
is not the goal, while an unexplained deviation remains a finding.
Mixed file ownership, duplicated live policy without its required parity proof,
shared fixtures at the wrong owner, and hidden temporal coupling are ownership
or readability findings, not style preferences; passing functional checks does
not close them. File length alone remains an observation.
Surrounding and transitive context informs the bounded judgment, while unrelated
or pre-existing defects, style preferences, and unproven suspicions remain
observations. Resolve repository-answerable uncertainty before asking another
actor.

## Validate

Reconcile the accepted outcome in both directions: each accepted changed or
deliberately unchanged behavior maps to the retained delta or current
authoritative evidence that no implementation change is required, and to
current proof or an explicit unverified remainder; every retained change and
completion claim maps back to an accepted criterion. Use the
[Validation Matrix](#validation-matrix)'s smallest matching proof.

### Deployment And Remote-Proof Preflight

Before a deployment or other slow, costly, quota-bound, or externally
mutating proof action, trace its exact artifact and data path through the target
observable. Close every material prerequisite or limit that current repository
evidence, canonical configuration, provider contracts, deterministic
calculation, or a faithful local rehearsal can falsify. Record the
representative input envelope, remaining target-only uncertainty, expected
success and safe-failure signals, and recovery boundary. The action is ready
only when no cheaper current evidence is likely to change its artifact,
configuration, sequence, or mechanism.

A failed external action does not authorize patch-and-redeploy. Preserve passed
gates, inspect the complete failure boundary, reopen the narrowest invalidated
decision or proof owner, and rerun only invalidated cheaper checks. Retry only
after the preflight closes again and the next attempt exercises only named
residual target-only uncertainties. Deployment must not be the first discovery
point for a condition that cheaper current evidence could expose.

### Validation Matrix

| Changed surface | Default proof |
| --- | --- |
| Docs or instructions | `git diff --check` |
| Local Go behavior | Focused package/test proof; changed-code lint when useful |
| Concurrency/lifecycle | Focused behavior plus race/liveness proof |
| Performance claim | The matching [benchmark level](../../benchmarking.md), equivalent workload/testbed evidence, and independent correctness proof |
| OpenAPI, sqlc, migration, generated source | Canonical generation/drift and affected runtime proof |
| Defect crossing a service, client, or managed-dependency boundary | Correlated evidence from each implicated side: what the caller emitted, what this service observed and returned, and what the next hop recorded for the same correlation id |
| Security, deployment, cross-service or release | The matching protected-domain and integrated proof |
| Publication, CI parity, or broad cross-cutting change | `check-full`, `ci-local`, `pr-check`, container, or security suites only when the claim needs them |

`make check` is a broad local baseline rather than the default edit loop. For
docs-only work, use the matching docs or instruction checks; service tests and
broad suites run only when the claim requires them.

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

## Independent Implementation Review

After local execution or Worker integration produces a fixed acceptance unit
that passes bounded acceptance-owner review and mapped validation, load [Review
Independence](../shared/review-independence.md) to decide whether review is
triggered. When it applies, load the [Independent Implementation
Review](../shared/implementation-review.md) conditional branch. The acceptance owner accepts
the fixed candidate only on `PASS`; `FAIL` remains in phase-owned correction or
reopen. `NEEDS_PARENT` returns to the acceptance owner for the bottom-up
resolution ladder and becomes `implementation complete; verification
incomplete` only when no evidence-changing in-scope remedy remains, without
accepting the unit or claiming outcome completion.

## Close Out

State what changed, the important behavior consequence, proof actually run, and remaining gap or reopen owner. Apply the [Task Contract](../../../AGENTS.md#task-contract) to readiness and completion language. Move each durable decision to its canonical owner with the provenance and reopen condition required by [Artifact Model](../shared/artifact-model.md#resume-order) before removing the completed bundle.

## Stop Rule

Close the outcome only when every accepted changed or deliberately unchanged
behavior has a terminal disposition; the bounded accepted outcome—including
any required real production wiring, shared-cause repair, replacement cleanup,
and canonical/generated synchronization—is complete; the retained task delta
contains no unrelated change and has been reviewed against every triggered
risk; and each completion claim has current proof of equal scope. A
production-path claim is complete only when current proof would fail if either
its owning behavior or production wiring were absent or wrong. Prefer one
exercise of the relevant observable path. Split proof qualifies only when it
separately exercises the owner and the production wiring or composition and
establishes that together they realize that same path; otherwise implementation
may be complete, but production-path verification remains incomplete.

If implementation is complete but required proof is unavailable, stop as
`implementation complete; verification incomplete`. Name the unverified claim,
the narrower evidence obtained, and the next proof or reopen owner; do not claim
outcome completion or readiness. This is the `blocked` lifecycle state for a
durable goal or artifact, not a third completion state.

For a ledger acceptance unit, these criteria determine whether the acceptance owner may
accept the fixed candidate and what a triggered reviewer must falsify. A changed
decision reopens its narrowest upstream owner; after acceptance, completing
[Acceptance-Unit Closure](#acceptance-unit-closure) is the only authorized next
action before closeout or later work.
