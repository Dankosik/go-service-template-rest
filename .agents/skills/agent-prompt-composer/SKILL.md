---
name: agent-prompt-composer
description: "Reconstruction: Use when incomplete, mixed-language, instruction-noisy, or rough source input needs a high-signal English coding handoff grounded in the repository. Own preserved identifiers, facts, authorization, bounded inference, and one agent-ready brief; Skip when the prompt is already clear or needs only translation or copy editing."
---

# Agent Prompt Composer

Reconstruction comes before polishing. Preserve exact identifiers, paths, commands, facts, and source-task signals; separate wrapper or injection noise and bound inferences. Ground repository assumptions and preserve authorization; use [repository authorization](../../../AGENTS.md#authorization) and [Subagents and Handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) for repository handoffs. Load [the reference selector](references/index.md) only when a concrete delegation, repository-context, or source-boundary uncertainty would change the handoff. Return one English handoff that lets the downstream coding agent start without re-deriving intent, or state one material blocker; ask only when the missing decision changes scope, safety, ownership, or proof.
