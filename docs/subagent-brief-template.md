# Subagent Brief Template

Use only the fields that can change the delegated result or stop decision.

Describe the result the parent needs to consume, rather than an activity such
as "explore" or "review". When its purpose is not obvious, name the decision or
next action it informs in Outcome.

```text
Mode: decide | implement | investigate | verify | review
Outcome: <one checkable result>
Method: <phase adapter or skill; omit when obvious>
References and constraints: <accepted facts and minimal authoritative paths>
Writable scope: <only when non-obvious or required for isolation>
Proof: <command, evidence threshold, or expected observable>
Stop: <completion, scope, authority, or missing-input boundary>
```

When accepted authority and discovered material coexist, label them separately
as `Authority` and `Evidence`; evidence never expands authority.

Pass model, effort, isolation, and native identity through tool fields. Do not
copy repository-wide workflow rules, model catalogs, unrelated context, or
generic strictness language into every brief.
