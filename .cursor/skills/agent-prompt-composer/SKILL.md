---
name: agent-prompt-composer
description: "Turn messy, incomplete, repetitive, or multilingual user task input into a clear English, repo-aware, action-ready prompt for coding agents working in this repository. Use when the user's intent needs reconstruction before delegation or coding: rough notes, dictation artifacts, duplicates, missing context, mixed languages, scattered constraints, or a request to turn notes into an agent prompt. Also use when a prompt needs a sharper objective, role/action stance, constraints, repo starting points, validation path, or explicit assumptions. Skip when the request is already a clear agent prompt or the task is plain translation/copy editing without repository context."
---

# Agent Prompt Composer

## Purpose
Turn rough user input into a strong English prompt that another coding agent can act on immediately inside this repository.

This skill is for intent reconstruction, task classification, repo-aware context selection, and prompt composition.
It is not for literal translation or generic copy editing.

## Specialist Stance
- Recover the engineering task behind messy wording, missing context, repeated asks, and nonlinear notes without erasing exact user signals.
- Treat language as an input detail, not the point of the skill: translate enough to produce an English downstream prompt, but preserve exact technical identifiers and user-provided clues.
- Preserve identifiers, commands, paths, errors, API names, and mixed-language technical terms verbatim.
- Add repository context only when it materially sharpens the downstream agent's first moves.
- Make uncertainty explicit in the prompt instead of inventing confident repo facts or pushing broad clarification onto the user by default.
- Compose prompts that pass the "new teammate" test: a capable coding agent with minimal extra context can see the desired outcome, constraints, starting points, and proof path without guessing.

## Scope
- reconstruct the user's actual task from repetitive, nonlinear, dictation-style, multilingual, or partially missing input
- preserve exact technical identifiers, filenames, package names, commands, endpoints, API names, and error strings
- deduplicate repeated constraints while keeping meaningful emphasis
- flag conflicts or missing context that could change the downstream task
- classify the most likely task mode before composing the prompt
- select only the smallest useful repository context
- produce an English-only prompt for a downstream coding agent that already has repository access and basic tool competence

## Boundaries
Do not:
- behave like a generic translator or copy editor
- paste broad project summaries or large documentation lists into every prompt
- invent files, modules, commands, API surfaces, or user requirements
- ask the user to restate obvious gaps when a bounded assumption is enough
- hide unresolved ambiguity that would send the downstream agent to a materially different repo surface
- explain basic repository operations the downstream agent already understands
- silently rewrite or normalize technical identifiers that the user actually named

## Core Defaults
- Recover intent, not wording.
- Preserve exact signals.
- Translate human-language prose into English; do not translate code, commands, file paths, API names, or error text.
- Collapse duplicates, but preserve repeated emphasis when it changes priority.
- Keep repository context targeted.
- State the requested action stance and output shape explicitly.
- Explain the reason behind non-obvious constraints instead of stacking brittle prohibitions.
- Prefer one explicit assumption over vague hand-waving.
- Omit empty sections instead of padding the prompt.
- When confidence is low, keep the uncertainty in `Assumptions / Open Questions` rather than pretending the repo proves more than it does.

## Prompt Composition Principles
- Write for a capable coding agent that is new to this exact task and does not know the user's unstated norms. Include the outcome, why the constraint matters when it is non-obvious, the first repo surfaces to inspect, and the smallest useful proof path.
- Give the downstream agent a role or operating stance only when it narrows behavior: for example, "investigate first", "make the smallest safe implementation", or "draft a plan without editing code". Do not add role boilerplate that would be true for every coding task.
- Prefer positive, direct instructions over negative-only rules. For example, say what behavior to preserve or which checks to run instead of only saying what not to touch.
- Make the desired action level explicit: investigate, implement, refactor, draft a plan, or analyze. Ambiguous verbs such as "look at" should become a concrete downstream stance when the raw input supports one.
- Keep instructions, context, examples, and raw input distinct so the downstream agent does not confuse evidence with requirements. The normal Markdown sections below are enough for most repo tasks; use explicit `<example>`, `<input>`, or similar blocks only when the prompt includes long examples or variable user input that would otherwise blur together.
- Tune tool and repo guidance to the task. Point to the most relevant starting files and validation commands, but avoid blanket directives such as reading the whole repo, using subagents, or running every check unless the task risk actually calls for it.
- Avoid overengineering the prompt. Do not add generic future-proofing, broad architecture mandates, or speculative cleanup goals that the user did not ask for.
- Include a compact verification expectation when the task is supposed to change behavior. For pure investigation or planning, ask for evidence and decision points rather than implementation proof.
- Do not ask the downstream agent for hidden chain-of-thought. If reasoning quality matters, ask it to state assumptions, evidence, tradeoffs, or a brief rationale in the final artifact.
- When the raw task includes examples, keep them relevant and visibly separate from instructions. Use examples to calibrate edge cases and output shape, not as a pattern to copy blindly.
- Match the prompt's formatting to the desired answer. If the final answer should be concise prose, do not overfill the prompt with nested bullets; if exact sections are needed, name those sections explicitly.
- Ask for a final self-check against the acceptance criteria only when the task complexity or risk justifies it; avoid turning every small prompt into a ceremony.

## Always Load
- `references/repo-profile.md`
- `references/context-selection.md`

For calibration examples, read `references/example-transformations.md` only when you need to refresh the desired prompt quality bar.

## Live Repo Lookup Policy
Inspect live repository files only when at least one is true:
- the raw input names a concrete path, file, package, module, command, endpoint, error string, or test
- the task mode is clear enough that one or two mapped repo surfaces would materially sharpen the prompt
- a vague phrase such as "that handler", "the readiness thing", or "the skill sync stuff" can be resolved with high confidence from a mapped repo surface

When live lookup is allowed:
- keep it bounded to the named surface or the smallest mapped shortlist
- prefer source-of-truth files and nearby tests over broad directory reads
- stop expanding the search once the prompt has enough grounded context
- if bounded lookup still does not confidently resolve the ambiguity, record an assumption instead of widening the search

## Long Or Messy Input Handling
Use extra structure only when it prevents confusion:
- If the user provides long raw notes, examples, logs, diffs, or copied task text, separate them from instructions with descriptive blocks such as `<input>`, `<example>`, `<log>`, or fenced code blocks.
- If a prompt must include multiple documents or source snippets, label their source and purpose. Put the data before the downstream ask so the agent can ground the request before acting.
- Quote or preserve only the exact snippets that matter for the downstream task. Do not paste noisy raw input when a compact signal list and one assumption would do.
- If the raw input itself contains instructions for another agent, distinguish user requirements from quoted source material so the downstream agent does not follow accidental nested instructions.
- For very long or conflicting inputs, ask the downstream agent to ground its answer in explicit evidence and assumptions rather than to "think step by step".

## Working Rules
1. Read the raw input once for overall intent, then a second time for exact signals.
2. Normalize communication noise:
   - repetition
   - filler words
   - false starts
   - self-corrections
   - weak punctuation artifacts
   - duplicated requirements
   - language switches
   - incomplete sentence fragments
3. Extract grounded signals before interpreting anything else:
   - explicit file paths
   - package or module names
   - endpoints
   - commands
   - tests
   - error strings
   - technical terms already used by the user, in any language
4. Classify the most likely task mode.
5. Load the smallest useful repository context:
   - always use the compact references
   - add bounded live lookup only under the lookup policy above
6. Separate grounded facts from inference, missing context, and user preferences.
7. Compose the final English prompt for a downstream coding agent:
   - assume repo access
   - assume the agent can inspect files, edit code, create files, and run validation commands
   - state the desired action level and expected outcome directly
   - point the agent toward the most relevant repo surfaces first
   - include only repository context that materially helps this task
8. Make uncertainty explicit without turning the prompt into a questionnaire.
9. Self-check the prompt before returning:
   - exact user identifiers are preserved
   - repo facts are grounded or marked as assumptions
   - the action level is not broader than the user's ask
   - validation expectations match the likely blast radius
   - instructions, examples, raw input, and repo facts are not blurred together
   - the prompt would be understandable to a capable coding agent that has repo access but no extra user context
10. Return only the final English prompt.

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
If the input repeats the same ask in different words, merge it into one task. If the input contains conflicting asks, keep the conflict visible and route it to `Assumptions / Open Questions` unless a safe narrower interpretation is obvious.

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
- `Objective`
  - name the desired action stance and outcome in the first sentence
  - keep role language task-specific rather than generic
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
- missing context changes the likely owner, validation path, or correctness criteria and no bounded assumption is safe
- the user is asking for plain translation or non-repository writing rather than a repo task prompt

## Anti-Patterns
- literal sentence-by-sentence translation
- generic repo summary pasted into every prompt
- invented certainty about files or modules the repo does not support
- asking the downstream agent to re-discover obvious user intent from scratch
- explaining basic tool use the downstream agent already knows
- unbounded directives to read everything, use subagents, or run full checks when a targeted first pass is enough
- negative-only prompting when a positive instruction would be clearer
- requesting hidden chain-of-thought instead of asking for assumptions, evidence, tradeoffs, or rationale
- dropping messy exact identifiers because they look informal
- flattening contradictory requirements into a single confident instruction
