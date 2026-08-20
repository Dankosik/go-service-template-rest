# Acceptance Review Result V1

A valid independent implementation review returns:

```text
unit: <one recorded acceptance unit or fixed inline outcome>
candidate_identity: <immutable diff, tree, or commit identity>
verdict: PASS | FAIL | NEEDS_PARENT
findings: <anchored findings or none>
evidence: <current receipts and adversarial falsifier results>
unverified_claim: <required for NEEDS_PARENT, otherwise none>
repair_or_parent_owner: <required for FAIL or NEEDS_PARENT, otherwise none>
```

- `PASS` means every accepted postcondition and constraint is present on the
  real path, the retained delta is in scope, and current evidence permits
  acceptance.
- `FAIL` means current evidence proves a candidate-caused regression, accepted
  criterion violation, missing required surface, or absent in-scope proof the
  acceptance owner can obtain.
- `NEEDS_PARENT` means the available evidence cannot establish `PASS` or `FAIL`
  because required proof or action is outside reviewer authority. Its evidence
  names the narrower proof, attempted falsifier, and exceeded boundary.

An unresolved ledger boundary returns no verdict:

```text
REVIEW_HANDOFF_INVALID
received: <task or unit IDs>
recorded_units: <matching singleton or grouped units, or none>
candidate_identity: <supplied identity or none>
next_owner: <acceptance owner that must repair the handoff>
```
