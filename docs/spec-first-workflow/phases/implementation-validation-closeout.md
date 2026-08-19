# Implementation / Validation / Closeout

## Acceptance Posture

Act as a contract closer. Seek the earliest proven state that satisfies the
accepted outcome while preserving already-valid behavior, decisions, candidate
state, and proof. Spend an edit only on an accepted criterion or current open
finding; all other discoveries retain observation status. Acceptance is
terminal: close out when the criteria and mapped proof pass.

Reason backward from the observable outcome: identify what must be true, its
narrowest implementation owner, the wiring that exposes it on the real path,
and the check that would fail if it were absent or wrong. The [Evidence
Contract](#evidence-contract) owns what qualifies; task status, file or symbol
presence, an unrelated passing check, and an implementation summary do not.

## Read When

- Structured or orchestrated work enters Implementation.
- Direct work needs non-obvious validation, deployment or remote proof,
  independent implementation review, integration, or blocked closeout.

## Inputs

- Accepted inline outcome or the smallest relevant durable artifact.
- Current repository state, including unrelated user changes.
- Canonical sources and repository-native proof commands for changed surfaces.

## Outputs

- The bounded code, tests, configuration, generated output, or documentation change.
- Claim-scoped review and proof evidence.
- A closeout that claims outcome completion only under the [Evidence
  Contract](#evidence-contract); otherwise its evidence-owned blocker.

## Acceptance-Unit Closure

For direct work, the current root is the acceptance owner and implementation
writer. Every structured or orchestrated implementation unit, whether fixed
inline or selected from a ledger, binds exactly one Acceptance-Unit Lead as its
acceptance owner: the current root binds that role for structured work, while
the Ledger Orchestrator dispatches it for orchestrated work.

For ledger work, select one ready acceptance unit from the authoritative ledger
and keep its recorded boundary fixed through implementation, bounded
acceptance-owner review, mapped validation, any triggered fresh one-shot review,
acceptance, and the immediate persisted transition defined by [Artifact
Model](../shared/artifact-model.md#minimal-status) and the [Planning Ledger
Contract](planning/ledger-contract.md#implementation-transitions). That transition is the
unit's completion criterion.

Only after that transition may task selection re-evaluate `Depends on` and
start a dependent unit or new planned wave. Members already running inside the
same accepted planned wave remain provisional and may continue. When the
transition cannot occur, the current unit remains selected for the phase-owned
correction, reopen, or blocker disposition.

For a fixed inline unit, keep the same boundary through implementation, proof,
review, and closeout; there is no synthetic ledger transition. When the current
session stops at a ledger transition and later implementation work
remains, apply the shared [Implementation Entry And Continuation
Handoff](../shared/implementation-handoff.md#implementation-entry-and-continuation-handoff)
so the next prompt carries the next ready unit and the receiving session chooses
its execution strategy from current evidence.

## Implement

Direct work follows root [Direct Work](../../../AGENTS.md#direct-work).
Structured and orchestrated work binds one role from the [Execution Role
Tree](implementation-worker-execution.md#execution-role-tree); the bound role's
skill loads its method. Apply only the Go and domain skills selected by [Go
Change Surface](../../../AGENTS.md#go-change-surface). This phase retains the
fixed unit, integration, acceptance, and closeout; evidence changing accepted
behavior, architecture, ownership, proof strategy, or rollout reopens its
narrowest upstream owner.

## Evidence Contract

Evidence qualifies only when it is current, claim-scoped, and would fail if the
claimed behavior or required production wiring were absent or wrong. Prefer one
exercise of the observable path. Split proof qualifies only when it separately
exercises the owner and wiring and establishes that together they realize the
same path.

Record the command, relevant environment and preconditions, result, and gaps.
Attach a commit or tree identity only across a checkout or integration boundary;
the current bounded diff is sufficient for local work. Reuse proof while its
content, preconditions, claim, provenance, and risk surface remain unchanged.

Assign one final owner to each deterministic gate. A Worker owns iterative
focused checks; the final owner uses its receipt when the same tree and
preconditions cross integration unchanged, otherwise the acceptance owner runs
the gate on the integrated tree. The acceptance owner validates provenance,
scope, identity, preconditions, and the claimed observable instead of
automatically repeating the command. A reviewer runs only a missing or
adversarial falsifier for its independent question.

When required proof cannot run, record the command, reason, narrower evidence,
and unverified remainder. Stop as `implementation complete; verification
incomplete`; do not accept the unit or claim outcome completion or readiness.
This is the durable `blocked` state, not a third completion state.

## Review

Self-review the bounded diff and resulting production path against every
accepted criterion and triggered risk. Use the matching skills as review lenses;
they own detailed correctness, ownership, lifecycle, security, data, rollout,
cleanup, and proof checks. Keep unrelated or pre-existing defects and unproven
suspicions as observations.

## Validate

Apply `go-verification-before-completion` to intended readiness or completion
claims and use repository [Validation
Routing](../../build-test-and-development-commands.md#validation-routing) to
select the smallest matching proof. The Evidence Contract decides whether the
result permits acceptance or only an explicit unverified remainder.

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

## Independent Implementation Review

After local execution or Worker integration produces a fixed acceptance unit
that passes bounded acceptance-owner review and mapped validation, load [Review
Independence](../shared/review-independence.md) to decide whether review is
triggered. When it applies, load the [Independent Implementation
Review](../shared/implementation-review.md) conditional branch. The acceptance owner accepts
the fixed unit candidate only on `PASS`; `FAIL` remains in phase-owned correction or
reopen. `NEEDS_PARENT` returns to the acceptance owner through [Implementation
Obstacle Recovery](implementation-obstacle-recovery.md); only after no
evidence-changing in-scope remedy remains does the Evidence Contract determine
the stop.

## Close Out

State what changed, the important behavior consequence, proof actually run, and remaining gap or reopen owner. Apply the [Task Contract](../../../AGENTS.md#task-contract) to readiness and completion language. Move each durable decision to its canonical owner with the provenance and reopen condition required by [Artifact Model](../shared/artifact-model.md#resume-order) before removing the completed bundle.

## Stop Rule

Close the outcome only when every accepted changed or deliberately unchanged
behavior has a terminal disposition; the bounded accepted outcome—including
any required real production wiring, shared-cause repair, replacement cleanup,
and canonical/generated synchronization—is complete; the retained task delta
contains no unrelated change and has been reviewed against every triggered
risk; and the Evidence Contract permits every completion claim.

For a ledger acceptance unit, these criteria determine whether the acceptance owner may
accept the fixed unit candidate and what a triggered reviewer must falsify. A changed
decision reopens its narrowest upstream owner; after acceptance, completing
[Acceptance-Unit Closure](#acceptance-unit-closure) is the only authorized next
action before closeout or later work.
