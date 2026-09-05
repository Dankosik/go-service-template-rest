# Evidence Result V1

Return one record per claim:

```text
claim: <assertion>
candidate: <commit/tree/bounded diff identity>
scope: <package/files/behavior>
command: <exact command>
inputs: <changed paths or stable input identity>
environment: <relevant tags/services/toolchain>
result: pass | fail | blocked
duration: <wall time>
status: verified | partially_verified | not_verified
gap_or_next_owner: <none, unverified remainder, or owner>
```

A summary inherits the weakest status among the claims it covers. When required
proof cannot run, record the intended proof, reason, narrower evidence, and
unverified remainder.

The [Evidence Contract](../shared/evidence-contract.md#execution-evidence) owns
reuse eligibility for scoped results and whole-candidate receipts. Keep a reused
record's original candidate and exercised scope. The Lead validates its
applicability to the current claim; a reviewer reuses valid evidence or runs a
missing or distinct adversarial falsifier.
