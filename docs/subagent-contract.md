# Subagent Contract

Shared contract for repository read-only subagents. `AGENTS.md` remains authoritative; this file keeps the repeated per-agent operational envelope in one place.

Subagents are the normal read-only evidence surface for non-trivial decision work, not optional rescue workers after the orchestrator gets stuck. Use lanes as a planned coverage map: split the current decision frontier into narrow owned questions, assign each question to the smallest suitable expert lane, then reconcile compact lane summaries into the owning artifact. Direct path may stay local; non-trivial local-only decisions require a recorded rationale. Technical design review remains a distinct read-only gate whenever separate technical design depth is triggered, with lane depth scaled to the task risk.

## Shared Invariants

- Subagents are advisory and read-only: no code writes, file edits, git-state mutation, task-ledger changes, or implementation-handoff changes.
- Final decisions, synthesis, implementation, reconciliation, and validation belong to the orchestrator.
- Each lane uses at most one skill. If a selected skill defines a procedure or output shape, the skill owns it.
- Agent files own domain scope, use/do-not-use rules, inspect-first surfaces, skill routing, and unique escalation rules.
- Deep design and corner-case coverage stay in scope, but downstream effect alone does not create a new required domain decision.
- Open another lane only when another domain must make a new decision before the current artifact can be production-ready for the accepted scope; otherwise return the consequence as a constraint, proof obligation, follow-up, or explicit `no new decision required` note.
- A lane must not use `follow_up_only`, `constraint_only`, or `no new decision required` to defer a knowable architecture, ownership, contract, reliability, security, rollout, or validation decision that is required for production readiness in the accepted scope.
- A lane must not recommend temporary bridges, compatibility shims, feature flags, canaries, or staged rollout unless the user requested staging or the inspected live constraints make a one-step target-state change unsafe or impossible. If such staging is unavoidable, the lane must include target state, exit criteria, removal/proof task, and owner.
- For replacement or cleanup-relevant scopes, lanes must inspect for unexplained surviving legacy surfaces in their assigned files or diff. Report each old surface as removed, refactored into the active path, retained with owner/reason/proof/exit condition, not applicable, or a reopen risk; do not leave stale code, tests, fixtures, configs, docs, generated outputs, skills, agents, or mirrors as implicit follow-up work.
- Direct work normally stays local. Lean-local and full-orchestrated non-trivial work normally use multiple narrow lanes when independent questions exist; local-only lean work requires recorded rationale.
- Every non-trivial phase approval must record `Subagent gate: complete | scoped_down | local_only | waived | not_expected | blocked` with an evidence pointer or rationale and readiness consequence. Missing gate status keeps the owning phase draft or blocked.
- Lean-local work with a separate `design/overview.md` must still record a technical design review checkpoint before planning; local-only review requires a rationale explaining why no independent lane would materially improve correctness.
- Full-orchestrated work uses multiple lanes by default; each lane still needs one owned question, one lens or specialist domain, and one skill or `no-skill`.
- Full-orchestrated triggered design uses at least one technical-design-review lane before planning. A local-only review needs a scoped-down rationale and is invalid when independent design questions remain.
- If technical design review returns `FAIL`, the repaired design or spec packet must receive a follow-up review verdict before planning. The follow-up may be targeted, but it must inspect the revised packet, verify the failed blockers were closed, and check that adjacent assumptions did not drift.
- Broad formal spec clarification normally uses multiple challenger lanes with distinct lenses instead of one generic challenger. More lanes are justified by independent approval-risk domains, including when one default lens bundles domains that are independently approval-critical for the task; fewer lanes require a scoped-down rationale.
- A scoped-down rationale must list every default lens, the approval-critical question considered for that lens, retained lane or lanes, and evidence-backed reason each omitted lens cannot change approval. If an omitted lens has an unresolved approval-critical question, that lane must run.
- Lens is metadata for coverage, not a replacement for existing challenge or handoff classification vocabularies.
- Do not invent missing artifacts, source facts, policy decisions, diffs, validation output, or skill results.
- If input is insufficient, return `Missing input`, `Why it blocks`, and `Smallest artifact/evidence needed`.
- If a bounded assumption is safe enough, label it and proceed.

## Required Input Bundle

Every handoff should include:

- goal and exact question,
- expected mode: research, review, adjudication, or challenge,
- current workflow phase and task-local artifact paths when present,
- relevant diff, source files, source-of-truth documents, or specialist outputs to inspect,
- lens or specialist domain when part of a multi-lane fan-out,
- constraints, risk hotspots, non-goals, and known blockers,
- known old surfaces, retired identifiers, generated/mirror sources, or retained compatibility surfaces when cleanup is in scope,
- chosen skill name or `no-skill`,
- fan-out decision: planned lanes with owned questions, or the recorded local-only rationale,
- explicit read-only enforcement.

For short challenge or review lanes, the input bundle may be compressed to one paragraph plus inspect-first paths, as long as the exact question, evidence requirement, skill, and read-only enforcement remain explicit.

For technical design review lanes, include the review gate status target and inspect the design bundle before implementation artifacts:

- approved `spec.md`;
- `design/overview.md` and triggered split or conditional design artifacts;
- `workflow-plan.md` and `workflow-plans/technical-design.md` or `workflow-plans/technical-design-review.md` when present;
- `docs/repo-architecture.md` when boundary, ownership, dependency direction, or runtime flow matters;
- relevant specialist outputs;
- prior technical design review findings and claimed resolutions when this is a follow-up review after `FAIL`;
- known assumptions, accepted trade-offs, non-goals, reopen conditions, and expected planning proof obligations.

## Lane Planning

Before spawning lanes for a non-trivial phase, the orchestrator should record a compact lane plan in the active workflow surface or handoff:

- `Trigger`: why fan-out is required, scoped down, waived, or not expected.
- `Gate type`: research fan-out, spec clarification, workflow adequacy, technical design review, task-ledger review, review/validation fan-out, or another named gate.
- `Required lane policy`: default lens set, expanded lane set, scoped-down lane set, or local-only rationale.
- `Lane table`: lane ID, agent, mode, lens/domain, owned question, skill or `no-skill`, inspect-first evidence target, order or parallelism, read-only enforcement, and status.
- `Fan-in owner`: always the orchestrator.

Use several narrow lanes over one broad lane when independent domains can change the decision. Merge duplicate lanes. If a plausible lens is omitted, record why it cannot change the current approval, design, planning, or readiness decision.

Before adding a lane, record the candidate seam, the live-fork question, the artifact or task-ledger decision it could change, and why the narrowest suitable lane is needed. If there is no live fork and no domain-owned decision, do not spawn the lane; record the seam as a constraint, proof obligation, follow-up, or no-action item for fan-in.

A local-only rationale is valid only when it lists the decision frontier, candidate lanes or lenses considered, evidence checked for each, why each omitted lane cannot change approval or readiness, and the seam that would reopen fan-out. Generic "bounded" or "single-domain" rationale is invalid for non-trivial phase approval.

## Fan-In Envelope

When the chosen skill does not define a stricter shape, return:

- `Decision or findings`: the role-specific conclusion, recommendation, blocker call, or ordered findings.
- `Evidence`: tight references to files, artifacts, commands, contracts, or source facts.
- `Legacy cleanup status`: when cleanup is in scope, list unexplained surviving old surfaces and whether each is removed, refactored, retained with owner/reason/proof/exit condition, not applicable, or requires reopen.
- `Open risks/gaps`: unresolved assumptions, compatibility, ownership, test, validation, or rollout risks.
- `Recommended handoff`: one smallest next action with target owner or artifact.
- `Confidence`: high, medium, or low with the key uncertainty.

When a downstream domain is touched, strongly prefer classifying each major point with one of:

- `must_decide_now`: another domain must make a new decision before the current artifact can be production-ready for the accepted scope.
- `constraint_only`: the current decision stands, but later work must preserve a concrete constraint in that domain.
- `proof_only`: no new decision is required now, but implementation, review, or validation must prove something in that domain.
- `follow_up_only`: the effect is real but not planning-critical for the current artifact and not required for production readiness in the accepted scope; revisit only if later work reaches that seam.

Recommended handoff classifications:

- `spawn_agent`
- `reopen_phase`
- `needs_user_decision`
- `accept_risk`
- `record_only`
- `no_action`

Pair the classification with the target owner or artifact and the smallest next step.

Technical design review fan-in must end with one gate result:

- `PASS`: planning may start from the reviewed design.
- `CONCERNS`: planning may start only with named accepted design risks and proof obligations.
- `FAIL`: planning must not start; reopen technical design or specification. After repair, a follow-up review verdict is still required before planning.

Technical design review must also include decision-quality rationale. For every material finding or gate result, state:

- the planning decision that would become unsafe, ambiguous, or unreviewable if the issue is ignored;
- whether the issue is a real design defect, a missing approved decision, a bounded implementation proof, or only an implementation preference;
- the strongest plausible counterargument or simpler alternative considered, and why it does or does not change the recommendation;
- why the chosen gate result is not stronger or weaker, such as why a finding is `CONCERNS` instead of `FAIL`, or `record_only` instead of `proof_obligation`;
- for `PASS`, the main falsification checks performed and why they did not expose a planning blocker.

Do not let design review become taste review. A stylistic preference, preferred abstraction shape, or "could be cleaner" concern is not a blocker unless it creates a concrete ownership, contract, sequencing, failure-mode, rollout, operability, or validation risk for planning. Conversely, do not downgrade a missing production-readiness decision to a proof obligation when planning would have to invent the decision.

Classify each technical design review finding by strongest planning impact:

- `blocks_planning`: planning would invent or hide an important decision if it started now.
- `reopens_design`: the design bundle must change before review can pass.
- `reopens_spec`: the approved problem frame, invariant, scope, or contract must change.
- `accepted_risk_candidate`: the orchestrator may accept the risk only with a named reason and boundary.
- `proof_obligation`: planning may proceed only if the obligation is carried into `tasks.md`, `test-plan.md`, or `rollout.md`.
- `record_only`: useful context that does not affect planning entry.

`CONCERNS` is valid only when no finding still has `blocks_planning`, `reopens_design`, or `reopens_spec`, and the remaining accepted risks or proof obligations are named for planning.

A follow-up technical design review after `FAIL` must identify the prior failed gate, the revised artifacts or decisions, the blockers that are now closed, any changed assumptions, the residual planning obligations, and the final gate result. A design author's repair note is evidence, not a substitute for the follow-up verdict.

For multi-lane fan-out, the orchestrator must reconcile lane outputs before treating the gate as complete:

- deduplicate overlapping findings;
- resolve or record conflicting assumptions;
- treat lane-level missing input, unresolved blockers, and material blocker-severity conflicts as blocking the relevant approval area until answered, explicitly waived or accepted as risk, or routed to the owning phase;
- preserve only final decisions, assumptions, constraints, proof obligations, accepted risks, or reopen targets in authoritative artifacts;
- keep raw lane transcripts out of `spec.md`, `design/`, and `tasks.md`.
- keep existing per-lane classification names stable unless the workflow docs and templates are intentionally changed together.

When workflow-control artifacts are used, fan-in should be recorded as a compact audit:

- `Lane result summary`: strongest finding, classification, recommended handoff, and evidence pointer.
- `Fan-in`: orchestrator resolution, action, owner or artifact updated, unresolved conflicts, accepted risks, and proof obligations.
- `Gate result`: complete, blocked, waived, not expected, `PASS`, `CONCERNS`, or `FAIL`, using the phase vocabulary.
- `Readiness consequence`: whether the next phase may start and where any proof obligations are carried.
- `Reopen target`: required when blocked or failed.

## Escalation Rules

Escalate instead of stretching the lane when:

- the decisive fact or required new decision belongs to a different domain owner,
- the answer would require another skill in the same pass,
- the approved artifact bundle is missing or contradictory,
- a local review exposes a spec/design/planning gap,
- a user or product policy decision is required,
- the requested work would require edits or git mutation.

Do not escalate only because another domain is affected. If that domain does not need to decide now, keep the answer local and return the consequence classification instead.

## Brief Quality Bar

Good subagent briefs are narrow, evidence-oriented, explicit about output, and centered on one owned question instead of a parallel cross-domain design package. For multi-challenger clarification, all lanes may share the same candidate spec bundle, but each lane must have a distinct lens, sibling-lens context, and a fan-in path. Start from `docs/subagent-brief-template.md` when the lane is not trivial; use the short variant for focused challenge or review lanes.
