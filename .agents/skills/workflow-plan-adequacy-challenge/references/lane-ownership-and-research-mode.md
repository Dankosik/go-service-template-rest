# Lane Ownership And Research Mode

## When To Load
Load this when the phase plan mentions local research, fan-out, challenger lanes, expert lanes, duplicate roles, or parallelism, but does not clearly state research mode, role, owned question, single skill, order/parallelism, or fan-in path.

## Authoritative Inputs
- `AGENTS.md`: each subagent pass is read-only and uses at most one skill; fan-out plans by lane, not unique role name.
- `docs/spec-first-workflow.md`: phase workflow plans record research mode, lanes, order or parallelism, fan-in or challenge path, blockers, and parallelizable work.

## Good Findings
- `Gap`: Phase plan says "use data and API agents" without naming the owned question or skill for each lane.
  `Why It Matters`: Two agents could cover the same seam while a material contract or data question remains unexamined.
  `Classification`: `blocks_specific_lane`.
  `Recommended Action`: `clarify_lane_ownership`.
  `Exact Orchestrator Addition`: In `workflow-plans/specification.md`, add `Research mode: fan-out`; add `Lane A: api-agent; owned question: request/response compatibility and error semantics; skill: api-contract-designer-spec`; add `Lane B: data-agent; owned question: persistence ownership and migration impact; skill: go-data-architect-spec`; add `Fan-in: orchestrator compares assumptions before candidate decisions`.
- `Gap`: Phase plan names fan-out lanes but omits whether they run in parallel or sequence.
  `Why It Matters`: A dependency-sensitive lane may be synthesized before prerequisite evidence exists.
  `Classification`: `non_blocking_but_record` if lanes are independent; `blocks_specific_lane` if one lane depends on another.
  `Recommended Action`: `add_missing_routing`.
  `Exact Orchestrator Addition`: Add `Order: Lane A and Lane B parallel; Lane C waits for synthesis only if A or B changes ownership assumptions`.
- `Gap`: Research mode is absent for a non-trivial workflow-planning phase.
  `Why It Matters`: Later sessions cannot tell whether missing lanes are intentionally local or accidentally omitted.
  `Classification`: `blocks_phase_handoff`.
  `Recommended Action`: `add_missing_routing`.
  `Exact Orchestrator Addition`: Add `Research mode: local; rationale: bounded single-domain skill edit, no subagent fan-out planned`.

## Bad Findings
- "Use more agents." Bad because it does not identify the missing lane ownership or why fan-out changes adequacy.
- "Let the data-agent decide the plan." Bad because subagents advise; the orchestrator owns decisions and implementation planning.
- "Put both go-data-architect-spec and go-db-cache-spec in one lane." Bad because each lane gets at most one skill.

## Blocker Classification Examples
- `blocks_phase_handoff`: research mode is absent for non-trivial or agent-backed work, or all lanes are too vague to support handoff.
- `blocks_specific_lane`: one lane lacks role, owned question, or single skill while other lanes remain usable.
- `non_blocking_but_record`: duplicate-role lanes are clear but should record why both exist, such as different owned questions or evidence targets.

## Exact Orchestrator Additions
- `workflow-plans/<phase>.md`: `Research mode: local|fan-out; Lanes: Lane <id>: role=<role>, owned question=<question>, skill=<one skill or no-skill>, scope=<read-only evidence target>; Order/parallelism: <parallel|sequential dependency>; Fan-in: <how orchestrator compares results>; Parallelizable work: <what can run together or none>`.
- `workflow-plan.md`: `Current phase: <phase>; phase plan status: active; research mode: local|fan-out; blocker: <only if lane gap blocks handoff>`.

## Exa Source Links
Exa MCP was attempted before authoring, but this environment returned a 402 credits-limit error. These fallback links are calibration only; repository-local docs remain authoritative.
- [Atlassian DACI](https://www.atlassian.com/team-playbook/plays/daci) for differentiating driver, approver, contributors, and informed roles without moving decision authority away from the owner.
- [Atlassian RACI chart](https://www.atlassian.com/work-management/project-management/raci-chart) for clarifying responsibility, accountability, consultation, and informed stakeholders.
- [Asana action log template](https://asana.com/templates/action-log) for assigning owners, due dates, priority, and status to follow-up actions.
