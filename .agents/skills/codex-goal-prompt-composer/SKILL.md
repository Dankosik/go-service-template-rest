---
name: codex-goal-prompt-composer
description: "Compose high-signal Codex Goal prompts. Use in exactly two cases: when the user explicitly asks to write, improve, or review a prompt for a Codex Goal, or when the repo workflow must render an implementation handoff from an approved and reviewed tasks.md. This skill consumes a Goal Contract, task ledger, validation loop, constraints, completion and blocked-stop rules to produce one durable objective, one verifiable completion condition, read-before-coding context, checkpoint progress rules, and a separate execution brief. Skip ordinary next-session prompts and draft tasks.md work that is not yet approved for implementation."
---

# Codex Goal Prompt Composer

## Purpose
Write the prompt that starts or hands off a Codex Goal.

A good Codex Goal is a durable objective for long-running work with a clear validation loop and a verifiable completion condition. In this repository, Goal prompts are mainly used when implementation starts from an approved, reviewed `tasks.md` and should execute the whole ledger through named proof without stopping between task IDs.

This skill owns the chat-rendered Goal prompt. It does not approve `tasks.md`, invent missing decisions, or turn a draft ledger into an implementation handoff.

## Trigger Policy
Use this skill only in these trigger paths:
- explicit user request: the user asks to write, improve, review, or compose a prompt for a Codex Goal;
- automatic workflow handoff: `tasks.md` exists, has passed task-ledger review/readiness with `PASS`, eligible `CONCERNS`, or eligible `WAIVED`, and the next session is implementation.

Do not trigger it merely because a draft `tasks.md` exists. In this repository, non-trivial `tasks.md` is normally written Goal-ready, but the Goal prompt is rendered only when the ledger is approved for implementation or the user explicitly asks for a Goal prompt.

## When To Use
Use this skill when:
- the user explicitly asks for a Codex Goal prompt;
- writing a recommended next-session implementation prompt that tells the next session to set a Codex Goal from an approved and reviewed `tasks.md`;
- composing a `/goal` or Goal-first implementation handoff from an approved and reviewed `tasks.md`;
- improving a Goal prompt so it uses the Goal feature correctly;
- checking whether a proposed Goal prompt is too broad, too vague, or missing proof.

Skip this skill for ordinary phase handoffs that do not set a Goal.

## Inputs
Read only the smallest set needed:
- the approved `tasks.md`, especially `Goal Contract`, `Implementation Handoff`, executable task IDs, proof obligations, progress rules, completion condition, and blocked-stop rule;
- `spec.md` for the canonical objective, non-goals, accepted decisions, and validation obligations;
- compact design or `design/` only when named by `tasks.md`;
- technical-design-review or readiness records only when the ledger names concerns, waivers, or proof obligations that must be preserved.

If the Goal prompt is not ledger-driven, use the user's explicit objective, constraints, proof commands, completion condition, and blocked-stop behavior. If these are missing, ask for or record the missing input instead of fabricating a Goal.

## Hard Rules
- One Goal prompt gets one objective and one successful completion condition.
- The objective must be bigger than one normal turn but smaller than an open-ended backlog.
- The objective must be durable: it says what to complete and when to stop, not every file, command, risk, or instruction.
- Put artifact lists, constraints, accepted concerns, waiver rationale, proof commands, progress rules, and blocker handling in the execution brief, not in the Goal objective.
- For approved-ledger implementation, scope the Goal to every required task in `tasks.md` through final validation, not just the first task ID.
- Do not compose a Goal prompt from a draft ledger, missing Goal Contract, `FAIL` readiness, unresolved decision gate, `TBD`, or implementation-blocking open question. Return a blocked planning/reopen prompt instead.
- Do not treat "records a blocker" as a successful completion condition. A blocker is a valid stop with the Goal left blocked, not a done claim.
- Do not let skipped, unavailable, stale, failing, or too-narrow proof satisfy a task checkbox, checkpoint, or completion claim. The prompt must tell implementation to leave affected tasks unchecked and record `Blocked:` or a narrower claim.
- Do not create or approve missing workflow artifacts inside the Goal prompt.
- Do not weaken repository stop rules. If a blocker requires a prior phase, tell Codex to stop, record the blocker in the allowed surface, and return the exact reopen target.
- Keep the final prompt self-contained for a context-blind next session, but selective. Do not paste artifact dumps or generic repo rules that `AGENTS.md` already covers.

## Goal Line Quality Gate
Before finalizing the first Goal line, rewrite it until:
- it names one target outcome and one verifiable completion condition;
- it remains understandable without the current chat and does not say `continue this`, `finish everything`, or similar context-dependent phrasing;
- it excludes artifact lists, proof commands, risk notes, and execution rules that belong in `Implementation brief`;
- it scopes to the approved ledger or explicit user objective, not an unrelated backlog;
- it would still be a valid objective after compaction, resume, or several hours of independent work.

## Repository Goal-First Prompt Shape
For this repository's next-session handoffs, prefer this instruction form over a raw slash command because the handoff may be pasted into the app or CLI:

```text
First, set a Codex Goal for this session:
Complete <approved objective> by executing every required task in `<task-local>/tasks.md` without stopping until <verifiable completion condition>.

After the goal is set, execute every required task in `<task-local>/tasks.md` from start to finish. Start at <first task or checkpoint>, continue through the ledger's final validation/proof, and do not redefine success around a smaller slice.

Implementation brief:

Work in `<absolute repo path>`.
Read before coding:
- `<task-local>/tasks.md` because it is the approved implementation ledger and source of truth.
- `<task-local>/spec.md` because it is the canonical decision record.
- <additional required start artifact, with one-line reason>.

Read before relevant tasks:
- <task-specific artifacts named by `tasks.md`, with task IDs and one-line reasons>.

Current state:
- Next phase: implementation.
- Task-ledger review: <PASS | CONCERNS | WAIVED>.
- Implementation readiness: <PASS | CONCERNS | WAIVED>.
- Start at: <task ID or checkpoint>.

Preserve:
- <accepted non-goals, constraints, concerns, waivers, and risks that affect implementation>.

Proof:
- <commands or manual proof required by the ledger>.

Quality and evidence:
- Apply the task-local implementation quality bar and evidence format from `tasks.md`.
- Do not check tasks or claim completion from skipped, unavailable, stale, failing, or too-narrow proof; leave affected tasks unchecked and record `Blocked:` or the narrower claim.

Progress rule:
- Update only the progress/evidence and closeout surfaces permitted by `tasks.md` after each checkpoint.

Blocked-stop rule:
- If an implementation-blocking decision, missing artifact, unavailable required command, or failing proof cannot be resolved inside the approved ledger, stop with the Goal blocked, record the blocker in the allowed ledger/closeout surface, leave affected tasks unchecked, and return `<reopen target>` without marking completion.
```

If the user explicitly needs a CLI slash command, make the first line `/goal <durable objective>` and keep the same separate execution brief after it.

## Composition Procedure
1. Confirm the Goal is eligible:
   - approved `tasks.md` exists, or the user supplied an explicit long-running objective;
   - readiness is `PASS`, eligible `CONCERNS`, or eligible `WAIVED`;
   - the completion condition, blocked-stop behavior, and proof path are concrete and not conflated.
2. Extract the objective from the `Goal Contract` or explicit user request.
3. Extract the completion condition from the `Goal Contract`, implementation handoff, and proof obligations.
4. Identify the first executable task or checkpoint.
5. Build the read-first list from ledger authority first, then add task-specific read context only when named by the ledger or needed to preserve non-obvious constraints.
6. Move all detailed instructions into `Implementation brief`.
7. Add the progress rule and blocked-stop rule.
8. Self-check that the prompt:
   - can be pasted into a fresh session;
   - does not rely on chat memory;
   - sets one durable Goal;
   - preserves approved scope without adding decisions;
   - names concrete proof, completion, and blocked-stop conditions;
   - prevents skipped, unavailable, stale, failing, or too-narrow proof from becoming green task progress;
   - tells Codex what to do if blocked.

## Blocked Output
When a valid Goal cannot be composed, do not produce a vague Goal. Return a short blocked handoff:

```text
Do not set a Codex Goal yet.

Reopen <planning | specification | technical-design | technical-design-review> for `<task-local path>`.

Reason:
- <missing Goal Contract, failing readiness, unresolved decision, vague completion condition, conflated completion/blocker semantics, or missing proof>.

Expected output:
- repair the recorded workflow artifact so a future implementation handoff can name one objective, one completion condition, separate blocked-stop behavior, read-before-coding artifacts, task-specific read context when useful, proof obligations, progress rule, and blocked-stop rule.

Stop after that phase and return the next-session prompt.
```

When composing an implementation prompt from an approved ledger, include this line in the implementation brief so any required read-only review, validation, or adequacy fan-out has explicit tool authorization without relying on the user to remember it:

```text
Subagent authorization: I explicitly request and authorize read-only subagents, delegation, and parallel agent work for every repository workflow gate that requires or benefits from fan-out in this session. Spawn the required read-only lanes without asking again; the orchestrator retains final authority and reconciles results.
```

## Source Notes
OpenAI's Codex Goal guidance at `https://developers.openai.com/codex/use-cases/follow-goals` frames goals as long-running work toward a verifiable stopping condition, with one objective, read-first context, proof artifacts or commands, checkpoints, a short progress log, and explicit pause/stop behavior. If this product behavior appears stale or the user asks for current docs, refresh against the official Codex Goal page before changing this skill.
