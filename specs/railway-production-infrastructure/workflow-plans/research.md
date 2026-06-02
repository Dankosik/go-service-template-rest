# Railway Full Production Infrastructure Research

Phase: targeted read-only research
Status: complete; routes to specification reopen
Date: 2026-06-02
Owner: orchestrator

## Research Mode

Mode: local orchestrator research.

Rationale: the blocker set was concrete, source-located, and answerable through
read-only repository inspection, read-only Railway inventory/docs, and
read-only sibling `gonka-proxy` contract inspection. No subagent fan-out was
needed for this targeted checkpoint.

## Executed Lanes

| Lane | Question | Evidence Targets | Status |
| --- | --- | --- | --- |
| Data | Postgres, backups, PITR, restore, migration-version proof | Railway docs, live service list, `railway.toml`, Dockerfile, migrations, config policy | Complete; see `research/data-postgres-and-backups.md`. |
| Broker | Redpanda/Kafka strategy, topics, retention/partition, group, lag proof | Railway templates/docs, billing redpanda adapter, worker runtime, proxy searches | Complete; see `research/queue-worker-and-broker.md`. |
| Worker | `billing-worker` Railway service/image/start/health/scaling | Dockerfile, worker bootstrap/runtime, sibling Railway worker patterns | Complete; see `research/deployment-surfaces-and-sibling-examples.md` and `research/readiness-scaling-rollback-validation.md`. |
| Auth/network | service-auth/JWKS, proxy provider contract, private networking, no public metrics | billing-service auth/router/network code, OpenAPI scopes, dirty proxy checkout, Railway private networking docs | Complete; see `research/security-network-and-service-auth.md`. |
| Live topology | current app source/service assumptions | Railway MCP/CLI read-only inventory, local git remote/branch | Complete; see `research/live-railway-inventory.md`. |
| Fan-in | readiness for specification reopen | All notes above | Complete; see `research/fan-in-summary.md`. |

## Parallelism

No write-capable or mutating lanes were used. Railway reads stayed limited to
service inventory/config/deployment status, template search, and docs. Variable
values, DSNs, JWKS contents, bearer tokens, private keys, request bodies, and
dynamic proof URLs were not read or recorded.

## Fan-In Result

Research answered the recorded blocker categories, but left decisions in the
correct owner: `spec.md` must be reopened and repaired. Research does not
approve production database creation, broker deployment, worker deployment,
service-auth rollout, paid authority, or Railway mutation.

## Expected Later Challenge

Formal protected-domain `spec-clarification-challenge` remains required after
the specification reopen produces a coherent candidate spec. It was not run in
this research phase because `spec.md` still needs approval-changing repairs.

## Stop Rule

Stop at research boundary. Do not edit `spec.md`, design artifacts, `tasks.md`,
`test-plan.md`, `rollout.md`, generated artifacts, code, config, or Railway
resources in this phase.

## Next Action

Reopen specification. The next session should reconcile the research notes into
`spec.md`, run or route the required formal clarification challenge, and stop at
the specification boundary.
