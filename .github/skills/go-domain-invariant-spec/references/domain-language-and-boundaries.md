# Domain Language And Boundaries

## Behavior Change Thesis
When loaded for symptom "domain terms, actors, ownership, approval, done, active, tenant, or source-of-truth vocabulary is ambiguous", this file makes the model define the local policy boundary before writing invariants instead of likely mistake "encode vague nouns, package names, endpoint words, or storage labels as business rules."

## When To Load
Load this before the invariant register when important words in the prompt or repo artifacts can mean different things, especially `task`, `phase`, `session`, `agent`, `owner`, `approval`, `done`, `active`, `tenant`, `source of truth`, `canonical`, `mirror`, `accepted`, `valid`, or `completed`.

## Decision Rubric
- Define only terms that can change allowed behavior, violation outcome, ownership, proof, or handoff. Do not build a glossary for every noun.
- For each loaded term, state: local meaning, non-meaning, authority source, actor allowed to decide it, and one decision it affects.
- Prefer repo-local policy vocabulary over package, endpoint, table, queue, or UI names.
- Treat "owner" as a policy authority question, not merely the file or service that notices the condition.
- Treat "done", "approved", "valid", and "active" as state or gate words until proven cosmetic.
- If the term cannot be resolved from local evidence and changes correctness, record an assumption or user decision instead of writing precise invariants around it.

## Imitate
```text
Term: task
Means here: a user-requested unit of work governed by the spec-first workflow.
Does not mean: goroutine, make target, test case, or arbitrary code edit.
Authority source: AGENTS.md plus task-local workflow artifacts.
Decision it affects: whether the session-boundary and implementation-readiness invariants apply.
```

Copy the boundary: the invariant applies to a local workflow task, not every technical use of the word.

```text
Term: canonical skill source
Means here: `.agents/skills/<skill>` is the authoring source for repository skills.
Does not mean: runtime mirror copies under `.claude`, `.cursor`, `.gemini`, `.github`, or `.opencode`.
Authority source: skill source-of-truth policy and skills sync/check flow.
Decision it affects: edits must land in `.agents/skills` first; mirror drift is a repair target, not a competing decision.
```

Copy the source-of-truth distinction: derived surfaces can be checked or repaired but do not own the rule.

```text
Term: owner
Means here: the actor or policy surface with authority to keep the domain rule true.
Does not mean: whichever handler, table, or package first detects the violation.
Authority source: approved spec, workflow contract, or domain ownership map.
Decision it affects: the invariant register names `orchestrator workflow contract` as owner rather than `workflow-plan.md`.
```

Copy the authority lens: artifacts can hold evidence without becoming the owner.

## Reject
```text
The task owner is the service that writes the `tasks` table.
```

Failure: storage mechanics are being treated as policy authority without evidence that the table owns the business rule.

```text
Approved means the user likes the plan.
```

Failure: "likes" is not an observable gate. Resolve the approval source and allowed next transition, or record an assumption.

```text
Active means not deleted.
```

Failure: this collapses lifecycle, entitlement, visibility, and retention possibilities unless local evidence proves they are identical.

## Agent Traps
- Do not start with aggregate or endpoint names when the prompt's business words are still ambiguous.
- Do not use repo examples as reusable product policy for another task; copy the boundary-thinking shape only.
- Do not let a term appear in an invariant statement if two actors could interpret it differently.
- Do not over-normalize harmless wording. Only define terms that affect behavior or proof.
- Do not ask the user for every ambiguous noun; use local evidence and record assumptions unless correctness would change.

## Validation Shape
A term boundary is useful when it lets the invariant register say what is allowed, who decides it, what is rejected, and what proof would fail if the term were interpreted differently.
