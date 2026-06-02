# billing-service

`billing-service` is the GonkaGate internal service boundary for customer money, balances, top-ups, usage reservations, usage finalization, reversals, and billing reconciliation.

The repository was initialized from [`Dankosik/go-service-template-rest`](https://github.com/Dankosik/go-service-template-rest). It keeps the template's OpenAPI-first Go service baseline, agent workflow, CI guardrails, PostgreSQL/sqlc readiness, observability defaults, and Railway deployment policy.

## Product Scope

The product responsibility record is [docs/PRD.md](docs/PRD.md).
The transferred money-math and settlement context is [docs/critical-billing-context.md](docs/critical-billing-context.md).

In short:

- `billing-service` owns the authoritative USD customer ledger and all durable money effects.
- `gonka-proxy` stays the OpenAI-compatible gateway/facade and should not remain a second writer for balances after cutover.
- `pricing-service` owns price truth and quote math inputs.
- `payments-service` owns provider interaction and normalized payment evidence.
- idempotent billing operations are the service contract, not a best-effort implementation detail.

## Current Bootstrap State

The service currently exposes the template operational baseline:

- `GET /health/live`
- `GET /health/ready`
- `GET /metrics`
- `GET /api/v1/ping`

The billing API surface is intentionally captured in the PRD first. Do not add billing business logic without the repository's normal spec/design/tasks handoff.

## Local Commands

```bash
make bootstrap
make run
make test-summary
make openapi-check
```

Use Docker-backed commands when local Go or tooling versions drift from the pinned toolchain:

```bash
make docker-check
make docker-ci
```

## Repository Workflow

Follow [AGENTS.md](AGENTS.md) for the local spec-first workflow. Money, balances, idempotency, provider evidence, and cross-service settlement are protected domains here; non-trivial changes need explicit spec/design/task artifacts before implementation.
