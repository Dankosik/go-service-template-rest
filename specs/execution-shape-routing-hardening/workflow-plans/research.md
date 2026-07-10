# Research Phase

## Session Objective

Gather and reconcile durable evidence for every `B01-F01`-`B01-F11` routing-hardening finding, including Pattern Fit Diligence, then hand off one specification phase without choosing or approving the repaired policy.

## Success Criteria

- Required lanes `R1`-`R4` return comparable, anchored evidence and visible limits.
- Preserved research notes and `research/synthesis.md` retain the complete finding map, candidate repair models, rejected alternatives, compatibility consequences, proof obligations, and specification destinations.
- Fan-in checks that a candidate closure for one finding does not reopen another.
- Master and phase-local routing agree on research status, blockers, artifact status, next-session start, and stop state.
- No specification, design, test-design, task-breakdown, implementation, or validation work starts in this session.

## Phase State

- Current phase: `research`.
- Phase status: `complete`.
- Execution shape: `full orchestrated`.
- Research mode: `fan-out`.
- Master: `specs/execution-shape-routing-hardening/workflow-plan.md`.
- Session boundary reached: `yes`.
- Ready for next session: `yes`.
- Next session starts with: `specification`.
- Subagent gate: `complete`; all required lanes returned and orchestrator fan-in is preserved in `research/synthesis.md`.
- Blockers: none for specification entry.

## Constraints And Available Evidence

- Accepted scope and finding ownership come from `workflow-plan.md` and `workflow-plans/workflow-planning.md`.
- Repository authority and phase boundaries come from `AGENTS.md`, the workflow router, the owning research phase contract, and the shared subagent/handoff contract.
- Source inspection after lane activation is restricted to the surfaces assigned to that lane.
- User authorization is present for read-only subagents, delegation, and parallel work for every applicable repository workflow gate in this session.
- Each lane uses explicit `no-skill`, stays read-only and advisory, and cannot make final policy decisions.
- Allowed writes are limited to this phase-control file, the five named `research/*.md` evidence artifacts, and research-phase routing/status updates in the master workflow plan.

## Required Lanes

| Lane | Execution | Owned question | Finding coverage | Skill | Status | Durable destination |
| --- | --- | --- | --- | --- | --- | --- |
| `R1` | read-only subagent | Reconcile authority, shape classification versus workflow planning, direct-code execution, and capability-only versus substantive agent-backed routing. | `B01-F01`, `B01-F02`, `B01-F07` | `no-skill` | `complete` | `research/shape-and-execution-authority.md` |
| `R2` | read-only subagent | Model typed statuses, reclassification, stale-state handling, research-skip routing, phase-file triggers, and artifact-first resume. | `B01-F03`, `B01-F04`, `B01-F05`, `B01-F08`, `B01-F10` | `no-skill` | `complete` | `research/status-reclassification-and-resume.md` |
| `R3` | read-only subagent | Determine adequacy, semantic guardrail, eval/CI, and canonical/runtime mirror proof requirements. | `B01-F06`, `B01-F09`, `B01-F11` | `no-skill` | `complete` | `research/adequacy-enforcement-and-mirrors.md` |
| `R4` | local orchestrator research lane | Compare established workflow, state-machine, and status-model patterns against repository ownership and proof needs. | Cross-cutting `B01-F02`-`B01-F06`, `B01-F09`-`B01-F10` | `no-skill` | `complete` | `research/pattern-fit.md` |

## Order And Parallelism

1. Start `R1`-`R3` as parallel read-only subagents and run `R4` concurrently in the orchestrator flow.
2. Keep each lane bounded to its assigned source surfaces and output contract.
3. Wait for all four lanes; a missing or unusable lane keeps research fan-in blocked.
4. Preserve lane evidence in the named research notes, then reconcile it in `research/synthesis.md`.
5. Update the master and this phase file only after fan-in establishes an honest completion or blocker state.

## Lane Output Contract

Every lane must separate:

- confirmed facts with exact `file:line` anchors;
- candidate repair models that remain non-authoritative until specification;
- rejected alternatives and why they fail;
- compatibility consequences;
- required proof and fail-before evidence;
- missing or conflicting evidence;
- recommended specification destination and reopen implication.

## Fan-In Path

- Final authority: orchestrator.
- Compare lane assumptions and resolve terminology conflicts without converting recommendations into approved decisions.
- Preserve all eleven finding IDs independently.
- Reject any candidate combination that closes one finding by silently reopening another.
- Classify each remaining specification input as `blocks_spec`, `proof_only`, `accepted_risk`, or `needs_specialist`.
- Record whether `design/overview.md` is triggered, conditional, or not expected; do not create it here.
- A formal specification clarification/challenge remains expected in the later specification checkpoint and is outside this session.

## Subagent Gate Audit

- Trigger: full-orchestrated, broad, multi-owner workflow-contract research with four independent authority, state, enforcement, and Pattern Fit questions.
- Gate type: `research fan-out`.
- Required lane policy: expanded `R1`-`R4` set from the approved workflow-planning packet; no lane was omitted, waived, or scoped down.
- Lane table: the `Required Lanes` table above records agent/local execution, owned question, finding coverage, explicit `no-skill`, read-only mode, durable destination, and final status. `R1`-`R3` ran in parallel as read-only subagents; `R4` ran concurrently as a bounded local Pattern Fit lane.
- Lane result summary:

| Lane | Decisive evidence test | Strongest result | Classification | Handoff/evidence |
| --- | --- | --- | --- | --- |
| `R1` | Intersect direct no-bundle/edit rules with worker eligibility; trace classifier and authorization ownership. | Direct code has no eligible writer; classifier ownership is circular; capability authorization can self-escalate shape. | `blocks_spec` | `research/shape-and-execution-authority.md` |
| `R2` | Separate existing state fields and test resume/reclassification/phase-file behavior against them. | Flat statuses, partial escalation, coupled research skip, overbroad spec-review file trigger, and artifactless status boundary need one typed model. | `blocks_spec` | `research/status-reclassification-and-resume.md` |
| `R3` | Compare adequacy inputs/checklist with the trigger matrix, then trace semantic proof through guardrails/evals/CI/sync. | Adequacy validates the recorded shape rather than falsifying it; trigger predicates drift; mirrors may be absent while checks pass. | `blocks_spec` plus named `proof_only` runtime limits | `research/adequacy-enforcement-and-mirrors.md` |
| `R4` | Compare concrete decision-table, state-machine, typed-status, condition, and event-history patterns with repository forces. | A small composite fits; imported workflow engines/event sourcing are disproportionate. | `blocks_spec` pattern choice, Pattern Fit complete | `research/pattern-fit.md` |

- Fan-in: all eleven findings remain independent in `research/synthesis.md`; overlapping models were reconciled into one non-authoritative composite candidate. No candidate was accepted where it would silently reopen another finding. No lane blocker, missing input, or material severity conflict remains. `design/overview.md` is not expected on current evidence, with a specification-density reopen trigger. No accepted risk or new specialist lane is needed. Downstream proof-only limits are the authorization-runtime requirement, artifactless current-session carrier integration, external mirror discovery, and availability of any model-eval runner.
- Gate result: `complete`.
- Readiness consequence: specification may start. It must decide every `B01-F01`-`B01-F11` item, preserve selected/rejected models and proof obligations, and must not treat research recommendations as approved policy.
- Reopen target: `research` if specification discovers a required policy choice that cannot be made from the preserved evidence; `workflow-planning` if execution shape, lane coverage, artifact depth, or the phase route itself changes.

## Preserved Research Artifacts

- `research/shape-and-execution-authority.md`: `complete`; R1 evidence for `F01`, `F02`, and `F07`.
- `research/status-reclassification-and-resume.md`: `complete`; R2 evidence for `F03`, `F04`, `F05`, `F08`, and `F10`.
- `research/adequacy-enforcement-and-mirrors.md`: `complete`; R3 evidence for `F06`, `F09`, and `F11`.
- `research/pattern-fit.md`: `complete`; required Pattern Fit Diligence with external examples and rejected machinery.
- `research/synthesis.md`: `complete`; full finding map, cross-reopen audit, classification register, design-depth result, and specification handoff.

## Completion Marker

Research is complete only when all required lanes are complete, all five research artifacts are present and reconciled, the Subagent Gate Audit is recorded, every `blocks_spec` item has enough evidence and an explicit specification destination, no unresolved research blocker remains, and master/phase-local handoff state agrees.

Completion marker: `met`.

## Next Action

Start one specification-only session from the master context bundle and the five preserved research artifacts. Create `spec.md` and triggered specification control only in that next session; do not begin design, test design, task breakdown, implementation, or validation.

## Stop Rule

Stop after durable research evidence, orchestrator fan-in, and routing-state updates. Do not create `spec.md`, `workflow-plans/specification.md`, `design/`, `test-plan.md`, `tasks.md`, or any implementation or validation change.
