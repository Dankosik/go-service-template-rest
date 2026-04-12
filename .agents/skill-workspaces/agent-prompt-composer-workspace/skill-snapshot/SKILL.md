---
name: agent-prompt-composer
description: "Turn messy, incomplete, repetitive, or multilingual user task input into a high-signal context brief and English handoff prompt for coding agents working in this repository. Use when the hard part is reconstructing what the user wants, preserving exact signals, deduplicating rough notes, identifying missing context, grounding repo assumptions, or making an LLM understand the task correctly. Prompt polish is secondary; this skill is for intent/context reconstruction before delegation or coding. Skip when the input is already a clear agent-ready prompt or the request is plain translation/copy editing without repository context."
---

# Agent Prompt Composer

## Purpose
Turn rough user input into a context-rich English handoff that lets another coding agent understand the user's real task inside this repository.

The primary deliverable is not a fancy prompt. The primary deliverable is an accurate task context model: what the user wants, which exact signals matter, what the repo likely implies, what is missing, and which assumptions are safe enough to carry forward. The final prompt is just the packaging for that context.

Use this skill for intent reconstruction, context extraction, deduplication, repo-aware grounding, and downstream handoff clarity.
Do not use it as a generic translator or copy editor.

## Specialist Stance
- Optimize for correct understanding before elegant wording. A plain prompt with the right context beats a polished prompt that smooths away uncertainty or user intent.
- Recover the user's actual engineering ask from messy wording, partial context, repeated phrases, dictation artifacts, multilingual notes, and nonlinear fragments.
- Preserve exact user signals: paths, commands, errors, API names, package names, tests, mixed-language technical terms, and unusual wording that could identify the target surface.
- Treat repetition as evidence. Collapse duplicate text, but preserve repeated emphasis when it changes priority, urgency, or non-goals.
- Separate what the user explicitly said from what you infer, what the repository confirms, and what remains unknown.
- Add repository context only when it materially helps the downstream model start in the right place.
- Make uncertainty visible without turning the handoff into a questionnaire by default.
- Compose the final English handoff so a capable coding agent can act without re-interpreting the raw user input from scratch.

## Scope
Use this skill to:
- reconstruct intent from unstructured, incomplete, repetitive, dictation-style, or multilingual task input
- identify the user's desired action level: investigate, implement, fix, refactor, plan, review, or explain
- preserve exact identifiers, filenames, commands, endpoints, package names, error strings, and domain terms
- deduplicate repeated asks while retaining meaningful emphasis and priority
- identify conflicts, missing context, hidden assumptions, and underspecified success criteria
- select the smallest useful repository context for the handoff
- produce a compact English context brief plus downstream agent prompt

## Boundaries
Do not:
- optimize mainly for eloquence, generic prompt-engineering style, or "beautiful" wording
- convert the user's notes into a literal sentence-by-sentence translation
- invent files, modules, commands, API behavior, product requirements, or business goals
- erase uncertainty to make the prompt sound more confident
- paste broad project summaries or large documentation lists into every handoff
- ask the user to restate obvious gaps when a bounded assumption is enough
- silently normalize or translate technical identifiers the user actually named
- make the downstream agent rediscover the user's intent from the noisy raw input

## Core Defaults
- Context first, prompt second.
- Preserve exact signals before interpreting them.
- Translate human-language prose into English; do not translate code, commands, file paths, API names, package names, test names, or error text.
- Keep raw wording only when it carries useful task evidence.
- Collapse duplicates; keep priority signals.
- Prefer one clear bounded assumption over vague hedging.
- Keep repository context targeted and grounded.
- Omit empty sections instead of filling a template.
- When confidence is low, place the uncertainty in `Assumptions / Open Questions`.

## Always Load
- `references/repo-profile.md`
- `references/context-selection.md`

Read `references/example-transformations.md` only when you need to refresh the expected quality bar.

## Context Reconstruction Model
Build the handoff from these layers, in this order:

1. **Raw signal capture**
   - exact identifiers: paths, files, commands, packages, APIs, endpoints, tests, logs, errors, named skills, tools, or docs
   - repeated or emphasized asks
   - explicit constraints, preferences, and non-goals
   - language switches and terms that should remain verbatim

2. **Intent reconstruction**
   - the likely outcome the user wants
   - the action stance the downstream agent should take
   - what should not be done yet, if the input implies caution
   - the smallest task boundary that honors the user's wording

3. **Context gap analysis**
   - missing facts that could change correctness, repo ownership, validation, or scope
   - conflicts between duplicated or revised statements
   - assumptions that are safe enough to carry forward
   - questions that are genuinely blocking

4. **Repo grounding**
   - durable repo facts from the required references
   - bounded live lookup only when the raw input or task mode justifies it
   - likely starting files or commands, marked as `likely` when inferred rather than confirmed

5. **Handoff packaging**
   - concise English wording
   - clear separation between user signals, repo facts, inferences, and open questions
   - enough validation guidance for the likely blast radius

## Live Repo Lookup Policy
Inspect live repository files only when at least one is true:
- the raw input names a concrete path, file, package, module, command, endpoint, error string, test, or skill
- the task mode is clear enough that one or two mapped repo surfaces would materially sharpen the context
- a vague phrase such as "that handler", "the readiness thing", or "the skill sync stuff" can be resolved with high confidence from a mapped repo surface

When live lookup is allowed:
- keep it bounded to the named surface or smallest mapped shortlist
- prefer source-of-truth files and nearby tests over broad directory reads
- stop expanding the search once the handoff has enough grounded context
- if bounded lookup still does not confidently resolve the ambiguity, record an assumption instead of widening the search

## Working Rules
1. Read the raw input once for the user's apparent goal.
2. Read it again for exact signals and context clues.
3. Normalize communication noise:
   - repetition
   - filler words
   - false starts
   - self-corrections
   - weak punctuation artifacts
   - duplicated requirements
   - language switches
   - incomplete sentence fragments
4. Extract grounded signals before interpreting the task.
5. Classify the most likely task mode.
6. Load the smallest useful repository context:
   - always use the compact references
   - add bounded live lookup only under the lookup policy above
7. Build a context model:
   - explicit user intent
   - exact identifiers
   - inferred task boundary
   - repo facts
   - validation implications
   - assumptions and open questions
8. Compose the final English handoff:
   - assume the downstream agent has repo access and can inspect files, edit code, and run checks
   - state the desired action level directly
   - include the context needed to understand why the task matters
   - point to the most relevant repo surfaces first
   - include only repo facts that help this task
9. Self-check before returning:
   - exact user identifiers are preserved
   - repetition has been deduplicated without losing emphasis
   - repo facts are grounded or marked as assumptions
   - inferred context is labeled as inference
   - the action level is not broader than the user's ask
   - validation expectations match the likely blast radius
   - the downstream agent can understand the task without reading the raw messy input
10. Return only the final English handoff prompt.

## Task Mode Guidance
Use the inferred task mode to shape the handoff:

- `implement a feature`
  - emphasize desired behavior, user-visible outcome, likely ownership surfaces, acceptance criteria, and validation

- `fix a bug`
  - emphasize observed failure, exact symptoms, likely code paths, regression proof, and smallest relevant verification

- `investigate an issue`
  - emphasize diagnosis path, evidence to gather, likely first files/tests, and no premature fix promise

- `refactor or simplify`
  - emphasize current pain, behavior boundaries, invariants, and proof that behavior stays stable

- `draft a plan or spec`
  - emphasize decision areas, likely repo surfaces, open questions, and expected artifacts

- `analyze architecture`
  - emphasize boundaries, ownership, constraints, and decision forks rather than implementation details

- `improve prompts, skills, agents, or workflow tooling`
  - emphasize the local skill/tooling surface, desired agent behavior, triggering context, mirror/sync implications, and validation surfaces

- `clean up technical debt`
  - emphasize scoped cleanup, non-goals, and proof that no behavior regressed

If two task modes remain plausible, choose the narrower one and record the ambiguity under `Assumptions / Open Questions`.
If the input repeats the same ask in different words, merge it into one task.
If the input contains conflicting asks, keep the conflict visible unless a safe narrower interpretation is obvious.

## Repository Context Rules
- Start from the map in `references/context-selection.md`; do not bulk-read the repo.
- Prefer stable repo facts over template trivia.
- Mention generated-artifact rules only when the task touches OpenAPI, sqlc, mockgen, or stringer surfaces.
- Mention the spec-first orchestration model only when it materially shapes the downstream task.
- Do not overfit to the sample `ping` service unless the request actually touches those files.

## Output Expectations
Return only the final English handoff prompt.

Use these sections in this order when they are relevant:
- `Objective`
- `User Intent And Context`
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
- `User Intent And Context`
  - summarize what the user appears to want and why the repeated or messy input matters
  - include priority/emphasis signals that were repeated in the source input
  - do not include generic motivation that the user did not imply
- `Confirmed Signals And Exact Identifiers`
  - include only explicit user signals and repo facts confirmed by bounded lookup
  - preserve exact filenames, commands, endpoints, errors, skill names, and technical terms verbatim
- `Relevant Repository Context`
  - include grounded repo facts only
  - keep this section compact and task-specific
- `Inspect First`
  - list the most relevant starting points
  - mark inferred paths as `likely` when they are not directly confirmed
- `Requested Change / Problem Statement`
  - normalize the user's ask into clear engineering language
  - do not turn an investigation into an implementation promise unless the user clearly asked for a fix
- `Constraints / Preferences / Non-goals`
  - preserve user preferences and repo constraints that should shape the work
  - include non-goals when they prevent likely overreach
- `Acceptance Criteria / Expected Outcome`
  - define what would make the task feel correctly understood and complete
  - avoid adding acceptance criteria that come only from generic best practices
- `Validation / Verification`
  - mention the smallest useful commands or checks for the likely surface
  - keep broad checks conditional when a targeted check is the honest first proof
- `Assumptions / Open Questions`
  - hold material inferences, unresolved ambiguity, and genuinely blocking questions

Keep the handoff dense and high-signal.
Do not narrate how the transformation was performed.
Do not include a separate critique of the original user input.

## Escalate When
Escalate instead of composing a confident handoff if:
- task mode cannot be distinguished even after bounded lookup
- two materially different interpretations would send the downstream agent to different repo surfaces
- missing context changes the likely owner, validation path, or correctness criteria and no bounded assumption is safe
- the user is asking for plain translation or non-repository writing rather than a repo task handoff

When escalation is needed, ask the smallest possible clarification question or return a handoff that clearly marks the blocking ambiguity.

## Anti-Patterns
- treating prompt polish as the main success metric
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
- stripping repeated user emphasis so the final context loses the user's real priority
