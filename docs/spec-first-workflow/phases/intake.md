# Intake Phase

Phase 0 for turning raw user intent into an accepted task brief before workflow routing, research, specification, design, planning, or implementation.

## Read When

- A new workflow task starts from rough notes, dictation, a vague idea, or mixed-language/raw input.
- The agent cannot yet state what the user wants to change, why, for whom, and what is in or out.
- A later phase discovers that the accepted scope, non-goals, success signal, or constraint was inferred from chat rather than confirmed or bounded.

Skip only when the user request is already agent-ready enough to choose a direct path or next workflow phase without changing scope by interpretation. For tiny direct-path work, the intake brief may be one sentence in chat.

## Inputs

- The user's raw request and any corrections in the current conversation.
- Existing task-local artifacts only when the request is a continuation or repair.
- Bounded repository lookup only when a named file, command, package, route, artifact, or prior task is needed to ask better questions.

Do not run research, design, task breakdown, or implementation during intake.

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
- next route: direct path, workflow planning, research, specification, user decision, or blocked.

## Interview Rules

- First reconstruct the user's likely intent from the raw input; do not ask the user to restate what is already clear.
- Ask only questions whose answers can change scope, behavior, constraints, ownership, risk, validation, or workflow route.
- Prefer questions that can falsify or materially change the draft brief over questions that only add detail.
- Answer repo-factual questions from bounded lookup when a named artifact can answer them; ask the user for intent, priority, scope, policy, or external constraints.
- Prefer a small number of high-leverage questions. Ask another round only when the prior answer exposes a new approval-changing ambiguity.
- Explain why each question matters when the reason is not obvious.
- Separate facts from inferences. Mark repository surfaces as confirmed only after bounded lookup or existing artifacts prove them.
- Preserve exact identifiers, paths, commands, endpoints, errors, domain terms, and user wording that may identify the real task.
- Treat repetition as priority signal, not noise.
- If the user wants to proceed without answering, record the bounded assumption and the reopen trigger instead of pretending certainty.
- If a missing answer would change correctness, safety, ownership, or scope, keep intake blocked and ask for that answer before routing.

## Universal Intake Prompt

Use this prompt when starting Phase 0 in any agent environment. It must stay tool-agnostic: do not rely on a named model, IDE, or question-asking tool.

```text
Interview me to understand this task before you route or implement it.

Use my raw request as evidence, not as final scope. First restate the likely objective and the uncertainty you see. Then ask only the smallest set of non-obvious questions whose answers can change scope, behavior, constraints, ownership, risk, validation, or workflow route. Do not ask generic checklist questions.

Prefer questions that could prove your draft understanding wrong over questions that only add detail. Answer repo-factual questions from bounded lookup when a named artifact can answer them; ask me for intent, priority, scope, policy, or external constraints.

Keep facts, inferences, and assumptions separate. Preserve exact identifiers, paths, commands, errors, APIs, product terms, and repeated emphasis. If I answer vaguely, push once for the missing decision instead of guessing.

Continue until you can produce an intake brief I can confirm:
- Objective
- Intent / context
- Scope / non-goals
- Affected surfaces: confirmed vs inferred
- Constraints / preferences
- Success signal / proof expectations
- Open questions / bounded assumptions
- Recommended next route
- Confidence boundary: what is confirmed, what is assumed, and what would reopen intake

Do not research, design, plan, edit files, or implement during this interview. If one missing answer would change correctness, safety, ownership, or scope, stop and ask that question directly.
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
- remaining uncertainty is safely bounded with assumptions and reopen triggers;
- a blocking user decision is named and asked directly;
- the request is clear enough for direct path and the brief is captured inline.

Do not classify execution shape, create `spec.md`, plan lanes, or start implementation from raw dictation alone.

## Anti-Patterns

- treating the user's first rough phrasing as final scope;
- asking a generic questionnaire instead of task-specific questions;
- using intake to start research, design, planning, or coding;
- hiding unresolved intent gaps as later "spec detail";
- turning the intake brief into a second `spec.md` or task ledger.
