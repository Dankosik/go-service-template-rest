---
name: codex-goal-prompt-composer
description: "Compose the single compact root Codex Goal required for every authorized implementation, or when the user explicitly asks for one."
---

# Codex Goal Prompt Composer

Use before the first edit in every authorized implementation, regardless of task size, and when the user explicitly asks for a Codex Goal prompt. Direct work uses its accepted inline outcome; structured or orchestrated work uses the ready `tasks.md` ledger.

Compose one root-thread Goal covering implementation, post-code review, in-scope repair, re-review, validation, and closeout. Do not create separate Goals for subagents, workers, individual tasks, or internal checkpoints. Those are internal next actions, not handoffs.

Read the accepted inline outcome or ready `tasks.md` first. Read [the handoff contract](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) only for a next-session handoff. Add another artifact only when the accepted source points to a non-obvious constraint that must appear in the Goal.

Return only:

```text
Goal: <one durable outcome>
Completion: <one observable successful condition>
Read first: <tasks.md and minimal extra context>
Constraints: <task-specific boundaries only>
Proof: <required evidence or ledger section>
Stop/reopen: <exact blocker behavior and owner>
```

Omit workflow manuals, worker commands, model catalogs, broad repository summaries, repeated authorization rules, and empty fields. If a successful completion condition or proof is missing, return a blocked handoff only when the missing input satisfies the shared handoff gate; otherwise the owning root repairs the artifact without a prompt.
