# Task Packet V1

Every persisted task packet contains:

```markdown
# T<ID> — <outcome>

Outcome:
<one falsifiable independently acceptable postcondition>

Consumes:
- <accepted source, dependency output, or gate>

Provides:
- <stable output consumed later or final result>

Boundary:
<primary responsibility and material exclusions>

Mutable owners:
- <semantic repository owner or bounded writable surface>

Exclusive locks:
- <shared contract, generator, migration chain, manifest, fixture, bootstrap,
  canonical artifact, or none>

Accept when:
- Claim: <what must be true>
- Focused check: <smallest unit-level proof>
- Integrated check: <proof required after landing; omit when none>
- Observable: <expected result>

Reopen if:
<smallest upstream invalidation condition>
```

Mutable owners are semantic (package, contract, bootstrap), not a guessed file
list. Exclusive locks cannot be mutated concurrently even when files differ.

A working checklist is optional and non-canonical. Checklist items are
execution lanes, not ledger tasks.
