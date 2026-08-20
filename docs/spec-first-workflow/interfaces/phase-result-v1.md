# Phase Result V1

Use one result at a durable macro-phase boundary:

```text
status: ready | blocked | skipped
owner: <phase>
result: <artifact path or inline result>
movement_evidence: <why the next owner may act, or none>
reopen_owner: <owner or none>
next_owner: <owner or none>
```

`skipped` records that the router's trigger was false; it never requires loading
the skipped phase. `ready` is valid only under [Phase
Movement](../shared/phase-movement.md).
