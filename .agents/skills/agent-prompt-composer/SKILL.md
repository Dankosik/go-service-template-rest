---
name: agent-prompt-composer
description: "Task messages: Use when asked to turn rough input into a clear repository task or handoff. Write as to a capable colleague, preserving intent and leaving execution choices to them."
metadata:
  invocation: user
  kind: workflow
disable-model-invocation: true
---

# Agent Prompt Composer

Turn the user's input into a work message you could send to a capable colleague.
Apply [Prompt Composition](../../../docs/prompt-composition.md) for the writing
principles and handoff boundaries. Make the request understandable without
prescribing the receiver's method or forcing the text into a template.

Load [examples](references/example-transformations.md) only when tone or the
amount of detail is unclear. Return the ready-to-send message. Use
[Intake](../../../docs/spec-first-workflow/phases/intake.md) only when a missing
user-owned decision materially changes what is being requested.
