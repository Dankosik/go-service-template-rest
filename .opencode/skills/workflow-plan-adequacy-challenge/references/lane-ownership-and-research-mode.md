# Lane Ownership And Research Mode

## Behavior Change Thesis
When loaded for symptom agent lanes or research mode are muddy, this file makes the model require lane-level routing with one owned question and one skill instead of likely mistake saying only "use more agents" or letting subagents decide the plan.

## When To Load
Load this when a phase plan mentions local research, fan-out, challenger lanes, expert lanes, duplicate roles, or parallelism, but does not clearly state research mode, role, owned question, single skill, order/parallelism, or fan-in path.

## Decision Rubric
- For non-trivial or agent-backed phases, research mode must be explicit: `local` or `fan-out`.
- For each lane, require role, owned question, one skill or `no-skill`, read-only evidence target, and order/parallelism when it changes execution.
- Duplicate roles are fine when they answer different questions or use different evidence targets; they are a gap when they blur the same scope.
- Classify a missing research mode or wholly vague fan-out plan as `blocks_phase_handoff`; classify one incomplete lane as `blocks_specific_lane` when the rest of the phase can proceed around it.

## Imitate
### Vague expert lanes
`Gap`: Phase plan says "use data and API agents" without naming the owned question or skill for each lane.

Why to copy: it prevents duplicate coverage while a material API or data seam remains unexamined.

Use:
- `Classification`: `blocks_specific_lane`
- `Recommended Action`: `clarify_lane_ownership`
- `Exact Orchestrator Addition`: In `workflow-plans/specification.md`, add `Research mode: fan-out`; add `Lane A: api-agent; owned question: request/response compatibility and error semantics; skill: api-contract-designer-spec`; add `Lane B: data-agent; owned question: persistence ownership and migration impact; skill: go-data-architect-spec`; add `Fan-in: orchestrator compares assumptions before candidate decisions`.

### Missing order
`Gap`: Phase plan names fan-out lanes but omits whether they run in parallel or sequence.

Why to copy: the classification changes depending on whether lanes are independent.

Use:
- `Classification`: `non_blocking_but_record` if lanes are independent; `blocks_specific_lane` if one lane depends on another
- `Recommended Action`: `add_missing_routing`
- `Exact Orchestrator Addition`: Add `Order: Lane A and Lane B parallel; Lane C waits for synthesis only if A or B changes ownership assumptions`.

### Missing local/fan-out choice
`Gap`: Research mode is absent for a non-trivial workflow-planning phase.

Why to copy: later sessions cannot tell whether missing lanes are intentional local work or accidental omission.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `add_missing_routing`
- `Exact Orchestrator Addition`: Add `Research mode: local; rationale: bounded single-domain skill edit, no subagent fan-out planned`.

## Reject
- "Use more agents." This does not identify missing lane ownership or why fan-out changes adequacy.
- "Let the data-agent decide the plan." Subagents advise; the orchestrator owns decisions and planning.
- "Put both go-data-architect-spec and go-db-cache-spec in one lane." Each lane gets at most one skill.

## Agent Traps
- Do not demand fan-out just because multiple roles are available; challenge adequacy of the recorded mode.
- Do not collapse role and owned question. `data-agent` is not enough without the specific seam it owns.
- Do not classify duplicate-role lanes as redundant until you compare their owned questions and evidence targets.
