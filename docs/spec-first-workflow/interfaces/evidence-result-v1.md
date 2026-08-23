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

Reuse a prior receipt only when `candidate`, `scope`, `command`, and
`environment` are unchanged and `result` is `pass`. The Lead may accept a
worker receipt that matches those four fields. A reviewer does not rerun the
same command; it either accepts the receipt or runs a distinct adversarial
falsifier.
