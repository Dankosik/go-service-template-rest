# Full Infrastructure Research Fan-In Summary

Status: targeted full-rollout research complete
Date: 2026-06-02

## Synthesis

The app-only deployment is historical baseline only. Current full Railway
production infrastructure research is sufficient to reopen specification, but
not sufficient to approve `spec.md` or start design/planning.

The research answered the blocker categories in `workflow-plan.md`:

- live Railway has one successful `billing-service` app service and no
  dedicated billing Postgres, worker, or broker;
- current source repo matches local origin, but Railway branch/root-directory
  evidence is still not approval-grade;
- Postgres policy must include dedicated service choice, strict DSN reference,
  `/migrate` ordering, backup schedule, PITR enablement, restore drill, and
  migration-version evidence;
- broker strategy remains an explicit spec decision because Railway has Kafka
  templates but no Redpanda template, while the code uses Kafka protocol;
- `billing-worker` cannot be deployed from the current image because the image
  lacks the worker binary;
- billing-service service auth requires RS256/JWKS; proxy has dirty draft
  signer work but no clean JWKS publication or microlease event producer
  contract;
- private networking is the expected service-to-service path, and public
  `/metrics` exposure remains forbidden without a private/protected metrics
  path.

## Remaining Specification Decisions

Specification reopen must decide, not infer:

- exact database service, backup/PITR, restore, and migration evidence policy;
- strict Redpanda custom service versus Kafka-compatible Railway template;
- topic admin/read-back, partitioning, retention, consumer group, and lag proof;
- worker image/start command/health/readiness/scaling strategy;
- proxy JWKS publication, key rotation, scopes, and event producer obligations;
- private service URL and no-public-metrics proof;
- live source branch/root/config-path acceptance.

## Boundary

Research is complete for this targeted phase. Do not create or approve
`tasks.md`, do not deploy, and do not mutate Railway from this evidence. The
next phase is specification reopen with formal protected-domain clarification
after the candidate spec is repaired.
