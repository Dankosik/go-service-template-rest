<p align="center">
  <img src=".github/assets/go-service-template-hero.png" alt="Built for Go. Ready for agents. AI-native Go REST API and microservice template" width="100%" />
</p>

<h1 align="center">Go REST API &amp; Microservice Template</h1>

<p align="center">
  An OpenAPI-first Go service template with safe runtime defaults, optional PostgreSQL and agent-workflow profiles, observability, and CI.
</p>

<p align="center">
  <a href="https://github.com/Dankosik/go-service-template-rest/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/Dankosik/go-service-template-rest/actions/workflows/ci.yml/badge.svg?branch=main&amp;event=push"></a>
  <a href="https://github.com/Dankosik/go-service-template-rest/actions/workflows/nightly.yml"><img alt="Nightly reliability" src="https://github.com/Dankosik/go-service-template-rest/actions/workflows/nightly.yml/badge.svg?branch=main&amp;event=schedule"></a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/Dankosik/go-service-template-rest"><img alt="OpenSSF Scorecard" src="https://api.scorecard.dev/projects/github.com/Dankosik/go-service-template-rest/badge"></a>
  <a href="go.mod"><img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/Dankosik/go-service-template-rest"></a>
  <a href="LICENSE"><img alt="MIT License" src="https://img.shields.io/github/license/Dankosik/go-service-template-rest"></a>
</p>

<p align="center">
  <a href="https://github.com/new?template_owner=Dankosik&template_name=go-service-template-rest"><strong>Use this template</strong></a>
  ·
  <a href="#quickstart">Quickstart</a>
  ·
  <a href="#documentation">Documentation</a>
</p>

## Quickstart

Create a repository from the template and start the service:

```bash
gh repo create my-service \
  --template Dankosik/go-service-template-rest \
  --public \
  --clone

cd my-service
make template-init \
  MODULE=github.com/your-org/my-service \
  CODEOWNER=@your-org/backend \
  DATABASE=none \
  HTTP_IDEMPOTENCY=none \
  JOBS=none \
  WEBHOOKS=none \
  OUTBOX=none \
  GRPC=none \
  AUTHN=none \
  OBJECT_STORAGE=none \
  OUTBOUND_AUTH=none \
  MESSAGING=none
make check
make run
```

The defaults create a service with no database dependency. The complete agent
workflow is always retained. Choose `DATABASE=postgres` when the service owns
PostgreSQL. The fixed-authority HTTP client is always retained so feature code
only supplies its dependency target.
<!-- profile:object-storage:start -->
Choose `OBJECT_STORAGE=s3` only when this service needs the S3-compatible
capability. It requires a complete static tuple supplied by deployment and
does not certify a provider or configure a bucket; see
[S3-compatible object storage](docs/s3-compatible-object-storage.md).
<!-- profile:object-storage:end -->
<!-- profile:http-idempotency-postgres:start -->
Choose `HTTP_IDEMPOTENCY=postgres` only with `DATABASE=postgres`. It retains the
one-transaction PostgreSQL idempotency executor; an operation opts in with one
OpenAPI declaration and supplies only its authorized business effect. See
[PostgreSQL HTTP idempotency](docs/postgres-http-idempotency.md).
<!-- profile:http-idempotency-postgres:end -->
<!-- profile:jobs-postgres:start -->
Choose `DATABASE=postgres JOBS=postgres` for typed River jobs inserted in the
business transaction and executed by the separate jobs worker; see
[PostgreSQL background jobs](docs/postgres-durable-background-jobs.md).
<!-- profile:jobs-postgres:end -->
<!-- profile:outbound-auth-oauth2-client-credentials:start -->
Choose `OUTBOUND_AUTH=oauth2-client-credentials` to retain the small factory
that gives a concrete dependency adapter authenticated clients without exposing
tokens. See [outbound machine authentication](docs/outbound-machine-authentication.md).
<!-- profile:outbound-auth-oauth2-client-credentials:end -->
<!-- profile:outbox-postgres:start -->
Choose `DATABASE=postgres OUTBOX=postgres MESSAGING=nats-jetstream` when a
request transaction must durably record a typed outbound event for the
separately deployed River relay. Publication is at-least-once, so consumers
must tolerate duplicate event IDs; see the
[PostgreSQL transactional outbox](docs/postgres-transactional-outbox.md).
<!-- profile:outbox-postgres:end -->
<!-- profile:webhooks-durable:start -->
Choose `DATABASE=postgres JOBS=postgres WEBHOOKS=durable` when a feature
transaction must stage an immutable per-receiver webhook fan-out for the shared
jobs worker. Receiver processing is at-least-once; see
[outbound webhook delivery](docs/outbound-webhook-delivery.md).
<!-- profile:webhooks-durable:end -->
<!-- profile:authn-oidc-jwt:start -->
Choose `AUTHN=oidc-jwt` for OIDC discovery and signed JWT access-token
authentication; see [authentication](docs/authentication.md).
<!-- profile:authn-oidc-jwt:end -->
<!-- profile:authn-oidc-introspection:start -->
Choose `AUTHN=oidc-introspection` for uncached RFC 7662 token introspection;
see [authentication](docs/authentication.md).
<!-- profile:authn-oidc-introspection:end -->
<!-- profile:grpc:start -->
Choose `GRPC=enabled`
when the service publishes or consumes native gRPC; see the
[gRPC guide](docs/grpc.md).
<!-- profile:grpc:end -->
<!-- profile:messaging-nats-jetstream:start -->
Choose `MESSAGING=nats-jetstream` for typed event publishing and handler
registration over a separate durable worker; see [durable messaging](docs/durable-messaging.md).
Together with `DATABASE=postgres OUTBOX=postgres`, it supplies the outbox
relay's concrete NATS producer and W3C trace continuity. The generator rejects
outbox without messaging.
<!-- profile:messaging-nats-jetstream:end -->

`examples/reference-service` is a worked feature slice kept in this template for
reference; initialization removes it so a generated service does not inherit a
second OpenAPI contract to maintain. Pass `REFERENCE_EXAMPLE=keep` to retain
it, and read it here or in
[first production feature](docs/first-production-feature.md) either way.

## What You Get

| Area | Included |
| --- | --- |
| Service foundation | Go 1.26, `chi v5`, `koanf v2`, graceful shutdown, health and readiness |
| API contract | OpenAPI 3.0 and `oapi-codegen v2` with generated request bindings and typed responses |
| Data | No database by default; optional PostgreSQL 17, `pgx v5`, `goose v3`, and `sqlc` profile |
<!-- profile:jobs-postgres:start -->
| Background jobs | Optional River-backed typed jobs with transactional insertion and a separate worker process |
<!-- profile:jobs-postgres:end -->
<!-- profile:outbox-postgres:start -->
| Transactional outbox | Optional River-backed PostgreSQL job appended inside the business transaction and published by a separate NATS worker |
<!-- profile:outbox-postgres:end -->
<!-- profile:webhooks-durable:start -->
| Outbound webhooks | Optional Standard Webhooks job kind with atomic fan-out staging and public-HTTPS enforcement through the shared jobs worker |
<!-- profile:webhooks-durable:end -->
| Outbound HTTP | Standard library by default; optional fixed-authority transport bounds and response-size protection |
<!-- profile:messaging-nats-jetstream:start -->
| Messaging | Optional direct NATS JetStream producer and separate bounded durable pull-consumer worker |
<!-- profile:messaging-nats-jetstream:end -->
<!-- profile:authn-oidc-jwt:start -->
| Authentication | Optional OIDC discovery and RS256 JWT access-token verification for HTTP and native gRPC, with an explicit RFC 9068 profile |
<!-- profile:authn-oidc-jwt:end -->
<!-- profile:authn-oidc-introspection:start -->
| Authentication | Optional uncached RFC 7662 token introspection for HTTP and native gRPC |
<!-- profile:authn-oidc-introspection:end -->
| Observability | OpenTelemetry 1.x traces and metrics, Prometheus export, and structured logs |
| Testing | Race detection and goroutine leak checks; PostgreSQL Testcontainers coverage in the database profile |
| Delivery | Docker and GitHub Actions security gates; opt-in GHCR publishing with Cosign, CycloneDX, and durable migration-history enforcement |
| Agent workflow | The complete Codex, Claude Code, Qwen, Grok Build, and Cursor workflow, always retained ([what that costs](#what-the-agent-workflow-costs)) |

<!-- profile:grpc:start -->
The optional native gRPC profile adds gRPC-Go client/server support, Buf v2,
Edition 2023 Opaque messages, all four RPC cardinalities, standard health, and
bounded drain.
<!-- profile:grpc:end -->

Major versions describe the supported stack. [`go.mod`](go.mod) owns runtime
and test dependencies; [`tools/go.mod`](tools/go.mod) owns development tools.

### What the agent workflow costs

Initialization keeps the workflow byte-for-byte; there is no option to decline
it. A generated service inherits the repository contract, conditional domain
methods, five generic capability roles, and their generated Codex, Claude,
Qwen, Grok, and Cursor carriers. If your team will not use those harnesses, keep
[`AGENTS.md`](AGENTS.md) and `docs/`, and drop `.agents/`, `.codex/`,
`.claude/`, `.cursor/`, `.qwen/`, and `.grok/`.

`specs/` is not in that table because initialization deletes it. Those bundles
record decisions about building this template, and a generated service that kept
them would be handing its agents authoritative-looking records for a repository
it does not have.

This is a working service scaffold, not only a prompt collection. The code,
generated sources, database lifecycle, CI, and agent instructions share one
ownership model.

## Why This Template

Generic agent prompts often miss the constraints that make a Go service safe
to change: package ownership, `context`, error identity, generated sources,
migrations, partial failure, shutdown, and current proof.

Traditional Go templates provide code and commands but rarely tell an agent
how to make a non-trivial change without inventing architecture or declaring
success from an unrelated test.

This template connects both sides:

- one repository contract instead of a heavyweight prompt in every request;
- direct handling for small changes and durable artifacts only when decisions
  must survive;
- OpenAPI, PostgreSQL, telemetry, tests, security, and delivery already wired;
- completion claims tied to fresh evidence of the same scope.

Before adding the first production feature, use the maintained
[first production feature guide](docs/first-production-feature.md).

## How It Works

```mermaid
flowchart LR
    A["Codex / Claude Code / Qwen / Grok Build / Cursor"] --> B["Shared repository contract"]
    B --> C{"Risk-proportional path"}
    C -->|Direct| D["Small local change"]
    C -->|Structured| E["Spec + design + tasks"]
    C -->|Orchestrated| F["Multi-owner coordination"]
    D --> G["Go service change"]
    E --> G
    F --> G
    G --> H["Claim-scoped proof"]
```

The workflow keeps three paths:

- **Direct** — clear, local, reversible work with one owner and bounded proof.
- **Structured** — non-trivial work whose decisions need a `spec.md`,
  `tasks.md`, and only the design or test artifacts that add value.
- **Orchestrated** — broad, multi-owner, hard-to-reverse, or multi-session work.

Public contracts, persisted data, security, money, concurrency, deployment,
and cross-service ownership receive explicit decisions and matching proof
without forcing every task through the heaviest process.

### Autonomous implementation tree

Once Planning has produced a ready implementation ledger, a person launches
orchestration once. The system then runs autonomously until the ledger is
exhausted or it reaches an exact user-owned, external, or unrecoverable native
boundary.

```mermaid
flowchart TD
    user["User<br/>one orchestration launch"]
    orchestrator["LEDGER_ORCHESTRATOR<br/>fills independent ready frontier"]
    lead["ACCEPTANCE_UNIT_LEAD<br/>owns one unit end to end"]
    strategy{"Lead chooses the fastest safe strategy"}
    direct["Lead implements directly"]
    delegated["worker-agent<br/>implement · investigate · verify"]
    fanin["Lead fan-in<br/>integration · proof · self-review"]
    review{"Independent review required?"}
    reviewer["reviewer-agent<br/>independent falsification"]
    receipt["One receipt or precise blocker"]
    done["Ledger exhausted"]

    user --> orchestrator
    orchestrator -->|"dispatches independent ready frontier"| lead
    lead --> strategy
    strategy -->|"handoff costs more"| direct
    strategy -->|"bounded useful work"| delegated
    direct --> fanin
    delegated --> fanin
    fanin --> review
    review -->|"yes"| reviewer
    review -->|"no"| receipt
    reviewer --> receipt
    receipt -->|"canonical transition; refill frontier"| orchestrator
    orchestrator -->|"no ready unit or owner-held recovery"| done
```

The orchestrator does not implement or review units. It computes the ready
frontier and dispatches every mutually independent unit before waiting, within
capacity. Each fresh Lead chooses the simplest reliable workflow, may write
directly, delegates only when the boundary saves time, cost, or context, and
owns integration and acceptance of that unit. Only genuinely independent work
runs concurrently. Recoverable problems may change the route, reuse a useful
agent context, start fresh, or repair the smallest invalid upstream decision
without creating another semantic role.

The detailed contracts live in [Implementation](docs/spec-first-workflow/phases/implementation.md),
[Review](docs/spec-first-workflow/shared/review.md), and the selected [Agent
Harness](docs/agent-harness.md) adapter.

## Agent Support

| Harness | Entry point | Project-native support |
| --- | --- | --- |
| Codex | `AGENTS.md`, `$orchestrator` | `.codex/agents`, `.agents/skills` |
| Claude Code | `CLAUDE.md`, `/orchestrator` | `.claude/agents`, `.claude/skills` |
| Qwen Code | `QWEN.md`, `/orchestrator` | `.qwen/agents`, `.qwen/skills` |
| Grok Build | `Grok.md` | `.grok/agents`, `.grok/roles`, `.agents/skills` |
| Cursor | `AGENTS.md`, Grok 4.6, `/orchestrator` | `.cursor/agents`, `.cursor/rules`, `.agents/skills` |

Five generic capability roles provide evidence, specialist judgment, mutable
work, independent review, and adjudication. Domain skills cover API contracts,
architecture, data, security, reliability, concurrency, testing, delivery, and
Go maintainability only when the current pressure needs them.

See [Agent Harness](docs/agent-harness.md) and
[Spec-First Workflow](docs/spec-first-workflow.md) for the complete routing
contract.

## Repository Layout

```text
cmd/service/                     service entrypoint and bootstrap lifecycle
cmd/migrate/                     migration entrypoint (PostgreSQL profile)
internal/<feature>/              feature-owned business behavior (when added)
internal/config/                 runtime configuration
internal/health/                 readiness and drain behavior
internal/infra/http/             HTTP transport and middleware
internal/infra/httpclient/       fixed-authority outbound HTTP transport
internal/infra/postgres/         PostgreSQL adapters (PostgreSQL profile)
api/openapi/service.yaml         API source of truth
internal/openapi/                generated OpenAPI artifacts
examples/reference-service/      worked feature-slice example (upstream only)
migrations/                      SQL migrations (PostgreSQL profile)
specs/                           durable task decisions (upstream only)
.agents/skills/                  reusable domain methods
.codex/agents/                   generated Codex capability roles
.cursor/agents/                  generated Cursor capability roles
.cursor/rules/                   Cursor harness bootstrap
.grok/agents/                    Grok primary sessions and generated roles
```

<!-- profile:grpc:start -->
With `GRPC=enabled`, `api/proto/` owns protobuf contracts,
`internal/gen/proto/` is generated, `internal/infra/grpc/` owns server policy
and lifecycle, and `internal/infra/grpcclient/` constructs shared bounded
client connections.
<!-- profile:grpc:end -->

Use the [placement guide](docs/project-structure-and-module-organization.md)
before choosing packages or tests.

## Quality Gates

| Command | Purpose |
| --- | --- |
| `make check` | Broad local format, lint, and unit-test baseline |
| `make ci-local` | Native CI aggregate |
| `make check-full` | Native checks plus Docker-backed integration and image gates |
| `BASE_REF=origin/main make pr-check` | Pull-request checks and OpenAPI compatibility |
| `make openapi-check` | OpenAPI generation, drift, runtime, lint, and schema checks |
| `make sqlc-check` | SQL generation drift (PostgreSQL profile) |
| `make migration-check` | Goose source grammar and append-only review history (PostgreSQL profile) |
| `make migration-validate` | Migration rehearsal (PostgreSQL profile) |
| `make test-integration` | Container-backed integration tests when present |
| `make go-security` | Go static security and vulnerability checks |

<!-- profile:grpc:start -->
`make proto-check` owns protobuf format, contract documentation, lint, and
generated-code drift. Repositories retaining proto2/proto3 contracts use
`BASE_REF=origin/main make proto-check` so only paths present with legacy
syntax at that base are accepted; use
`BASE_REF=origin/main make proto-breaking` for compatibility.
<!-- profile:grpc:end -->

Performance work uses the narrowest matching benchmark level. DigitalOcean is
preferred only when `doctl` is already installed and authorized; local
benchmarks remain the supported fallback. See [Benchmarking](docs/benchmarking.md).

## Documentation

- [Repository Architecture](docs/repo-architecture.md)
- [Project Structure and Module Organization](docs/project-structure-and-module-organization.md)
- [Build, Test, and Development Commands](docs/build-test-and-development-commands.md)
- [Spec-First Workflow](docs/spec-first-workflow.md)
- [Agent Harness](docs/agent-harness.md)
- [Benchmarking](docs/benchmarking.md)
- [Railway Deployment Profile](docs/railway-deployment-profile.md)

<!-- profile:authn-bearer:start -->
For trust configuration, token policy, rotation, local testing, and operational
signals, see [Authentication](docs/authentication.md).
<!-- profile:authn-bearer:end -->

<!-- profile:grpc:start -->
For schema, server, client, streaming, lifecycle, and deployment guidance, see
[Native gRPC](docs/grpc.md).
<!-- profile:grpc:end -->

## Community

Contributions are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md), use the
issue forms for bugs and feature proposals, and follow the
[Code of Conduct](CODE_OF_CONDUCT.md).

Report vulnerabilities privately through [SECURITY.md](SECURITY.md).

Released under the [MIT License](LICENSE).
