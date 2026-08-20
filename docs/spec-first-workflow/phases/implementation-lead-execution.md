# Acceptance-Unit Lead Execution

## Read When

Read only when bound `ACCEPTANCE_UNIT_LEAD`. The shared [Role
Tree](implementation-worker-execution.md#execution-role-tree) owns authority;
this file owns one unit's Worker strategy, intake, correction, and acceptance.

## Execution Strategy

Inspect the fixed unit, current repository and dirt, dependencies,
generated/manual authority, mutable resources, and proof preconditions. Choose
the smallest Worker strategy that preserves the unit boundary:

- one serial Worker when one coherent slice can close the unit;
- several slices only when they have distinct implementation postconditions and
  current evidence proves their bases, inputs, writes, resources, and proofs can
  be isolated; or
- a dependency chain when a downstream slice must consume a concrete upstream
  output.

Every required write belongs to exactly one slice. Keep production code with
the tests and fixtures that prove it. A test-only slice is valid only when the
interface already exists at its base and its oracle needs no provisional
sibling output. Internal slicing never changes behavior, unit scope, or ledger
dependencies.

Emit a full Execution Map only when the strategy has multiple slices, a
non-initial base, a shared mutable resource or proof gate, or a cross-checkout
input. A single serial slice uses the compact Worker brief from [Worker
Contract](implementation-worker-contract.md#dispatch-contract); do not restate
empty edges, conflicts, or capacity.

When that trigger applies, load [Parallel Slice
Execution](implementation-lead-parallel.md) before dispatch. Otherwise dispatch
one serial Worker from the compact brief, record its native identity and
checkout, and apply the Agent Harness [Write-Carrier
Gate](../../agent-harness/shared/write-carrier.md#write-carrier-gate) before its
first write. On
`DONE`, validate scope, base, inputs, proof provenance, and mergeability before
integrating the returned delta.

## Scope Lock

Derive each returned path from the slice base, including untracked files. Every
retained change must be inside declared writes and map to the slice outcome or
proof. Reject a scope-invalid delta from its base without disturbing unrelated
provisional slices. Reopen the scope owner before any required expansion.

## Correction Loop

Keep each Worker reachable until the unit reaches its receipt or blocker. Send
one batched correction to the same Worker, naming each supported finding, its
violated criterion, and the proof that must change. Replace a Worker only for a
true execution stall or invalidated base.

Adopt a correction only when it closes or disproves a finding, introduces no
regression, stays in scope, and preserves passing proof. If a correction fails
to shrink the supported finding set, reject its delta, keep the prior fixed
candidate, and audit cause, owner, inputs, unit boundary, and proof before any
further write. Re-enter only on evidence that changes the causal hypothesis or
expected observable.

## Acceptance

After all slices are integrated, format the combined change, run mapped proof,
self-review the fixed unit candidate, and trigger independent review only under
the shared review rule. Current evidence may add a finding only for a
candidate-caused regression, an accepted criterion or repository invariant
violation, or missing proof for a completion claim.

Accept only an empty supported finding set with passing mapped proof and any
required independent `PASS`. Integrate only the accepted unit delta and persist
one canonical receipt or blocker. No slice, intermediate base, or Worktree
handoff releases a ledger dependency.
