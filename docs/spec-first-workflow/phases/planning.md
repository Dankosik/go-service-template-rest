# Planning

Turn accepted decisions into the smallest executable sequence. Planning chooses
order and proof placement; it does not invent behavior or design.

## Read When

- Work has multiple dependent steps, owners, generated-source order, or
  validation checkpoints.
- Another actor/session needs a durable implementation ledger.
- A single structured acceptance unit needs a checkable execution handoff.
- An existing `tasks.md` needs repair.

Direct changes may keep their execution plan inline.

## Inputs

- Ready behavior, design, proof, and rollout decisions, whether recorded as a
  delta artifact or authoritative OpenAPI, tests, code, mockup, or external
  contract reference.
- Current source owners, generated/mirror commands, and repository validation
  commands.
- Accepted risks and proof obligations.

## Method

### Obligation reconciliation

1. Build a de-duplicated working set of implementation-changing accepted
   obligations from the ready inputs. Normalize restatements only when their
   authority, postcondition, execution-changing constraints, and proof
   consequence are equivalent. A normative conflict reopens its narrow upstream
   owner.
2. Give every obligation exactly one disposition: one unit, several named unit
   deltas with distinct postconditions and proof, a proved no-implementation
   disposition, or a scope exit. Compile the dispositions into the smallest
   coherent outcomes.
3. Reconcile both directions: every retained delta and proof maps to one
   obligation or enabling change, every obligation is represented, and
   boundaries follow valid postconditions rather than source-document shape.
4. Link-check the working set against current repository and deployment owners.
   For each changed contract, schema, canonical/generated authority, identifier,
   composition point, migration, or rollout state, give every reached producer,
   consumer, mirror, proof carrier, configuration/documentation surface, and
   replacement surface one disposition: coupled into the outcome, assigned to
   a named delta with a valid handoff, or proved unchanged. Missing accepted
   ownership or placement reopens design.
5. Record a planned wave only when multiple ready acceptance units will actually
   run concurrently and current evidence proves their independence.
6. Prove the next acceptance unit or real wave executable from closed inputs;
   later work needs owners and dependencies, not prematurely materialized
   inputs.

A **scope exit** disposes of an obligation outside the accepted outcome. Record
its gist, the current scope or non-goal wording that excludes it, and the owner
who could reopen it. Without current excluding wording, narrowing is user-owned,
so the obligation remains active or blocked. A scope exit is not completion
evidence.

A behavior-preserving restructure stays in an obligation unit when it shares
that unit's owner and proof boundary. An **enabling change** carries no outcome
of its own and exists only when a separately consistent, proved restructure
makes named obligation units smaller or safer; name those units and prove the
moved surfaces still behave unchanged.

Reconciliation completes when every accepted obligation and reached current
surface has one auditable disposition and every retained delta and proof maps
back to one of them.

When integration uncertainty or broad mechanical contract fan-out is present,
load [Conditional Planning Branches](planning/conditional-branches.md) before
choosing the execution shape.

## Outputs

Return the smallest execution form that preserves the accepted decisions:

- one fixed inline acceptance unit when there is one owner and proof boundary,
  the current session can continue, and no durable resume or actor boundary is
  needed;
- a compact `tasks.md` when work has multiple units, dependencies or waves,
  crosses an actor/session boundary, or needs durable status.

When `tasks.md` is triggered, load [Planning Ledger
Contract](planning/ledger-contract.md). Before declaring either form ready,
load [Planning Proof And Readiness](planning/proof-and-readiness.md).

## Readiness Review

Apply focused root self-review to every plan. Run independent [Task Review /
Readiness](task-review-readiness.md) only for a persisted ledger when the shared
review trigger applies.

Repair planning-owned findings directly. Reopen an earlier owner when execution
would still need to choose behavior, source of truth, runtime mechanism, package
ownership, proof strategy, or rollout policy. Fresh review follows only `FAIL`
repair or material candidate change.

## Stop Rule

The plan is ready only when obligation reconciliation passes; the selected
inline or ledger form contains every owner, surface, resource, dependency,
handoff, proof, and objective reopen condition needed by its next unit or wave;
and [Planning Proof And Readiness](planning/proof-and-readiness.md) reaches
acceptance without chat history or a new behavior, mechanism, placement,
ownership, proof, rollout, concurrency, or carrier decision.

When a ledger is present, every applicable contract in [Planning Ledger
Contract](planning/ledger-contract.md) also passes. Every actual wave has
current positive pairwise independence evidence. Any triggered review has
returned `PASS` or dispositioned `CONCERNS`.

Readiness locks the next Implementation entry to the fixed inline unit or first
executable ledger unit or real wave. Status checks and compaction preserve that
entry; only concrete evidence invalidating a named input or readiness
disposition reopens its smallest owner.
