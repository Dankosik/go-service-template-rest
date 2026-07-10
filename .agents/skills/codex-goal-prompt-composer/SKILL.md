---
name: codex-goal-prompt-composer
description: "Compose a compact Codex Goal from an explicit user request or a ready tasks.md implementation ledger."
---

# Codex Goal Prompt Composer

Use only when the user asks for a Codex Goal prompt or a ready implementation ledger needs a handoff.

Do not compose a Goal for specification review, technical-design review, test-design QA review, task-readiness review, post-code review, in-scope repair, re-review, validation, or closeout while the owning root can continue. Those are internal next actions, not handoffs.

Read [the handoff contract](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) and `tasks.md` first. Add another artifact only when the ledger points to a non-obvious constraint that must appear in the prompt.

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
