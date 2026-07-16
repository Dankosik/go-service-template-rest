---
name: codex-goal-prompt-composer
description: "Compose the single compact root Codex Goal when implementation is ready to start."
---

# Codex Goal Prompt Composer

Use only for a ready implementation/validation/closeout outcome: an accepted inline outcome or ready `tasks.md` ledger. Read that source, then compose—not create—the single root-thread Goal. [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md) and [repository authorization](../../../AGENTS.md#authorization) own the phase and Goal policy.

Return only:

```text
Goal: <one durable outcome>
Completion: <root acceptance, final integrated review, and fresh proof>
Read first: <accepted inline outcome or ready tasks.md, plus minimal context>
Constraints: <task-specific boundaries>
Proof: <required evidence or ledger section>
Stop/reopen: <exact blocker behavior and owner>
```

When the source lacks a completion condition or proof, return the missing input and its owner instead of a Goal. Omit empty fields and workflow mechanics.
