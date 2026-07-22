---
name: agent-prompt-composer
description: "Reconstruction: Use when incomplete, mixed-language, instruction-noisy, or rough source input needs a high-signal English coding handoff grounded in the repository. Own preserved identifiers, facts, authorization, bounded inference, and one agent-ready brief; Skip when the prompt is already clear or needs only translation or copy editing."
---

# Agent Prompt Composer

Reconstruct only source material that can change the outcome, editable boundary, authority or safety, implementation owner, proof, or stop condition. Preserve an identifier, path, or command only when downstream action depends on it; discard wrapper noise and link inherited repository rules instead of copying them. Use [repository authorization](../../../AGENTS.md#authorization-and-boundaries) and [Subagents and Handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md). Load [examples](references/example-transformations.md) only when output calibration is genuinely unclear. Return one compact English handoff that lets the coding agent act without re-deriving a material decision; otherwise name the one blocker that changes scope, safety, ownership, or proof.
