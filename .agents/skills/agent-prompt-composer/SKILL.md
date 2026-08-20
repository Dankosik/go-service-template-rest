---
name: agent-prompt-composer
description: "Prompt packaging: Use whenever asked to write a repo-grounded prompt or rough input needs packaging. Own the shortest locator/delta prompt; Skip translation."
metadata:
  invocation: user
  kind: workflow
disable-model-invocation: true
---

# Agent Prompt Composer

Apply [Prompt Composition](../../../docs/prompt-composition.md). Use
[Intake](../../../docs/spec-first-workflow/phases/intake.md) only when outcome,
scope, authority, target, or stop meaning is unresolved. A routing-ready native
entrypoint or artifact takes the fast path: return its invocation/locator plus
only a missing user-owned delta. Load [examples](references/example-transformations.md)
only when calibration is unclear. Return the prompt or Intake's single blocking
question.
