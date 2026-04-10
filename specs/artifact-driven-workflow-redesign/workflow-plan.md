# Workflow Plan

## Execution Shape

- Shape: `lightweight local`
- Research mode: `local`
- Why: this follow-up is bounded to repository workflow docs, task-local workflow artifacts, and mirrored skill instructions. The decision model is already framed; the remaining work is contract alignment and resumability hardening rather than new multi-domain research.

## Current Session Control

- Current stage: `done`
- Current session scope: `implementation: session-boundary contract alignment`
- Phase status: `complete`
- Completion marker:
  - session-bounded phase policy is reflected in `AGENTS.md` and `docs/spec-first-workflow.md`
  - `workflow-plan.md` requirements include session control, completion markers, and ready-for-next-session state
  - `spec-document-designer`, `go-design-spec`, and `planning-and-task-breakdown` stop at their handoff boundaries instead of flowing into the next phase
  - `README.md` and task-local artifacts reflect the same model
- Session boundary reached: `yes`
- Ready for next session: `no`
- Next session starts with: `N/A` unless a later reopen or follow-up cleanup task is created
- Stop rule: this session is closed; later work should start from a fresh session if new follow-up scope appears.

## Local Work Sequence

1. Extend the active task-local spec and plan so the approved session-boundary policy is captured in the redesign artifacts instead of living only in chat.
2. Update `AGENTS.md` and `docs/spec-first-workflow.md` so non-trivial work uses explicit session-bounded phases and a session-boundary gate.
3. Align the canonical skill texts:
   - `.agents/skills/spec-document-designer/SKILL.md`
   - `.agents/skills/go-design-spec/SKILL.md`
   - `.agents/skills/planning-and-task-breakdown/SKILL.md`
4. Re-sync those skill texts to the mirrored runtime directories.
5. Update `README.md` and `specs/artifact-driven-workflow-redesign/session-prompts.md` so discoverability and future handoffs match the new session policy.
6. Validate wording drift with targeted searches and a final diff review, then update the task-local artifacts to reflect the completion state.

## Validation Evidence

- `rtk rg -n "Session boundary reached|Ready for next session|current session scope|session-bounded|stop rule|one session|new session|phase-scoped" AGENTS.md docs/spec-first-workflow.md README.md specs/artifact-driven-workflow-redesign/spec.md specs/artifact-driven-workflow-redesign/plan.md specs/artifact-driven-workflow-redesign/workflow-plan.md specs/artifact-driven-workflow-redesign/session-prompts.md .agents/skills/spec-document-designer/SKILL.md .agents/skills/go-design-spec/SKILL.md .agents/skills/planning-and-task-breakdown/SKILL.md`
- `rtk rg -n "same session|next session|handoff boundary|phase-collapse waiver|waiver" .agents/skills/spec-document-designer/SKILL.md .agents/skills/go-design-spec/SKILL.md .agents/skills/planning-and-task-breakdown/SKILL.md AGENTS.md docs/spec-first-workflow.md`
- `rtk git diff --check -- AGENTS.md docs/spec-first-workflow.md README.md specs/artifact-driven-workflow-redesign/spec.md specs/artifact-driven-workflow-redesign/plan.md specs/artifact-driven-workflow-redesign/workflow-plan.md specs/artifact-driven-workflow-redesign/session-prompts.md .agents/skills/spec-document-designer/SKILL.md .agents/skills/go-design-spec/SKILL.md .agents/skills/planning-and-task-breakdown/SKILL.md .claude/skills/spec-document-designer/SKILL.md .claude/skills/go-design-spec/SKILL.md .claude/skills/planning-and-task-breakdown/SKILL.md .cursor/skills/spec-document-designer/SKILL.md .cursor/skills/go-design-spec/SKILL.md .cursor/skills/planning-and-task-breakdown/SKILL.md .gemini/skills/spec-document-designer/SKILL.md .gemini/skills/go-design-spec/SKILL.md .gemini/skills/planning-and-task-breakdown/SKILL.md .github/skills/spec-document-designer/SKILL.md .github/skills/go-design-spec/SKILL.md .github/skills/planning-and-task-breakdown/SKILL.md .opencode/skills/spec-document-designer/SKILL.md .opencode/skills/go-design-spec/SKILL.md .opencode/skills/planning-and-task-breakdown/SKILL.md`

## Expected Artifacts

- Required now:
  - [spec.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/artifact-driven-workflow-redesign/spec.md)
  - [plan.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/artifact-driven-workflow-redesign/plan.md)
  - [workflow-plan.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/artifact-driven-workflow-redesign/workflow-plan.md)
- Updated in this pass:
  - [session-prompts.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/artifact-driven-workflow-redesign/session-prompts.md)
  - [AGENTS.md](/Users/daniil/Projects/Opensource/go-service-template-rest/AGENTS.md)
  - [docs/spec-first-workflow.md](/Users/daniil/Projects/Opensource/go-service-template-rest/docs/spec-first-workflow.md)
  - [README.md](/Users/daniil/Projects/Opensource/go-service-template-rest/README.md)
  - canonical and mirrored copies of the three affected skills
- Design skip rationale:
  - this is a workflow-contract and skill-boundary rewrite, not a service-runtime change
  - no repository runtime sequence, adapter ownership, or data model needs a task-local `design/` bundle beyond what the core docs already describe
  - `spec.md` plus `plan.md` are sufficient to control this documentation/skill alignment pass honestly
