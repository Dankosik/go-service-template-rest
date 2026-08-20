# Evidence Contract

Canonical proof semantics for correctness, readiness, and completion claims.

## Qualifying Evidence

Evidence qualifies only when it is current, matches the claim's scope, and
would fail if the claimed behavior or required production wiring were absent
or wrong. Prefer one exercise of the observable path. Split proof qualifies
only when it separately exercises the owner and wiring and establishes that
together they realize that path.

Task status, file or symbol presence, an unrelated passing check, and an
implementation summary do not qualify by themselves.

## Proof Record And Reuse

Record the command, relevant environment and preconditions, result, and gaps.
Attach a commit or tree identity only across a checkout or integration
boundary; the current bounded diff is sufficient for local work. Reuse proof
only while its content, claim, provenance, preconditions, and risk surface are
unchanged; otherwise rerun it.

Assign one final owner to each deterministic gate. A Worker owns iterative
focused checks. The acceptance owner may reuse its receipt when the same tree
and preconditions cross integration unchanged; otherwise that owner runs the
gate on the integrated tree. The acceptance owner validates scope, identity,
provenance, preconditions, and the claimed observable instead of automatically
repeating a matching command. A reviewer runs only a missing or adversarial
falsifier for its independent question.

## Claim Disposition

Return one record per claim:

```text
claim: <intended assertion>
evidence: <current command or source and result>
status: verified | partially_verified | not_verified
gap_or_next_owner: <none, unverified remainder, or owner>
```

A summary inherits the weakest status among the claims it covers. When required
proof cannot run, record the command, reason, narrower evidence, and unverified
remainder. Stop as `implementation complete; verification incomplete`; do not
accept the unit or claim outcome completion or readiness. For ledger work this
is the canonical blocked transition owned by its acceptance owner.

## Verification Boundary

Verification reports evidence and gaps; it does not repair the implementation,
invent missing rollout criteria, or diagnose an unknown cause. Route those to
their owning method and rerun only the evidence invalidated by the resulting
change.
