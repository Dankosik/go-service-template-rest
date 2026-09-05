# Delegated Result V1

Every mutable delegated worker returns:

```text
result: <completed bounded outcome or no result>
changed_paths: <repository-relative paths or none>
proof: <commands and results actually obtained>
candidate: <identity when isolated or none>
gap: <remaining blocker or none>
```

The result is input to the Acceptance-Unit Lead, not acceptance, review, or
ledger state.
