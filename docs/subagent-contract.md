# Subagent Contract

Shared contract for every repository subagent. `AGENTS.md` owns fan-out policy and final authority. Agent TOMLs contain only domain deltas; workflow phase files own gate-specific mechanics.

## Boundary

- Subagents are read-only and advisory. Enforce this with the execution sandbox: no file or code writes, git mutation, task-ledger mutation, implementation-handoff changes, approval, or completion claims.
- The root orchestrator owns lane selection, synthesis, conflict resolution, authoritative artifacts, final decisions, validation, and completion.
- Each lane owns one concrete, independent, bounded question. Do not delegate a broad role label, a sequential step in the root reasoning chain, or work that would contend over shared mutable state.
- Each lane uses one skill, or explicitly `no-skill`. A selected skill owns its procedure and stricter output shape.
- Use no more than three concurrently active subagent lanes per root task by default. A larger concurrent set requires a task-specific reason recorded in the lane plan.
- Keep agent nesting at depth one. A child returns any newly discovered independent question to the root instead of spawning another child.

## Required Brief

Every lane brief includes:

- owned question and why separate context helps;
- mode: research, review, adjudication, or challenge;
- inspect-first artifacts or source surfaces;
- evidence boundary, constraints, and non-goals;
- expected specialist output;
- one skill or `no-skill`;
- model route: custom-agent profile, exact model and reasoning effort selected by the root before launch, task-complexity rationale, and the launch surface that enforces the pair;
- read-only execution choice.

Add only fields that can change the lane result. Review and challenge gates also name the reviewed artifact revision/content anchor, review-cycle attempt, prior findings whose closure must be checked, finding classification vocabulary, required evidence anchor, and allowed verdict recommendation. Do not copy repository-wide authority, fallback, model matrix, or handoff prose into each brief.

The `evidence-agent` may gather or reduce facts and propose a mechanical repair, but it cannot recommend or record a gate verdict. Role profiles do not select models. The root chooses and explicitly applies a task-appropriate model/reasoning pair before every lane launch under the canonical routing contract. A follow-up review uses a fresh thread/process and the same or a stronger task-appropriate capability choice than the review that found the issue.

## Compact Result

When the chosen skill does not require a stricter shape, return:

- `Conclusion`: direct answer to the owned question.
- `Evidence`: bounded file, artifact, source, or command anchors; separate facts from assumptions.
- `Finding or decision`: specialist-specific fields requested by the brief.
- `Open gap`: only unresolved input or risk that can change the conclusion.
- `Escalation`: target owner or artifact and the smallest next action, or `none`.

Every material finding names its `Fan-in destination`: `spec_decision`, `spec_constraint`, `proof_obligation`, `accepted_risk_candidate`, `reopen`, or `record_only`. Review or challenge findings also name the owner/reopen target and why the severity is not stronger or weaker.

## Shared Review Finding Envelope

Review skills inherit this envelope instead of copying it into every `SKILL.md`. A specialist skill adds only domain-specific axes, evidence, severity calibration, no-finding wording, or escalation fields.

Order review output as `Findings`, `Handoffs`, `Design Escalations`, `Residual Risks`, and `Validation Commands`. Put merge-risk findings first. Write `None.` for an empty section, or use a specialist's explicit no-finding sentence when it defines one; still disclose residual risks and evidence gaps.

Each finding uses:

```text
[severity] [skill-name] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

Every finding includes the exact source anchor, concrete defect, observable impact or realistic failure mode, smallest safe correction, supporting contract or evidence when one exists, and a focused validation command or missing-proof statement when useful. It also names whether the correction is local, needs a specialist handoff, or reopens design; material subagent findings retain the `Fan-in destination`, owner/reopen target, and why the severity is not stronger or weaker.

Calibrate severity by merge risk:

- `critical`: confirmed high-impact correctness, security, data, availability, compatibility, or release-safety failure that makes merge unsafe;
- `high`: strong evidence of a significant defect or unbounded risk on a material path;
- `medium`: bounded but meaningful correctness, operability, maintainability, or evidence weakness;
- `low`: local hardening or clarity improvement with concrete value and limited blast radius.

Do not invent a finding to fill the envelope. If the evidence does not support a merge-risk defect, record the proof gap or residual risk instead.

## Input Gaps And Escalation

- Do not invent missing artifacts, source facts, policy decisions, validation output, or sibling-lane results.
- If a safe bounded assumption permits progress, label it and continue.
- Otherwise return `Missing input`, `Why it blocks`, and `Smallest evidence needed`; do not broaden the lane or ask for unrelated context.
- Escalate when the answer belongs to another owner, the lane question is not independent, required evidence is outside the brief, or resolving it would change approved scope, contract, ownership, or workflow state.
- A lane may recommend another lane only by naming the new independent question. The root decides whether to spawn it, merge it, run it later, or keep it local.

## Gate Preservation

Mandatory specification review, formal clarification challenge, technical design review, task-ledger review, and other recorded independent gates remain mandatory when triggered. Mandatory means the checkpoint and independent review/challenge must occur; it does not imply several broad lanes. One focused lane is sufficient when it owns the only independent approval question. Multiple lanes are justified only by multiple non-duplicative questions that can change the gate result.

Applicable repository instructions provide standing `capability_only` authorization for read-only lanes. Do not ask the user to repeat authorization between a macro phase and its internal checkpoints. If both the primary spawn surface and configured independent Codex fallback are unavailable, block only the genuinely required lane; do not create extra lanes to manufacture the blocker or downgrade required review to local-only work.
