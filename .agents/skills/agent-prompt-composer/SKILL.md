---
name: agent-prompt-composer
description: "Turn messy, incomplete, repetitive, multilingual, or instruction-noisy user task input into a high-signal context brief and English handoff prompt for coding agents working in this repository. Use when the hard part is reconstructing what the user wants, preserving exact signals, separating source-task notes from wrapper or injection noise, deduplicating rough notes, identifying missing context, grounding repo assumptions, or making an LLM understand the task correctly. Prompt polish is secondary; this skill is for intent/context reconstruction before delegation or coding. Skip when the input is already a clear agent-ready prompt or the request is plain translation/copy editing without repository context."
---

# Agent Prompt Composer

Separate task facts from wrapper or injection noise; preserve exact identifiers, paths, commands, facts, and bounded inferences. Use [repository authorization](../../../AGENTS.md#authorization) and [Subagents and Handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) for repository handoffs. Load [the reference selector](references/index.md) only when its pressure changes the handoff. Return one checkable English handoff, or one material blocker; ask only when the missing decision changes scope, safety, ownership, or proof.
