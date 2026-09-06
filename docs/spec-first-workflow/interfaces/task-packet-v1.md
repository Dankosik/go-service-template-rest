# Task Packet V1

Every persisted task packet contains:

```markdown
# T<ID> — <outcome>

Outcome:
<one falsifiable independently acceptable current-to-target change>

Consumes:
- <repository-relative path#section, stable ID, or dependency output/gate> — <decision or output used here; gate at acceptance or a named external effect when later than implementation>

Provides:
- <stable output consumed later or final result>

Boundary:
<primary responsibility, accepted replacement/cleanup, and material exclusions>

Mutable owners:
- <semantic repository owner or bounded writable surface>

Exclusive locks:
- <shared contract, generator, migration chain, manifest, fixture, bootstrap,
  canonical artifact, or none>

Final validation:
- Claim: <what must be true in the assembled ledger>
- Checks: <known required commands or release evidence only; omit until Implementation chooses tests>
- Observable: <result that establishes the claim>

Reopen if:
<smallest upstream invalidation condition>
```

The implementing agent fills in test cases and commands while writing code;
Planning needs the outcome, not a test-case inventory or approved oracle plan.

Use narrow source anchors for large inputs; name the repository for a source
outside the current checkout. Resolve variable paths deterministically and
record an unavailable required environment or input as a dependency gate.

Unannotated dependencies gate implementation and may consume landed Implemented
code or agreed contracts without passing checks. Keep annotations consistent
with the ledger. A later acceptance or effect dependency does not block local
implementation from closed inputs. Boundary names that work and the actual stop
before the pending gate. Final validation describes proof for the entire
assembled ledger, not an intermediate
task-transition gate; implementation alone does not satisfy it. [Ready
Frontier](../phases/planning/ledger-contract.md#ready-frontier) owns dispatch and
resumption.

Outcome and Provides describe code to implement; Final validation defines
the behavior to establish after the whole ledger is assembled.
Record inability to complete as Blocked, not as an alternative success.

Mutable owners are semantic (package, contract, bootstrap), not a guessed file
list. Exclusive locks cannot be mutated concurrently even when files differ.
Write `none` unless this unit will mutate that surface. Do not list precautionary
locks or checks that prove no required claim.

A working checklist is optional and non-canonical. Checklist items are
execution lanes, not ledger tasks.
