# Repository Architecture

This is the architecture front door for `go-service-template-rest`. Select only
the leaf whose pressure can change the current decision; code, contracts,
generated sources, tests, and package documentation remain final factual
authority.

## Global Invariants

1. Business behavior stays in `internal/<feature>`; transports and providers
   stay in `internal/infra`; concrete wiring and process lifecycle stay in an
   explicit `cmd/<binary>` composition root.
2. Generated code is derived. Edit the contract or generation input first,
   regenerate, and prove drift with its repository-owned check.
3. One fact has one canonical writer. A projection, transport, cache, or
   generated view does not become authority because it is easier to inspect.
4. Optional profiles remain inert unless selected and fully wired; a partial
   capability is not a supported architecture.
5. Trust, ingress/egress, dependency criticality, failure/recovery, and rollout
   choices are explicit at the boundary that enforces them.
6. A task-local design may retain, replace, or remove current architecture, but
   it must name the changed owner and proving surface.

## Source Of Truth

| Source | Derived or consuming surfaces |
| --- | --- |
| `api/openapi/service.yaml` | `internal/openapi/` and HTTP transport wiring |
| `api/proto/**/*.proto` when enabled | `internal/gen/proto/` and bootstrap registration |
| `internal/config/` build and validation | Runtime snapshot consumed by bootstrap and adapters |
| `env/config/*.yaml`, `APP__...`, runtime flags | Inputs whose precedence/secret rules live in [Configuration Source Policy](configuration-source-policy.md) |
| `migrations/*.sql` | Database shape consumed by runtime and generated SQL access |
| `internal/<feature>/*` | Behavior consumed by transports, jobs, and future binaries |
| `cmd/<binary>/internal/bootstrap/*` | That binary's composition and lifecycle |

Concrete adapter wiring belongs in the composition root. Generated outputs are
never edited as the source of truth.

## Domain Vocabulary

Keep only accepted cross-task terms whose interpretation changes behavior,
violation outcome, authority, proof, or handoff. Task-local or unsettled terms
stay in their owning specification.

| Term | Means here | Does not mean | Authority source | Semantic owner | Decision affected |
| --- | --- | --- | --- | --- | --- |

The template defines no service-specific terms. Derived services add rows only
for stable domain decisions.

## Select One Leaf

| Changed pressure | Load |
| --- | --- |
| Package/component ownership, dependency direction, generated/manual boundary, or composition root | [Component Boundaries](architecture/boundaries.md) |
| HTTP contract, routing, middleware, exposure, or handler composition | [HTTP Architecture](architecture/http.md) |
| Startup, readiness, drain, shutdown, or process-resource lifecycle | [Runtime Lifecycle](architecture/runtime-lifecycle.md) |
| Queue, outbox, job, webhook, event, or background process | [Async Architecture](architecture/async.md) |
| System neighbor, outbound provider, cross-service contract, or runtime evidence path | [Integration Boundaries](architecture/integration.md) |
| Native gRPC transport or client | [Native gRPC](grpc.md) |
| PostgreSQL pool, repository, migration, transaction, or query | [Persistence Architecture](architecture/persistence.md) |
| Caller identity or OIDC/JWT trust | [Authentication](authentication.md) |
| S3-compatible object storage | [S3-Compatible Object Storage](s3-compatible-object-storage.md) |
| Configuration source, precedence, or secret input | [Configuration Source Policy](configuration-source-policy.md) |
| File placement or full repository tree | [Project Structure](project-structure-and-module-organization.md) |
| Build, generation, and validation commands | [Build, Test, and Development Commands](build-test-and-development-commands.md) |
| Delivery gates and production readiness | [CI/CD Production-Ready Checklist](ci-cd-production-ready.md) |

Load another leaf only for an independent changed pressure.
