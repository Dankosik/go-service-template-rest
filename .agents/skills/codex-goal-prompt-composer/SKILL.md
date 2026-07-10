---
name: codex-goal-prompt-composer
description: "Compose high-signal Codex Goal prompts. Use in exactly two cases: when the user explicitly asks to write, improve, or review a prompt for a Codex Goal, or when the repo workflow must render an implementation handoff from an approved and reviewed tasks.md. This skill renders the compact handoff contract owned by docs/spec-first-workflow/shared/subagents-and-handoff.md. Skip ordinary next-session prompts and draft tasks.md work that is not yet approved for implementation."
---

# Codex Goal Prompt Composer

## Purpose

Render the compact Codex Goal handoff contract owned by `docs/spec-first-workflow/shared/subagents-and-handoff.md`.

This skill is a renderer, not a second contract owner. It selects approved task-local signals, checks that a valid Goal can be formed, and returns the compact chat prompt. It does not approve `tasks.md`, invent decisions, duplicate workflow policy, or embed worker execution manuals.

## Trigger Policy

Use this skill only when:

- the user explicitly asks to write, improve, or review a Codex Goal prompt; or
- a current approved `tasks.md` has an eligible task-review/readiness result and implementation is next.

Do not trigger it for ordinary phase handoffs or merely because a draft ledger exists.

## Inputs

Read the smallest sufficient set:

1. `docs/spec-first-workflow/shared/subagents-and-handoff.md` for the compact handoff contract.
2. Approved `tasks.md` for the Goal Contract, minimal read order, current implementation mode, preserved constraints, proof, progress/closeout owner, blocked-stop behavior, and reopen target.
3. Add `spec.md`, design, test, rollout, or review artifacts only when `tasks.md` names them and a non-obvious constraint cannot be rendered accurately without reading them.

For a user-authored Goal that is not ledger-driven, use only the explicit objective, completion condition, constraints, proof, and blocked-stop behavior supplied by the user. Do not manufacture missing workflow state.

## Eligibility Gate

Render a Goal only when all are true:

- there is one durable objective, larger than a normal turn and smaller than an open-ended backlog;
- there is one successful, verifiable completion condition distinct from blocked stop;
- an approved ledger is current for the active route when repository implementation is being handed off;
- readiness is `PASS`, eligible `CONCERNS`, or eligible `WAIVED`;
- required proof and the exact reopen behavior are known;
- no unresolved decision, `TBD`, missing required artifact, or `FAIL` gate would force implementation to invent policy.

If any condition fails, return a short blocked reopen prompt instead of weakening the Goal.

## Render Selectively

Use the compact implementation shape from `docs/spec-first-workflow/shared/subagents-and-handoff.md`. The generated prompt contains only:

- one durable objective covering every required `tasks.md` item through final validation;
- one successful completion condition;
- `tasks.md` first and only the minimal additional read order;
- current implementation mode and preserved constraints, accepted concerns, or waivers that affect execution;
- required proof or the exact ledger section that owns it;
- blocked-stop behavior and exact reopen target.

Do not copy into the prompt:

- worker launch commands, resume procedure, sandbox flags, patch-intake steps, or integration-proof mechanics;
- a generic orchestrator or workflow manual;
- broad repository summaries or artifact inventories;
- generic instructions such as `be strict`;
- repeated prohibitions or constraints already carried by `AGENTS.md`, approved `tasks.md`, or the implementation phase owner.

Task-specific worker boundaries stay in approved `tasks.md`. Repository-wide worker mechanics stay in `docs/spec-first-workflow/phases/implementation-validation-closeout.md`.

Headings are optional except where the user requests a particular format. Omit empty fields. If the user explicitly needs a CLI slash command, render `/goal <durable objective>` as the first line; otherwise use the shared contract's app-friendly `First, set a Codex Goal for this session:` form.

## Quality Gate

Before returning the prompt, confirm that:

- the Goal line names the target outcome and remains understandable after compaction or resume;
- completion requires the approved proof rather than merely recording a blocker;
- all required ledger tasks and final validation are in scope, not only the first task or checkpoint;
- every included sentence can change the next session's action, proof, or stop decision;
- no worker command, generic strictness language, duplicated artifact rule, or empty heading remains;
- skipped, unavailable, stale, failing, or too-narrow proof cannot be presented as successful completion.

## Blocked Output

When a valid Goal cannot be rendered, return only the compact reopen handoff:

```text
Do not set a Codex Goal yet. Reopen <owning phase> for `<task-local path>`.

Reason: <missing or failing Goal input>.

Expected output: repair the owning artifact so it supplies one durable objective, one successful completion condition, minimal read order, current implementation mode and preserved constraints, required proof, and separate blocked-stop/reopen behavior.

Stop after that phase and return the next-session prompt.
```

Do not include a `Subagent authorization:` line in implementation Goal prompts. Repository-standing `capability_only` authorization already covers required read-only review, validation, and adequacy lanes; preserve `agent_request=substantive` only when the accepted task intent makes multi-agent participation part of the result.
