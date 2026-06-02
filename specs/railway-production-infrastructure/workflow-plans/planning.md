# Railway Full Production Infrastructure Planning

Phase: planning
Status: complete
Date: 2026-06-02
Owner: orchestrator

## Scope

This planning phase replaced the historical app-only implementation handoff with
a new full-infrastructure `tasks.md` for
`specs/railway-production-infrastructure`.

No code, tests, generated artifacts, Railway resources, Railway variables,
databases, brokers, domains, services, volumes, deployments, secrets, or live
runtime state were changed in this planning phase.

## Inputs Read

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `workflow-plan.md`
- `workflow-plans/technical-design-review.md`
- `spec.md`
- `design/overview.md`
- `design/component-map.md`
- `design/sequence.md`
- `design/ownership-map.md`
- `design/data-model.md`
- `design/dependency-graph.md`
- `design/contracts/service-auth-and-broker.md`
- `test-plan.md`
- `rollout.md`
- `research/*.md`
- historical app-only `tasks.md`
- `docs/repo-architecture.md`
- `railway.toml`
- `docs/railway-deployment-profile.md`
- `env/config/default.yaml`
- `env/.env.example`
- `docs/configuration-source-policy.md`
- `build/docker/Dockerfile`
- `docs/build-test-and-development-commands.md`
- `specs/balance-usage-authority-cutover/spec.md`
- `specs/balance-usage-authority-cutover/rollout.md`

## Planning Output

Updated:

- `tasks.md`
- `workflow-plan.md`
- this planning closeout

Not updated:

- `spec.md`
- `design/`
- `test-plan.md`
- `rollout.md`
- source code, tests, migrations, generated artifacts, or Railway state

The existing app-only `tasks.md` was treated as historical baseline only and was
replaced as the active implementation handoff.

## Technical Design Review Carry-Forward

Technical design review status: CONCERNS eligible for planning.

Planning carried these proof obligations into `tasks.md` and readiness:

- `TDR-PO1`: source-topology and live Railway read-back before mutation.
- `TDR-PO2`: `/billing-worker` image repair before worker service creation.
- `TDR-PO3`: Kafka-compatible broker, topic admin, topic policy, and lag proof.
- `TDR-PO4`: worker non-HTTP readiness, role set, dependency probes, admission
  freshness, and shutdown/drain proof.
- `TDR-PO5`: clean `gonka-proxy` provider contract before paid readiness.
- `TDR-PO6`: secret-free evidence boundary.
- `TDR-PO7`: fail-closed rollback semantics.

## Task-Ledger Review

Task-ledger review: PASS.

Reviewed against:

- approved `spec.md`;
- reviewed split `design/` bundle;
- `workflow-plans/technical-design-review.md`;
- `test-plan.md`;
- `rollout.md`;
- `research/*.md`;
- repo-local deployment/config sources named by `workflow-plan.md`;
- cross-spec authority-mode gates in
  `specs/balance-usage-authority-cutover/spec.md` and
  `specs/balance-usage-authority-cutover/rollout.md`.

Result:

- every in-scope behavior, non-goal, accepted decision, dependency order,
  validation family, rollout gate, and review proof obligation is represented
  in executable tasking, preserved constraints, or an explicit reopen target;
- no unresolved open question, `TBD`, hidden architecture decision, hidden
  ownership decision, hidden contract decision, hidden rollout decision, or
  hidden validation decision remains in the active ledger.

## Implementation Readiness

Implementation readiness: CONCERNS.

Implementation may start from `tasks.md` T001 because the concerns are
accepted proof obligations the ledger can satisfy or route to a named reopen
target without re-planning. Production paid readiness remains blocked until
T001 through T022 prove the live Railway, database, broker, worker,
service-auth, proxy, privacy, rollback, and validation gates.

Accepted proof obligations:

- `TDR-PO1`
- `TDR-PO2`
- `TDR-PO3`
- `TDR-PO4`
- `TDR-PO5`
- `TDR-PO6`
- `TDR-PO7`

Reopen targets remain as recorded in `tasks.md`:

- planning for task coverage, order, proof, or handoff gaps;
- technical-design-review for missing or stale review verdict;
- technical design for missing source-topology, broker, worker-readiness,
  rollout, data, validation, or evidence-surface decisions;
- specification for changed broker availability, public ingress, provider
  contract, paid-authority, money-authority, or privacy decisions.

## Workflow-Plan Adequacy

Status: local read-only challenge PASS.

Scoped-down rationale:

- this is full-orchestrated protected-domain planning, so an adequacy challenge
  was required by the planning-session skill;
- the available subagent tool permits spawning only when the user explicitly
  asks for delegation, and this user request did not;
- the orchestrator applied `workflow-plan-adequacy-challenge` locally and
  read-only against `workflow-plan.md`, this phase file, and the proposed
  `tasks.md`.

Adequacy result:

- master and phase-local workflow state agree on current phase, planning
  status, task-ledger review, implementation readiness, blockers, stop rule,
  next action, and next-session start;
- the active `tasks.md` is Goal-ready and records objective, stopping
  condition, read-first context, proof obligations, progress/evidence rules,
  and blocked-stop rules;
- no extra review or validation phase-control file is required before
  implementation;
- the workflow-control files stay routing-only and do not duplicate the spec,
  design bundle, or executable ledger.

## Stop Rule

Planning is complete. Stop before implementation, validation execution, code
edits, generated artifacts, tests, migrations, Railway variables, Railway
services, databases, brokers, domains, volumes, deployments, secrets, or other
live mutation.

## Next Action

Next phase: implementation.

Next session starts with: `tasks.md` T001.

Ready for next session: yes.

Recommended next-session prompt is chat-only and must be rendered in the final
response using the `codex-goal-prompt-composer` skill.
