---
name: codex-goal-prompt-composer
description: "Compose the single compact root Codex Goal when implementation is ready to start."
---

# Codex Goal Prompt Composer

Use only when an accepted inline outcome or ready independently reviewed `tasks.md`
enters [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md).
[Repository authorization](../../../AGENTS.md#authorization) owns when the single
root-thread Goal starts and ends. Before implementation, return no Goal and name
the missing accepted input.

Compose one durable Goal for that whole macro phase. Bind completion to the
accepted outcome or ledger, final integrated review, and fresh proof without
copying Worker commands or phase mechanics. Do not create separate Goals for
Workers, ledger tasks, corrections, or internal checkpoints.

Read the accepted inline outcome or ready `tasks.md` first. Read [the handoff contract](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) only for a next-session handoff. Add another artifact only when the accepted source points to a non-obvious constraint that must appear in the Goal.

Return only:

```text
Goal: <one durable outcome>
Completion: <one observable successful condition>
Read first: <accepted inline outcome or ready tasks.md, plus minimal extra context>
Constraints: <task-specific boundaries only>
Proof: <required evidence or ledger section>
Stop/reopen: <exact blocker behavior and owner>
```

Omit workflow manuals, Worker commands, model catalogs, broad repository
summaries, repeated authorization rules, and empty fields. If completion or
proof is unresolved, return a blocked handoff only when the missing input
satisfies the shared handoff gate; otherwise the owning root repairs the source
artifact without a prompt.
