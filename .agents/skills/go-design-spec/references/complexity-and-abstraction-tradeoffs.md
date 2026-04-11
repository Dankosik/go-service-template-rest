# Complexity And Abstraction Tradeoffs

## When To Load
Load this when the design proposes or removes:
- a new interface, adapter, helper package, or layer
- a shared `common`, `utils`, `manager`, `factory`, or `service` abstraction
- a new package boundary
- a simplification that might weaken an API, data, reliability, or security contract
- a "future-proof" indirection with unclear present-day value

Do not load this to do line-level Go implementation cleanup. This reference is for design-bundle trade-offs before planning.

## Good Examples

Good same-package policy extraction:

```markdown
Selected option: keep request-to-domain mapping in `internal/infra/http` and extract one same-package helper for repeated problem-response mapping.

Why now:
- Three handlers need the same mapping policy.
- The policy belongs to the HTTP adapter, not `internal/app`.
- A cross-package helper would widen the ownership boundary without adding reuse value.

Rejected:
- `internal/common/errors`: rejected because it would mix transport formatting with app/domain errors.
- Per-handler copies: rejected because the mapping policy would drift across handlers.
```

Good app-facing interface:

```markdown
Selected option: define a narrow repository contract owned by `internal/app/orders` only because the app use case needs persistence inversion and the concrete implementation belongs in `internal/infra/postgres`.

Rejected:
- Interface-per-struct for every Postgres type; rejected because callers do not need those contracts.
- App importing Postgres directly; rejected because it violates the repository dependency direction.
```

Good "no abstraction" decision:

```markdown
Selected option: keep the two call sites direct.

Why:
- The code paths differ in validation and error semantics.
- A helper would hide the difference and force option flags.
- No stable policy is repeated yet.
```

## Bad Examples

Bad future-proofing:

```markdown
Create `internal/platform/service/manager/factory` so future transports can plug in later.
```

Why it is bad: no current policy or ownership problem is removed, and the change makes planning reason through invented layers.

Bad generic helper:

```markdown
Move order validation, HTTP mapping, tenant checks, and SQL formatting into `internal/common/orderutils`.
```

Why it is bad: it mixes abstraction levels and blurs API, app, security, and persistence ownership.

Bad simplification that weakens a contract:

```markdown
Remove idempotency handling because it makes the sequence shorter.
```

Why it is bad: simpler control flow is not acceptable if it removes a reliability or API correctness guarantee.

## Contradictions To Detect
- A new abstraction claims to reduce duplication but has only one call site and no stable policy.
- An interface is producer-owned even though no caller needs inversion or test seam ownership.
- A helper crosses transport, app, and persistence layers.
- A simplification removes validation, authorization, idempotency, retry bounds, migration safety, or observability correlation.
- A "shared" package becomes the de facto owner of a domain invariant without being named in the ownership map.
- A design says "modular" while all modules still coordinate releases through shared schema, shared helpers, or direct DB access.

## Escalation Rules
- Escalate to `go-architect-spec` when the abstraction changes module, runtime, service, or ownership boundaries.
- Escalate to `go-data-architect-spec` when the abstraction hides data ownership, schema evolution, cache, or transaction boundaries.
- Escalate to `api-contract-designer-spec` when simplification changes client-visible behavior, idempotency, pagination, status semantics, or error shape.
- Escalate to `go-security-spec` when the simplification touches authn/authz, tenant isolation, validation, SSRF, injection, or secret handling.
- Escalate to `go-reliability-spec` when the simplification changes timeout, retry, fallback, overload, startup, shutdown, or recovery behavior.
- Reject the abstraction in the design bundle when it lacks a present-day complexity reduction or widens change radius without clear benefit.

## Repo-Native Sources
- `docs/repo-architecture.md`: dependency direction, component boundaries, and adapter/app/domain ownership rules.
- `.agents/skills/go-design-spec/SKILL.md`: design readiness bar and complexity/maintainability stance.
- `.agents/skills/go-architect-spec/SKILL.md`: boundary and decomposition evidence requirements.
- `.agents/skills/go-language-simplifier-review/SKILL.md` and `.agents/skills/go-idiomatic-review/SKILL.md`: review-time smell vocabulary to keep design trade-offs honest.

## Source Links Gathered Through Exa
- arc42 building block view: https://docs.arc42.org/section-5
- arc42 architecture decisions: https://docs.arc42.org/section-9/
- arc42 risks and technical debt: https://docs.arc42.org/section-11/
- C4 component diagram: https://c4model.com/diagrams/component
- Michael Nygard, "Documenting Architecture Decisions": http://thinkrelevance.com/blog/2011/11/15/documenting-architecture-decisions
- ISO/IEC/IEEE 42010 getting started: http://www.iso-architecture.org/42010/getting-started.html
