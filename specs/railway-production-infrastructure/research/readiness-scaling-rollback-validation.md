# Readiness, Scaling, Rollback, And Validation

Status: targeted full-rollout research complete
Date: 2026-06-02

## Question

What proof surfaces must the specification preserve for app readiness, worker
readiness, scaling, rollback, and secret-free validation?

## Readiness Findings

The HTTP app has Railway healthcheck path `/health/ready`
(`railway.toml:16`). Postgres readiness is enabled by default when Postgres is
enabled, while Redpanda readiness is disabled by default in app config
(`env/config/default.yaml:115`). If the production app should fail readiness
when broker connectivity is missing, specification must require
`redpanda_readiness_probe=true` or an equivalent proof.

The worker has an internal readiness model, not an HTTP readiness endpoint. It
probes Postgres and the broker before entering ready state
(`cmd/billing-worker/internal/bootstrap/runtime.go:132`), and it exits cleanly
when worker mode is disabled (`cmd/billing-worker/internal/bootstrap/run.go:45`).
Railway worker health therefore needs an explicit future design decision:
disable HTTP health checks for the worker and prove readiness by logs/metrics,
or add a worker health surface before deploy.

## Scaling Findings

`railway.toml` records an app replica baseline comment of `>=2`, but live
read-back currently shows the app service as one active deployment. Full
rollout must decide desired app replicas and prove the Railway service settings
rather than rely on comments.

Worker tasks are fixed at `MaxConcurrency: 1`, and the three event consumers
share one consumer group. Scaling beyond one worker replica may be valid only
after the spec/design decide partitions, assignment behavior, idempotency proof,
lag proof, and safe reconciliation/outbox concurrency.

## Rollback Findings

App rollback can use Railway deployment rollback/redeploy semantics, but the
full rollout adds stateful rollback domains:

- Postgres restore/PITR restores to staged volume or sibling service and needs
  explicit manual cutover policy;
- broker topic cleanup or retention changes should not be destructive without
  reviewed approval;
- worker rollback must cover stop/drain behavior and not lose uncommitted
  terminal/checkpoint/close processing;
- paid authority rollback must remain fail-closed and avoid reintroducing
  Redis or legacy direct reserve authority as an implicit fallback.

## Validation Findings

Repository-owned proof surfaces from
`docs/build-test-and-development-commands.md` include:

- `make guardrails-check` for repository guardrails;
- `make migration-validate MIGRATION_DSN=...` or Docker migration rehearsal for
  `env/migrations` up/down/up;
- generated-artifact checks such as `make sqlc-check` and `make openapi-check`;
- integration tests when the changed surface requires them.

Production validation must remain secret-free: record deployment IDs, service
IDs, status, migration version/dirty state, topic names, lag summaries, and
key names only. Do not print DSNs, variable values, JWTs, JWKS documents,
private keys, request bodies, or dynamic proof URLs.

## Handoff Implications

Specification reopen must decide:

- app readiness dependency set, including whether broker readiness gates app
  readiness;
- worker health/readiness proof before a Railway worker service can be
  approved;
- app and worker replica baselines;
- broker lag and topic proof;
- restore drill and migration-version proof;
- rollback order across app, worker, database, broker, variables, and paid
  authority.
