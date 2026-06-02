# Billing Service Railway Full Production Infrastructure Workflow Plan

Mode: full orchestrated
Status: planning complete; implementation next with concerns
Current phase: planning
Phase status: complete
Research mode: local orchestrator research
Owner: orchestrator
Date: 2026-06-02

## Objective

Reopen the verified app-only Railway deployment into a full production
infrastructure rollout for `billing-service`.

The accepted scope covers:

- existing HTTP app service preservation and future production enablement;
- dedicated billing Postgres, migrations, schema-version proof, backup/PITR,
  restore proof, and semantic reconciliation before restored cutover;
- Kafka-compatible private broker, topics, producer/consumer contracts, and lag
  proof;
- `billing-worker` service, worker roles, readiness, scaling, and shutdown;
- service auth/JWKS and exact `gonka-proxy` provider-contract handoff;
- Railway variables, secret-source policy, private networking, and no public
  `/metrics` exposure by accident;
- rollout, rollback, fail-closed, and validation proof.

## Current Live Baseline

Read-only Railway inventory on 2026-06-02 confirmed:

- project `empathetic-clarity`
  (`58904982-91dd-4780-9b70-699dc9271e18`);
- environment `production`
  (`1c272106-e6be-4547-ae8d-2e7933eca6df`);
- app service `billing-service`
  (`64592463-335e-4be0-a8d1-330438fd61d0`);
- latest app deployment timestamp `2026-06-02 10:53:54.790 UTC`, `SUCCESS`;
- source repo `Dankosik/go-service-template-rest`;
- builder `DOCKERFILE`;
- health check path `/health/ready`;
- production service list has no `billing-service-postgres`, no
  `billing-worker`, and no billing-specific broker service.

This baseline must be preserved unless the approved implementation ledger
explicitly authorizes a mutation.

## Current Phase Result

Planning is complete.

The stale app-only `tasks.md` and `workflow-plans/planning.md` were treated as
historical app-only baseline and replaced with a full-infrastructure
implementation handoff.

No Railway resources, variables, services, databases, brokers, domains,
deployments, code, generated files, or runtime state were changed in planning.

## Artifact State

| Artifact | State | Trigger / Notes |
| --- | --- | --- |
| `workflow-plan.md` | planning complete | Master routing for the full infrastructure scope. |
| `workflow-plans/specification.md` | complete | Specification reopen and formal clarification reconciliation recorded. |
| `workflow-plans/research.md` | complete for full rollout | Targeted read-only research recorded for blocker categories. |
| `research/*.md` | complete for full rollout | Preserved evidence for data, broker/worker, service-auth/network, live inventory, deployment surfaces, validation, and fan-in. |
| `spec.md` | approved full infrastructure decision record | Canonical full infrastructure decisions and proof obligations. |
| `workflow-plans/technical-design.md` | complete | Full infrastructure design reopen state and review handoff. |
| `design/overview.md` | reviewed planning input | Full app/database/broker/worker/auth/proxy design entrypoint. |
| `design/component-map.md` | reviewed planning input | Full component map. |
| `design/sequence.md` | reviewed planning input | Dependency-gated sequence and failure policy. |
| `design/ownership-map.md` | reviewed planning input | Source-of-truth owners and rejected non-owners. |
| `design/data-model.md` | reviewed planning input | Postgres, migration, restore, reconciliation, and rollback data semantics. |
| `design/dependency-graph.md` | reviewed planning input | App/worker/Postgres/broker/proxy dependency shape and gates. |
| `design/contracts/service-auth-and-broker.md` | reviewed planning input | RS256/JWKS, route scopes, proxy handoff, Kafka topics, producer identity, and lag proof. |
| `test-plan.md` | approved planning input | Layered repo/Railway/database/broker/worker/auth/proxy/rollback validation. |
| `rollout.md` | approved planning input | Stateful deploy, restore, mixed-version, drain, authority, rollback, and failback choreography. |
| `workflow-plans/technical-design-review.md` | complete with eligible CONCERNS | Planning carried `TDR-PO1` through `TDR-PO7`. |
| `workflow-plans/planning.md` | complete | Full-infrastructure planning closeout, task-ledger review, readiness, and adequacy result. |
| `tasks.md` | approved with implementation CONCERNS | Active full-infrastructure implementation ledger. Historical app-only ledger is superseded. |

## Technical Design Review Carry-Forward

Technical design review is complete with eligible `CONCERNS`.

Planning carried these proof obligations into `tasks.md` and implementation
readiness:

- `TDR-PO1`: front-load source-topology and live Railway read-back before any
  mutation.
- `TDR-PO2`: repair and inspect the Docker image for `/billing-worker` before
  creating or starting `billing-worker`.
- `TDR-PO3`: prove the selected Kafka-compatible broker candidate, topic admin
  path, topic policy, and lag/backlog evidence before worker/app Redpanda
  readiness or paid authority.
- `TDR-PO4`: prove worker non-HTTP readiness, role set, dependency probes,
  admission-control freshness, and shutdown/drain, or reopen technical design
  before adding a health surface.
- `TDR-PO5`: keep paid authority blocked until a clean `gonka-proxy` provider
  contract or sibling ledger proves RS256/JWKS, scopes, private URL, event
  production, child-debit lineage, and no legacy fallback.
- `TDR-PO6`: preserve secret-free evidence boundaries.
- `TDR-PO7`: preserve fail-closed rollback semantics.

## Task-Ledger Review And Readiness

Task-ledger review: PASS.

Implementation readiness: CONCERNS.

Rationale:

- `tasks.md` covers the accepted target-state scope without asking
  implementation to choose a new architecture, ownership model, contract source,
  data class, sequencing policy, rollout policy, or validation class.
- The remaining concerns are live/external proof obligations the ledger can
  satisfy or route to a named reopen target.
- Production paid readiness remains blocked until the ledger proves the live
  Railway, database, broker, worker, service-auth, proxy, privacy, rollback, and
  validation gates.

Accepted proof obligations:

- `TDR-PO1`
- `TDR-PO2`
- `TDR-PO3`
- `TDR-PO4`
- `TDR-PO5`
- `TDR-PO6`
- `TDR-PO7`

## Workflow-Plan Adequacy

Status: local read-only challenge PASS.

Scoped-down rationale:

- full-orchestrated protected-domain planning requires an adequacy challenge;
- the available subagent tool permits spawning only when the user explicitly
  asks for delegation, and this request did not;
- the orchestrator applied `workflow-plan-adequacy-challenge` locally and
  read-only against `workflow-plan.md`, `workflow-plans/planning.md`, and
  `tasks.md`.

Adequacy result:

- master and phase-local workflow state agree on current phase, planning
  status, task-ledger review, implementation readiness, concerns, blockers,
  stop rule, next action, and next-session start;
- no extra pre-code review or validation phase-control file is required;
- the next session can start from `tasks.md` T001 without reading chat history
  or inventing workflow artifacts.

## Blockers

No blockers remain for entering implementation from `tasks.md` T001.

Production paid readiness remains blocked until implementation proves every
named gate in `tasks.md`.

## Reopen Conditions

Reopen planning if:

- task coverage, order, proof, evidence fields, or handoff state are incomplete
  before implementation starts.

Reopen technical-design-review if:

- a required review verdict is missing, stale, or contradicted by the revised
  packet before implementation starts.

Reopen technical design if:

- source-topology, broker candidate, worker readiness, app/worker resource,
  restore/reconciliation, proxy proof, rollback sequence, or validation evidence
  needs a different design while preserving the approved spec.

Reopen specification if:

- Railway cannot provide a private persistent Kafka-compatible broker satisfying
  approved requirements;
- a clean `gonka-proxy` provider contract cannot satisfy RS256/JWKS, scope,
  private URL, event producer, child-debit lineage, and no-fallback
  requirements;
- public billing ingress or public `/metrics` exposure becomes required;
- future source-topology read-back changes the approved deployment owner or
  target source assumptions;
- rollback or paid authority requires direct reserve fallback, proxy-local money
  writers, Redis/memory spend authority, weaker auth, or weaker privacy policy.

## Next Phase

Next session starts with: implementation from `tasks.md` T001.

Implementation must:

- set a Codex Goal first;
- execute every required task in `tasks.md` from T001 through final validation;
- preserve `TDR-PO1` through `TDR-PO7` as named proof obligations;
- keep evidence secret-free;
- update only ledger-owned progress/evidence and closeout surfaces allowed by
  `tasks.md`;
- stop and record the exact reopen target if a blocker requires a prior phase.

## Session Boundary

Session boundary reached: yes.

Ready for next session: yes.

Next session starts with: implementation from `tasks.md` T001.

Next session context bundle:

- `specs/railway-production-infrastructure/tasks.md` because it is the
  approved implementation ledger and source of truth;
- `specs/railway-production-infrastructure/spec.md` because it is the canonical
  decision record;
- `specs/railway-production-infrastructure/workflow-plans/technical-design-review.md`
  because it records the eligible `CONCERNS` verdict and `TDR-PO1` through
  `TDR-PO7`;
- `specs/railway-production-infrastructure/design/overview.md` and linked
  design artifacts because they define topology, ownership, sequencing, data,
  dependency, service-auth, broker, and proxy constraints;
- `specs/railway-production-infrastructure/test-plan.md` and
  `specs/railway-production-infrastructure/rollout.md` because validation and
  rollout triggers are planning-critical;
- `specs/railway-production-infrastructure/research/*.md` for evidence and
  limits;
- `docs/repo-architecture.md`, `railway.toml`,
  `docs/railway-deployment-profile.md`, `env/config/default.yaml`,
  `env/.env.example`, `docs/configuration-source-policy.md`,
  `build/docker/Dockerfile`, and
  `docs/build-test-and-development-commands.md` as repo-local source material;
- `specs/balance-usage-authority-cutover/spec.md` and
  `specs/balance-usage-authority-cutover/rollout.md` for authority-mode and
  cohort gates;
- `/Users/daniil/Projects/GonkaGate/gonka-proxy` only for read-only provider
  contract verification unless a separately approved proxy ledger authorizes
  writes.

## Recommended Next-Session Prompt

Chat-only. See the final response from this planning session for the
copy-pastable prompt.
