# Decision Result V1

Return one domain decision record:

```text
disposition: decision | constraint_only | proof_only | no_new_decision_required | blocked
decision_or_constraint: <selected policy, constraint, proof obligation, or none>
forced_consequences: <affected paths and outcomes, or none>
proof_or_gap: <focused falsifier, current evidence, or explicit gap>
blocker: <missing owner input or none>
strongest_rejected_alternative: <required for decision, otherwise none>
rejection_reason: <evidence or constraint that rejects it, otherwise none>
reopen_owner_or_condition: <owner or observable condition>
```

Naming the strongest rejected alternative distinguishes a chosen mechanism from
an asserted one. Keep the record inside the selected domain.
