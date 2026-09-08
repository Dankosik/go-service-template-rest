# Subagent Brief Template

Write the assignment as a work message to a capable colleague, following
[Prompt Composition](prompt-composition.md). The fields below are optional
prompts for relevant information, not a required message format. Use natural
prose when it is clearer, and keep only what matters to this assignment.

Describe the result the parent needs to consume, rather than an activity such
as "explore" or "review". When its purpose is not obvious, name the decision or
next action it informs.

```text
Mode: decide | implement | investigate | verify | review
Outcome: <one checkable result>
Method: <phase adapter or skill; omit when obvious>
References and constraints: <accepted facts and minimal authoritative paths>
Writable scope: <only when non-obvious or required for isolation>
Proof: <planned final claim, or eligible coding-feedback question and bounded scenario; no per-task acceptance gate>
Stop: <completion, scope, authority, or missing-input boundary>
```

When accepted authority and discovered material coexist, label them separately
as `Authority` and `Evidence`; evidence never expands authority.

Pass model, effort, isolation, and native identity through tool fields. Do not
copy repository-wide workflow rules, model catalogs, unrelated context, or
generic strictness language into every brief.
