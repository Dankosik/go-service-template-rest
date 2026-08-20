# Evidence Result V1

Return one record per claim:

```text
claim: <intended assertion>
evidence: <current command or source and result>
status: verified | partially_verified | not_verified
gap_or_next_owner: <none, unverified remainder, or owner>
```

A summary inherits the weakest status among the claims it covers. When required
proof cannot run, record the intended proof, reason, narrower evidence, and
unverified remainder.
