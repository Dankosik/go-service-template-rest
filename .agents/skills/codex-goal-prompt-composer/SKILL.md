---
name: codex-goal-prompt-composer
description: "Compose the single compact root Codex Goal when implementation is ready to start."
---

# Codex Goal Prompt Composer

Use only when entering the implementation/validation/closeout macro phase: direct work has an accepted inline outcome, or structured/orchestrated work has a ready independently reviewed `tasks.md` ledger. Establish the Goal immediately before the first implementation edit.

Do not use during intake, research, specification, technical design, test design, planning, or their review and repair loops, even when those phases edit repository workflow artifacts. If the user explicitly asks for a Codex Goal before implementation is ready, state that it starts only on entry to implementation and return no Goal.

Compose one root-thread Goal covering implementation, root diff inspection, any risk-triggered independent review and repair, validation, and closeout. A Goal does not itself trigger independent review for small direct work. Do not create separate Goals for subagents, workers, individual tasks, or internal checkpoints. Those are internal next actions, not handoffs.

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

Omit workflow manuals, worker commands, model catalogs, broad repository summaries, repeated authorization rules, and empty fields. If a successful completion condition or proof is missing, return a blocked handoff only when the missing input satisfies the shared handoff gate; otherwise the owning root repairs the artifact without a prompt.
