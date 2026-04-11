# Approved Decision Conformance

## When To Load
Load this when the diff appears to introduce behavior, ownership, lifecycle, fallback, contract, or delivery decisions that are not present in the approved `spec.md`, `design/`, `plan.md`, or repository baseline.

This reference is about review conformance, not creating new architecture. If the code reveals a better design, report a design escalation rather than rewriting the plan in the review.

## Concrete Review Examples
Finding example: the approved plan says synchronous request handling, but the diff adds a background goroutine and in-memory queue.

```text
[critical] [go-design-review] internal/infra/http/imports.go:97
Issue: The endpoint now defers work to an in-memory background queue, but the approved spec describes synchronous request handling.
Impact: The diff silently changes durability, shutdown, retry, and response semantics without a design decision or proof path.
Suggested fix: Restore the synchronous flow, or reopen the spec/design to decide async ownership, lifecycle, persistence, and validation.
Reference: task `spec.md` Decisions and `design/sequence.md` if present.
```

Finding example: code adds a fallback to skip dependency admission when a service is slow.

```text
[high] [go-design-review] cmd/service/internal/bootstrap/dependencies.go:58
Issue: Startup admission now falls back to serving when a dependency probe times out, but the approved lifecycle model requires dependency validation before accepting traffic.
Impact: This changes availability and correctness semantics in code rather than in the reliability/design decision record.
Suggested fix: Remove the fallback or route the new fail-open policy through approved reliability/design work.
Reference: `docs/repo-architecture.md` startup path and task reliability decisions if present.
```

Finding example: the diff leaves TODO-driven ownership for a later phase.

```text
[medium] [go-design-review] internal/app/imports/service.go:114
Issue: The TODO leaves classification ownership to be decided after merge while the code already branches on that classification.
Impact: A planning-time decision becomes a hidden runtime contract, making the next reviewer inherit an undocumented seam.
Suggested fix: Either keep the behavior behind the currently approved owner or reopen the design to assign classification ownership.
Reference: task `plan.md` and `tasks.md` phase scope if present.
```

Finding example: a PR updates code generated from a contract but the spec/decision artifact says contract changes are out of scope.

```text
[high] [go-design-review] api/openapi/service.yaml:142
Issue: The API schema changes despite the approved scope excluding contract behavior.
Impact: Clients, generated handlers, and validation obligations change without the contract review path.
Suggested fix: Remove the schema change from this diff or reopen the spec for an API-contract decision and hand off to the API reviewer.
Reference: task `spec.md` Scope / Non-goals.
```

## Non-Findings To Avoid
- Do not demand an ADR for every local helper or refactor. Decision documentation matters for important, risky, expensive, or structural choices.
- Do not flag code for lacking a spec when the task is an eligible tiny/direct-path fix and no approved artifact exists.
- Do not override repo intent with external best-practice links. If the approved spec is clear, review against it.
- Do not conflate an implementation detail with a hidden decision unless it changes ownership, external behavior, lifecycle, or validation obligations.

## Smallest Safe Correction
- Restore the approved behavior and keep the review finding local.
- If the new behavior is intentional, require a spec/design update rather than accepting the code as the new authority.
- Add a compact design escalation note that identifies the missing decision, affected owner, and proof obligation.
- Cite the exact approved section when possible.

## Escalation Rules
- Escalate to `go-design-spec` or `go-architect-spec` when the code proposes a new structural decision.
- Escalate to `api-contract-designer-spec` when conformance drift changes client-visible REST behavior.
- Escalate to `go-reliability-spec` when conformance drift changes timeout, retry, fallback, startup, shutdown, or degradation semantics.
- Escalate to `go-devops-spec` when conformance drift changes generated-code, CI, rollout, release, or compatibility policy.

## Exa Source Links
- [arc42 Section 9 - Architecture Decisions](https://docs.arc42.org/section-9/)
- [Example Decision: Use ADRs in Nygard format - arc42](https://docs.arc42.org/examples/decision-use-adrs/)
- [Architecture Decision Record - Martin Fowler](https://martinfowler.com/bliki/ArchitectureDecisionRecord.html)
- [Decision record template by Michael Nygard](https://github.com/joelparkerhenderson/architecture-decision-record/blob/main/locales/en/templates/decision-record-template-by-michael-nygard/index.md)
