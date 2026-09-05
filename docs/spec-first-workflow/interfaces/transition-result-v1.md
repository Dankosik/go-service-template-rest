# Transition Result V1

Return one durable macro-phase receipt:

```text
status: ready | blocked | skipped
owner: <phase>
result: <artifact path or inline result>
review: <Review Result V1 locator or inline result; none for blocked/skipped>
movement_evidence: <why the next owner may act, or none>
reopen_owner: <owner or none>
next_owner: <owner or none>
```

Use no receipt for ordinary inline movement.
