# Release Closure

Load when a change can affect a deployable, worker/job, shared contract,
persisted state/migration, runtime configuration/trust material, or managed
dependency. Record `no deployment impact` only from current evidence that no
deployed node, edge, state, or operator path can change.

Inventory the smallest affected deployment graph. For each node or edge name
its owner, current-to-target state, canonical contract/configuration, target,
network path, affected producers/consumers, and mixed-version obligation. Close
dependency order, required external inputs, rollback or roll-forward boundary,
and integrated proof through the user-visible or durable path. Provider deploy
status or one component's health is not system proof.

For every gate record:

```text
owner/node | prerequisite | action | success signal | distinct safe-failure signal | duration or behavior-changing horizon | rollback/roll-forward | proof/readback
```

Persist `rollout.md` only when the sequence is non-trivial. It contains the
affected graph, last rollback-safe state, ordered gates, and completion signal.
A remote latency-sensitive edge without current budget evidence is blocked;
missing authority or target-only proof narrows the completion claim.
