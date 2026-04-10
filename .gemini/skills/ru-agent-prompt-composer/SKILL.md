---
name: ru-agent-prompt-composer
description: "Turn messy Russian task descriptions into an English, repo-aware prompt for coding agents working in this repository. Use when the user writes in Russian, mixed Russian/English, dictation-style notes, or repetitive rough wording and the real task needs reconstruction before delegation or coding. Skip when the request is already a clear English agent prompt or the task is plain translation without repository context."
---

# RU Agent Prompt Composer

## Purpose
Turn rough Russian input into a strong English prompt that another coding agent can act on immediately inside this repository.

This skill is for intent reconstruction, task classification, repo-aware context selection, and prompt composition.
It is not for literal translation.

## Scope
- reconstruct the user's actual task from repetitive, nonlinear, dictation-style Russian text
- preserve exact technical identifiers, filenames, package names, commands, endpoints, API names, and error strings
- classify the most likely task mode before composing the prompt
- select only the smallest useful repository context
- produce an English-only prompt for a downstream coding agent that already has repository access and basic tool competence

## Boundaries
Do not:
- behave like a generic translator or copy editor
- paste broad project summaries or large documentation lists into every prompt
- invent files, modules, commands, API surfaces, or user requirements
- ask the user to restate obvious gaps when a bounded assumption is enough
- explain basic repository operations the downstream agent already understands
- silently rewrite or normalize technical identifiers that the user actually named

## Core Defaults
- Recover intent, not wording.
- Preserve exact signals.
- Keep repository context targeted.
- Prefer one explicit assumption over vague hand-waving.
- Omit empty sections instead of padding the prompt.
- When confidence is low, keep the uncertainty in `Assumptions / Open Questions` rather than pretending the repo proves more than it does.

## Always Load
- `references/repo-profile.md`
- `references/context-selection.md`

For calibration examples, read `references/example-transformations.md` only when you need to refresh the desired prompt quality bar.

## Live Repo Lookup Policy
Inspect live repository files only when at least one is true:
- the raw input names a concrete path, file, package, module, command, endpoint, error string, or test
- the task mode is clear enough that one or two mapped repo surfaces would materially sharpen the prompt
- a vague phrase such as “that handler” or “that readiness thing” can be resolved with high confidence from a mapped repo surface

When live lookup is allowed:
- keep it bounded to the named surface or the smallest mapped shortlist
- prefer source-of-truth files and nearby tests over broad directory reads
- stop expanding the search once the prompt has enough grounded context
- if bounded lookup still does not confidently resolve the ambiguity, record an assumption instead of widening the search

## Working Rules
1. Read the raw Russian input once for overall intent, then a second time for exact signals.
2. Remove dictation noise:
   - repetition
   - filler words
   - false starts
   - self-corrections
   - weak punctuation artifacts
3. Extract grounded signals before interpreting anything else:
   - explicit file paths
   - package or module names
   - endpoints
   - commands
   - tests
   - error strings
   - English technical terms already used by the user
4. Classify the most likely task mode.
5. Load the smallest useful repository context:
   - always use the compact references
   - add bounded live lookup only under the lookup policy above
6. Separate grounded facts from inference.
7. Compose the final English prompt for a downstream coding agent:
   - assume repo access
   - assume the agent can inspect files, edit code, create files, and run validation commands
   - point the agent toward the most relevant repo surfaces first
   - include only repository context that materially helps this task
8. Make uncertainty explicit without turning the prompt into a questionnaire.
9. Return only the final English prompt.

## Task Mode Guidance
Use the inferred task mode to shape the prompt:

- `implement a feature`
  - emphasize desired behavior, likely ownership surfaces, acceptance criteria, and validation

- `fix a bug`
  - emphasize observed failure, likely code paths, regression proof, and smallest relevant verification

- `investigate an issue`
  - emphasize diagnosis path, evidence to gather, likely first files/tests, and no premature fix language

- `refactor or simplify`
  - emphasize current pain, boundaries not to change, invariants, and proof that behavior stays stable

- `draft a plan or spec`
  - emphasize decision areas, likely repo surfaces, open questions, and expected artifacts

- `analyze architecture`
  - emphasize boundaries, ownership, constraints, and decision forks rather than implementation details

- `improve prompts or tooling`
  - emphasize local skill, agent, prompt, workflow, mirror, and validation surfaces

- `clean up technical debt`
  - emphasize scoped cleanup, non-goals, and proof that no behavior regressed

If two task modes remain plausible, choose the narrower one and record the ambiguity under `Assumptions / Open Questions`.

## Repository Context Rules
- Start from the map in `references/context-selection.md`; do not bulk-read the repo.
- Prefer stable repo facts over template trivia.
- Mention generated-artifact rules only when the task touches OpenAPI, sqlc, mockgen, or stringer surfaces.
- Mention the spec-first orchestration model only when it materially shapes the downstream task.
- Do not overfit to the sample `ping` service unless the request actually touches those files.

## Output Expectations
Return only the final English prompt.
Use these sections in this order when they are relevant:
- `Objective`
- `Confirmed Signals And Exact Identifiers`
- `Relevant Repository Context`
- `Inspect First`
- `Requested Change / Problem Statement`
- `Constraints / Preferences / Non-goals`
- `Acceptance Criteria / Expected Outcome`
- `Validation / Verification`
- `Assumptions / Open Questions`

Section rules:
- `Confirmed Signals And Exact Identifiers`
  - include only explicit user signals and repo facts confirmed by bounded lookup
  - preserve exact filenames, commands, endpoints, errors, and technical terms verbatim
- `Relevant Repository Context`
  - include grounded repo facts only
  - keep this section compact
- `Inspect First`
  - list the most relevant starting points
  - mark inferred paths as `likely` when they are not directly confirmed
- `Requested Change / Problem Statement`
  - normalize the user's real ask into clear engineering language
  - do not turn a bug investigation into an implementation promise unless the user clearly asked for a fix
- `Validation / Verification`
  - mention the smallest useful commands or checks for the likely surface
  - keep broad checks conditional when a targeted check is the honest first proof
- `Assumptions / Open Questions`
  - hold every material inference or unresolved ambiguity here

Keep the prompt dense and high-signal.
Do not narrate how the transformation was performed.

## Escalate When
Escalate if:
- task mode cannot be distinguished even after bounded lookup
- two materially different interpretations would send the downstream agent to different repo surfaces
- the user is asking for plain translation or non-repository writing rather than a repo task prompt

## Anti-Patterns
- literal sentence-by-sentence translation
- generic repo summary pasted into every prompt
- invented certainty about files or modules the repo does not support
- asking the downstream agent to re-discover obvious user intent from scratch
- explaining basic tool use the downstream agent already knows
- dropping messy exact identifiers because they look informal
