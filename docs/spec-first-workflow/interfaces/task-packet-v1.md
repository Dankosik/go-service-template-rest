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

Accept when:
- Claim: <what must be true>
- Focused check: <working directory + exact command, or precise procedure>
- Integrated check: <assembled-unit candidate proof required before Accepted; omit when none>
- Observable: <result that establishes the claim>

Reopen if:
<smallest upstream invalidation condition>
```

Use narrow source anchors for large inputs; name the repository for a source
outside the current checkout. Resolve variable paths deterministically and
record an unavailable required environment or input as a dependency gate.

Unannotated dependencies gate implementation. Keep gate annotations consistent
with the ledger's Depends on entries. When earlier preparation is justified,
Boundary names the work and local proof permitted from accepted inputs and the
stop before the pending gate. Accept when retains all required integration
proof; preparation does not satisfy it. [Ready
Frontier](../phases/planning/ledger-contract.md#ready-frontier) owns dispatch and
resumption.

Acceptance: Outcome, Provides, and Accept when describe successful work.
Record inability to complete as Blocked, not as an alternative success.

Mutable owners are semantic (package, contract, bootstrap), not a guessed file
list. Exclusive locks cannot be mutated concurrently even when files differ.
Write `none` unless this unit will mutate that surface. Do not list a lock or an
Integrated check as precaution. Omit Integrated check unless this unit's own
claim requires proof that the focused check cannot give.

A working checklist is optional and non-canonical. Checklist items are
execution lanes, not ledger tasks.
