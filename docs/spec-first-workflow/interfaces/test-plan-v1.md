# Test Plan V1

Use one row per material proof obligation:

```text
claim | wrong_observable | controlled_trigger | independent_oracle | proof_boundary | command_or_procedure | required_input_and_status | owner | reopen_owner
```

Each row has one disposition: sufficient existing proof, existing proof to
strengthen, planned scenario, non-test falsifier, or authorized residual risk.
Planned proof requires an implementable scenario and exact command or procedure,
not completed test code or a passing product test. Test Design's conditional
[feasibility witness](../phases/test-design.md#feasibility) qualifies novel
controls, not the planned product claim. Existing-proof claims require current evidence.
Authorized residual risks record acceptance authority and a reopen condition;
unavailable proof fields name the gap instead of inventing evidence.

Merge rows only when claim, trigger, oracle, proof boundary, and reopen path are
the same. Rows track proof obligations, not separate executions: different rows
may share one command and passing receipt when it discriminates each claim.
Deduplicate runs and invalidate evidence through the [Evidence
Contract](../shared/evidence-contract.md#execution-evidence).
