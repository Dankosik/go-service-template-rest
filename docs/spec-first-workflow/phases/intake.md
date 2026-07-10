# Intake Phase

Phase 0 for turning raw user intent into an accepted task brief before workflow routing, research, specification, design, planning, or implementation.

## Read When

- A new workflow task starts from rough notes, dictation, a vague idea, or mixed-language/raw input.
- The agent cannot yet state what the user wants to change, why, for whom, and what is in or out.
- A later phase discovers that the accepted scope, non-goals, success signal, or constraint was inferred from chat rather than confirmed or bounded.

Skip only when the user request is already agent-ready enough to record an explicit clear-input rationale without changing scope by interpretation. Shape selection still follows under `AGENTS.md`; a tiny task's accepted brief may be one sentence in chat.

## Inputs

- The user's raw request and any corrections in the current conversation.
- Existing task-local artifacts only when the request is a continuation or repair.
- Bounded repository lookup when a named file, command, package, route, artifact, or prior task can resolve factual uncertainty or sharpen the brief.

Do not run research, design, task breakdown, or implementation during intake.

Ordinary Phase 0 is a minimal clarification pass, not an exhaustive interview. Reconstruct the likely brief before asking anything. Resolve repository-factual questions through bounded inspection of the named surfaces instead of asking the user. When a user-owned decision is still necessary, ask only the smallest missing question whose answer can change correctness, scope, ownership, safety, validation, or route; ask one question at a time and provide a recommended default when possible. If a safe bounded assumption is available, state it with its reopen condition and continue without forcing another interview round.

Use the repo-local `grilling` skill only when the user explicitly asks to grill, stress-test, challenge every branch, or conduct an exhaustive design interview. Ambiguity, risk, or unresolved user-owned decisions do not activate `grilling` by themselves.

## Outputs

Phase 0 produces an accepted intake brief, normally in chat. Preserve it in `workflow-plan.md` only when multi-session routing or later resume needs durable state.

The brief must include:

- objective: the change or outcome the user appears to want;
- user intent and context: why the change matters, including repeated emphasis from rough input;
- scope and non-goals: what is in, what is out, and what is intentionally unknown;
- affected surface: confirmed files, modules, APIs, docs, skills, artifacts, or likely surfaces labeled as inferred;
- constraints and preferences: process, safety, rollout, validation, language, style, deadline, or tool constraints;
- success signal: how the user and later phases will know the task is understood or done;
- open questions and assumptions: questions answered, questions still blocking, and bounded assumptions accepted for routing;
- next route: `shape_selection`, user decision, or blocked; Phase 0 does not preselect direct/lean/full or create workflow artifacts.

## Interview Rules

- First reconstruct the user's likely intent and draft brief from the raw input; do not ask the user to restate what is already clear.
- Ask only the smallest missing question whose answer can change correctness, scope, behavior, constraints, ownership, safety, validation, or workflow route.
- Ask one decision-changing question at a time. Include the recommended default and its consequence when possible.
- Prefer questions that can falsify or materially change the draft brief over questions that only add detail.
- Answer repository-factual questions from bounded inspection when a named surface can answer them; ask the user only for intent, priority, scope, policy, or external constraints that the repository cannot decide.
- Ask another question only when the prior answer exposes a new decision-changing ambiguity and no safe bounded assumption is available.
- Explain why each question matters when the reason is not obvious.
- Separate facts from inferences. Mark repository surfaces as confirmed only after bounded lookup or existing artifacts prove them.
- Preserve exact identifiers, paths, commands, endpoints, errors, domain terms, and user wording that may identify the real task.
- Treat repetition as priority signal, not noise.
- Prefer a safe bounded assumption over another interview round. State the assumption, its basis, and the condition that reopens intake.
- If the user wants to proceed without answering, record the bounded assumption and the reopen trigger instead of pretending certainty.
- If a missing answer would change correctness, safety, ownership, or scope, keep intake blocked and ask for that answer before routing.
- Stop clarifying as soon as objective, scope/non-goals, constraints, success criteria, and reopen conditions are sufficient for routing.

## Universal Intake Prompt

Use this prompt when starting Phase 0 in any agent environment. It must stay tool-agnostic: do not rely on a named model, IDE, or question-asking tool.

```text
Clarify this task only as much as needed before you route or implement it.

Use my raw request as evidence, not as final scope. First reconstruct the likely objective, scope/non-goals, constraints, success criteria, and reopen conditions. Resolve repository-factual uncertainty through bounded inspection of named surfaces. Do not ask me for facts the repository can answer.

If a user-owned answer is still necessary, ask only the single smallest missing question whose answer can change correctness, scope, ownership, safety, validation, or workflow route. Ask one question at a time and include your recommended default and its consequence when possible. Prefer a safe bounded assumption with a reopen condition over another interview round. Do not ask generic checklist questions.

Keep facts, inferences, and assumptions separate. Preserve exact identifiers, paths, commands, errors, APIs, product terms, and repeated emphasis. If I answer vaguely, push once for the missing decision instead of guessing.

Stop clarifying once you can produce a routing-sufficient intake brief:
- Objective
- Intent / context
- Scope / non-goals
- Affected surfaces: confirmed vs inferred
- Constraints / preferences
- Success signal / proof expectations
- Open questions / bounded assumptions
- Recommended next route
- Confidence boundary: what is confirmed, what is assumed, and what would reopen intake

Do not research, design, plan, edit files, or implement during intake. If one missing answer would change correctness, safety, ownership, or scope and no safe bounded assumption exists, stop and ask that question directly. Use `grilling` only if I explicitly ask to grill, stress-test, challenge every branch, or conduct an exhaustive design interview.
```

## Question Lenses

Use these as coverage lenses, not a checklist:

- desired outcome and user-visible change;
- current pain, observed failure, or opportunity;
- actors, callers, operators, or maintainers affected;
- accepted scope, non-goals, and replacement or cleanup expectations;
- constraints, process boundaries, and authorization for subagents when later phases may need them;
- risk areas such as API/data/security/money/reliability/concurrency/rollout;
- success criteria, proof expectations, and acceptable stop condition;
- existing artifacts, branches, prior decisions, or continuation state.

## Stop Rule

Stop intake when one of these is true:

- the user confirms or corrects the intake brief and the next route is explicit;
- objective, scope/non-goals, constraints, success criteria, and reopen conditions are sufficient for routing, including when remaining uncertainty is safely bounded with assumptions;
- a blocking user decision is named and asked directly;
- the request is clear enough for an explicit clear-input rationale and the brief is captured inline, so `SHAPE-*` evaluation may follow.

Do not classify execution shape, create `spec.md`, plan lanes, or start implementation from raw dictation alone.

## Anti-Patterns

- treating the user's first rough phrasing as final scope;
- asking a generic questionnaire instead of task-specific questions;
- asking the user for repository facts that bounded inspection can answer;
- asking several questions at once or continuing after the brief is sufficient for routing;
- activating `grilling` merely because ordinary intake has a user-owned decision;
- using intake to start research, design, planning, or coding;
- hiding unresolved intent gaps as later "spec detail";
- turning the intake brief into a second `spec.md` or task ledger.
