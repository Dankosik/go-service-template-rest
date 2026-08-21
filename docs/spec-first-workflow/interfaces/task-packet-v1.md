# Task Packet V1

Every persisted task packet contains:

```markdown
# T<ID> — <outcome>

Outcome: <one falsifiable postcondition>

Inputs:
- <accepted source, dependency output, or exact locator>

Boundary:
<primary responsibility and material exclusions>

Accept when:
- Claim: <what must be true>
- Check: <smallest claim-matched proof>
- Observable: <expected result>

Provides:
<stable output consumed by a later task or final completion>

Reopen if:
<smallest upstream condition; omit when none>
```

A working checklist is optional and non-canonical. If one checklist item can be
implemented, proved, repaired, or accepted independently, promote it to its own
ledger task.
