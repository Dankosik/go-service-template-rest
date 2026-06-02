# Railway Full Infrastructure Specification Reopen

Phase: specification
Status: complete
Date: 2026-06-02
Owner: orchestrator

## Specification Outcome

`spec.md` was repaired and approved for the full `billing-service` Railway
production infrastructure technical design phase.

Current result: approved for technical design with named proof obligations.

This approval does not authorize Railway mutation, production paid traffic, or
reuse of stale app-only implementation artifacts. The current app-only
deployment remains live and preserved.

## Inputs Used

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/specification-session/SKILL.md`
- `.agents/skills/spec-document-designer/SKILL.md`
- `.agents/skills/spec-clarification-challenge/SKILL.md`
- `.agents/skills/specification-session/references/spec-clarification-gate-flow.md`
- `specs/railway-production-infrastructure/workflow-plan.md`
- `specs/railway-production-infrastructure/workflow-plans/research.md`
- `specs/railway-production-infrastructure/research/fan-in-summary.md`
- `specs/railway-production-infrastructure/research/data-postgres-and-backups.md`
- `specs/railway-production-infrastructure/research/queue-worker-and-broker.md`
- `specs/railway-production-infrastructure/research/security-network-and-service-auth.md`
- `specs/railway-production-infrastructure/research/live-railway-inventory.md`
- `specs/railway-production-infrastructure/research/deployment-surfaces-and-sibling-examples.md`
- `specs/railway-production-infrastructure/research/readiness-scaling-rollback-validation.md`
- `railway.toml`
- `docs/railway-deployment-profile.md`
- `env/config/default.yaml`
- `env/.env.example`
- `docs/configuration-source-policy.md`
- `build/docker/Dockerfile`
- `docs/build-test-and-development-commands.md`
- read-only Railway service config, environment status, and service list
- prior billing authority and microlease rollout artifacts for rollback
  invariants

Existing app-only `spec.md`, `tasks.md`, `workflow-plan.md`, `design/`,
`test-plan.md`, and `rollout.md` under this task directory were treated as
historical baseline only.

## Reopened Scope

The approved specification scope includes:

- HTTP app preservation and future production enablement;
- dedicated billing Postgres, migrations, backup/PITR, restore proof, and
  semantic reconciliation before restored cutover;
- Kafka-compatible private broker, topics, producer/consumer groups, and lag
  proof;
- `billing-worker` service, image/start policy, seven required worker roles,
  readiness, scaling, and shutdown proof;
- scoped service-auth/JWKS and exact `gonka-proxy` provider-contract handoff;
- Railway variables, private networking, no accidental public `/metrics`, and
  secret-free evidence;
- rollout, rollback, fail-closed, and validation obligations.

## Clarification Challenge

Formal clarification challenge: complete.

Gate status: complete; reconciled with concerns.

Lanes: five read-only `challenger-agent` lanes with
`spec-clarification-challenge`.

Lenses:

- scope and spec coherence;
- domain invariants and edge cases;
- architecture ownership and dependency boundaries;
- API, data, compatibility, and source-of-truth consequences;
- security, reliability, delivery, and validation proof.

Scoped-down rationale: N/A; broad protected-domain default lens set used.

## Clarification Fan-In

| Lens | Strongest classification | Orchestrator resolution |
| --- | --- | --- |
| Scope/spec coherence | `blocks_spec_approval` | Repaired stale blocked state, app-only artifact boundary, and proxy/source-topology classification in `spec.md`. |
| Domain invariants and edge cases | `blocks_spec_approval` | Added restore semantic reconciliation, worker-disabled no-spend posture, broker-degraded fail-closed policy, rollback cutoff/cap semantics, and proxy lineage constraints. |
| Architecture ownership and dependency boundaries | `blocks_spec_approval` | Chose dedicated `billing-service-postgres`, Kafka-compatible private broker, separate `billing-worker`, same-repo repaired image topology, `gonka-proxy` JWKS ownership, and pre-mutation source-topology proof gate. |
| API/data/compatibility/source-of-truth | `blocks_spec_approval` | Recorded billing-service Postgres authority, approved topic/group defaults, scoped service-auth matrix, exact proxy handoff, and source-topology reopen condition. |
| Security/reliability/delivery/validation | `blocks_spec_approval` | Added backup/PITR/restore baseline, readiness policy, scaling guard, secret-free validation matrix, rollback fail-closed requirements, and paid-readiness block on clean proxy proof. |

Resolution status: all approval-changing questions were answered from existing
research and repository evidence or converted into explicit proof obligations
and reopen conditions. No raw challenger transcript was copied into `spec.md`.

Targeted research reopened: no.

Follow-up clarification: complete. A scoped read-only follow-up pass checked
the repaired `spec.md`, `workflow-plan.md`, `workflow-plans/specification.md`,
and research fan-in against the prior blocker categories and found no surviving
approval-changing question.

Rerun clarification only if technical design changes an approval-level decision,
weakens a proof obligation, or pre-mutation/source/provider evidence falsifies a
recorded reopen condition.

## Approved Spec Decisions

The canonical decisions are in `spec.md`. In short:

- current app service remains preserved baseline, not full readiness;
- production Postgres target is a dedicated `billing-service-postgres` service
  or exact reviewed single-tenant equivalent;
- daily backups, manual pre-cutover backup, PITR, restore to sibling, migration
  version proof, and semantic reconciliation are mandatory;
- private Kafka-compatible broker strategy is selected; config can remain under
  the `redpanda` namespace because the code speaks Kafka protocol;
- approved topic/group defaults are the four `billing.microlease.*.v1` topics
  and `billing-service-microleases`;
- `billing-worker` is a separate private Railway service that requires an image
  containing `/billing-worker`;
- RS256/JWKS service auth is required; shared-key auth is rejected;
- `gonka-proxy` owns signing/JWKS publication and must prove a clean provider
  contract before paid readiness;
- source repo/branch/root/config-path read-back is a pre-mutation proof gate;
- public `/metrics` remains forbidden without a later approved private or
  protected metrics design;
- rollback must fail closed and cannot revive direct per-request reserve or
  proxy-local money writes for migrated scopes.

## Blockers

No blockers remain for entering technical design.

Production paid readiness remains blocked until later phases prove the
specification's named gates. This is a proof and implementation block, not a
specification-approval blocker.

## Artifact Effects

- `spec.md`: approved for full infrastructure technical design with proof
  obligations.
- `workflow-plan.md`: updated to route next session to technical design reopen.
- existing `tasks.md`: remains verified app-only only, not a full rollout
  implementation handoff.
- existing `design/`, `test-plan.md`, and `rollout.md`: stale app-only inputs,
  not approved full-rollout artifacts.

## Stop Rule

Stop at the specification boundary.

Do not edit downstream design/test/rollout/task artifacts in this session.
Do not deploy or mutate Railway resources.

## Next Action

Next action: technical design reopen.

The next session should repair or replace the stale app-only design context for
the approved full infrastructure specification, produce a review-ready design
bundle and conditional `test-plan.md`/`rollout.md` decisions, update workflow
state, and stop before technical design review or planning.
