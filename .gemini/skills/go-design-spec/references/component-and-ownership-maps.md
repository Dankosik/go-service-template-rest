# Component And Ownership Maps

## Behavior Change Thesis
When loaded for unclear package responsibility or source-of-truth ownership, this file makes the model choose explicit owner and dependency-direction records instead of likely mistakes like `internal/common` ownership, generated-file authority, or cross-layer dependencies hidden in a component table.

## When To Load
Load this when the symptom is unclear component ownership: `design/component-map.md` or `design/ownership-map.md` needs package responsibility, source-of-truth, generated-code, dependency-direction, or stable-versus-changing surface decisions.

Do not load this to choose a new service boundary from scratch. If the hard question is decomposition, service extraction, or modular-monolith design, use the architecture specialist and integrate its result here.

## Decision Rubric
- Name surfaces as concrete repo paths or runtime components; avoid capability words without owners.
- Separate "current responsibility", "change", and "stable boundary" so planning can preserve what is not changing.
- Treat generator inputs as authorities and generated outputs as derived unless another repo-owned source says otherwise.
- Keep `internal/app` transport-agnostic and keep concrete adapters in `internal/infra/*` or bootstrap wiring.
- Give every invariant-bearing state one source of truth and a named writer.
- Mark temporary bridges with owner, exit condition, and reconciliation rule, or reject them.

## Imitate

Component map with concrete surfaces and stable boundaries:

```markdown
| Surface | Current responsibility | Change | Stable boundary |
| --- | --- | --- | --- |
| `api/openapi/service.yaml` | REST contract source of truth | Add `POST /orders` request/response schema | Generated code remains derived. |
| `internal/api` | Generated OpenAPI bindings | Regenerate from contract only | Do not hand-edit generated files. |
| `internal/infra/http` | Request mapping, problem responses, route policy | Add order handler adapter | Must not own order business rules. |
| `internal/app/orders` | Transport-agnostic use case behavior | Add create-order service | Must not import HTTP or Postgres packages. |
| `internal/infra/postgres` | Repository and pool-facing persistence | Add order repository implementation | Concrete adapter stays wired from bootstrap. |
```

Ownership map that separates truth, owner, consumers, and rules:

```markdown
| Truth or responsibility | Owner | Consumers | Rule |
| --- | --- | --- | --- |
| REST contract | `api/openapi/service.yaml` | `internal/api`, HTTP adapter, client tests | Change contract first, then regenerate. |
| Order state | `env/migrations` plus Postgres repository | `internal/app/orders` through a narrow contract if inversion is needed | Do not let HTTP handlers write persistence directly. |
| Use-case policy | `internal/app/orders` | HTTP adapter, future workers | Keep transport details out of app behavior. |
| Adapter wiring | `cmd/service/internal/bootstrap` | Service binary | Concrete dependencies are admitted and wired only at the composition root. |
```

Stable/change split:

```markdown
Stable: startup/shutdown remains owned by bootstrap; app services remain transport-agnostic.
Changing: OpenAPI path, HTTP adapter mapping, order use case, Postgres repository, migration.
```

## Reject

Component map with vague ownership:

```markdown
`internal/common` owns shared order helpers used by HTTP, app, and Postgres.
```

Why it is bad: "common" hides ownership and encourages cross-layer coupling.

Ownership map that violates dependency direction:

```markdown
`internal/app/orders` calls `internal/infra/http` to format problem responses.
```

Why it is bad: app behavior must stay transport-agnostic.

Generated-code authority:

```markdown
Edit `internal/api` to change the request shape.
```

Why it is bad: generated bindings are derived from `api/openapi/service.yaml`.

## Agent Traps
- A component is marked stable while the sequence requires new behavior from it.
- Two components claim write authority for the same invariant-bearing state.
- `internal/app` depends on a concrete `internal/infra/*` package without an approved inversion boundary.
- A generated surface is treated as the source of truth instead of the generator input.
- A component map says "new worker" but ownership map has no bootstrap or lifecycle owner.
- A data artifact says cache/projection is derived-only, but ownership map gives it write authority.
- A "temporary" bridge has no owner, exit condition, or reconciliation rule.

## Validation Shape
Before handoff, every changed surface should have one responsibility, one owner for source-of-truth decisions, and an explicit stable boundary. If the table cannot say who owns a write, block planning or route to the relevant specialist.

## Escalation Rules
- Escalate to `go-architect-spec` when the map exposes a real boundary, runtime, service extraction, or module ownership decision.
- Escalate to `go-data-architect-spec` when source-of-truth ownership depends on schema, cache, projection, ledger, retention, or migration shape.
- Escalate to `api-contract-designer-spec` when the component boundary changes client-visible REST behavior.
- Escalate to `go-security-spec` when trust boundary, identity propagation, tenant isolation, or fail-closed authorization owns the map.
- Block design readiness when two surfaces claim the same source of truth or when dependency direction is unresolved.

## Repo Pointers
- `docs/repo-architecture.md`: stable component boundaries, source-of-truth table, dependency direction, request/response path, startup/shutdown path, async extension path.
- `docs/spec-first-workflow.md`: required `design/component-map.md` and `design/ownership-map.md` purpose.
- `.agents/skills/technical-design-session/SKILL.md`: required design artifacts and handoff boundary.
