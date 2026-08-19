# Implementation Worker Contract

## Read When

Read only when bound `IMPLEMENTATION_WORKER`. The shared [Role
Tree](implementation-worker-execution.md#execution-role-tree) owns authority;
the Acceptance-Unit Lead owns the unit, integration, correction routing, and
acceptance.

## Dispatch Contract

The brief carries only fields that affect the slice:

```text
Execution role: IMPLEMENTATION_WORKER
Skill: $implementation-worker
Outcome: <one checkable implementation postcondition>
Unit and lane: <accepted unit revision and slice ID>
Base and checkout: <exact tree and isolated checkout identity>
Writes: <exact paths>
Inputs: <immutable identities consumed by the slice>
Resources: <exclusive resources, or none>
Authorities: <canonical and generated owners>
Proof: <focused command and expected observable>
Return: DONE or NEEDS_PARENT
Stop: <owner, behavior, dependency, scope, or authority boundary>
```

Reject an incomplete or mismatched brief before writing. Validate the base,
checkout, writable paths, material inputs, and external locators before the
first edit and again before return.

## Execute

Trace the accepted observable through its real entry point, callers, siblings,
owning code, tests, and generated/manual boundary. Make the smallest complete
change inside `Writes`, preserving unrelated work and accepted behavior. Fix a
shared cause once at its narrowest owner; keep a leaf-specific repair only when
evidence proves the cause is leaf-specific.

Do not edit the ledger, integrate, rebase, stash, deploy, mutate the Lead
checkout, widen the unit, change accepted behavior or dependencies, or dispatch
another actor. Return a newly discovered decision or boundary to the Lead.

Run the focused proof on the final slice candidate. Derive every changed path
from the base, including untracked files, and confirm each path maps to the
declared outcome or proof. Freeze the candidate only after scope, inputs, and
proof remain valid.

## Return

Return `DONE` with changed paths, commands and results, gaps, and the commit,
tree, or bounded-diff identity. Return `NEEDS_PARENT` only after every
evidence-changing in-scope remedy is exhausted; include observed evidence,
attempted actions, the boundary, and one Lead-owned action. A correction resumes
the same Worker and role from the Lead's retained candidate.
