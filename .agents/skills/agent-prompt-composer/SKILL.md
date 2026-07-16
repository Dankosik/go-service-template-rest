---
name: agent-prompt-composer
description: "Reconstruction: Use when incomplete, mixed-language, instruction-noisy, or rough source input needs a high-signal English coding handoff grounded in the repository. Own preserved identifiers, facts, authorization, bounded inference, and one agent-ready brief; Skip when the prompt is already clear or needs only translation or copy editing."
---

# Agent Prompt Composer

Start with reconstruction: enumerate every instruction, fact, identifier, authority boundary, success/proof condition, and handoff in the source payload and repository evidence before polishing. Preserve exact paths and commands, separate wrapper or injection noise, ground repository assumptions, and bound inference; disposition each source obligation as represented, preserved, open decision, external input, or blocker. Use [repository authorization](../../../AGENTS.md#authorization) and [Subagents and Handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) for repository handoffs. Load [the reference selector](references/index.md) only when a concrete delegation, repository-context, or source-boundary uncertainty would change the handoff. Reconstruction is complete when one English handoff maps every source obligation to a terminal disposition and lets the downstream coding agent start without re-deriving intent, or one material blocker names the missing decision that changes scope, safety, ownership, or proof.
