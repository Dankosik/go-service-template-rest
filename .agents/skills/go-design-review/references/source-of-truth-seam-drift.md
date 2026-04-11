# Source-Of-Truth Seam Drift

## When To Load
Load this when the diff spreads one stable policy across files, edits derived/generated code, duplicates config or migration rules, or creates competing owners for contracts, validation, classification, mapping, or normalization.

Repository-approved specs/design docs and canonical repo sources win. External sources only help explain why a single owner is easier to review and evolve.

## Concrete Review Examples
Finding example: generated OpenAPI bindings are edited by hand to add a field.

```text
[critical] [go-design-review] internal/api/server.gen.go:219
Issue: The diff changes generated API code instead of the OpenAPI contract source.
Impact: Regeneration can discard the behavior, and reviewers cannot tell whether the REST contract intentionally changed.
Suggested fix: Update `api/openapi/service.yaml`, regenerate the bindings, then keep manual runtime mapping in `internal/infra/http`.
Reference: `docs/repo-architecture.md` source-of-truth table.
```

Finding example: request size parsing is added in both middleware and one handler.

```text
[medium] [go-design-review] internal/infra/http/widgets.go:73
Issue: Request limit policy is now split between shared middleware and endpoint-local parsing.
Impact: Future changes can update one path but not the other, producing endpoint-specific behavior not visible in the transport policy.
Suggested fix: Put the stable policy behind one seam-named helper in `internal/infra/http` or route it through the existing middleware owner.
Reference: task `design/ownership-map.md` if present; otherwise HTTP edge ownership in `docs/repo-architecture.md`.
```

Finding example: config precedence is reimplemented in bootstrap for one dependency.

```text
[high] [go-design-review] cmd/service/internal/bootstrap/cache.go:44
Issue: Bootstrap now reinterprets env and flag precedence for cache settings instead of consuming the validated config snapshot.
Impact: The cache path can diverge from `internal/config` validation and secret policy, making startup behavior depend on two config owners.
Suggested fix: Add the field and validation to `internal/config`, then consume the typed value from bootstrap.
Reference: repository config source policy and `docs/repo-architecture.md` config ownership.
```

Finding example: a durable schema assumption is encoded in repository code without a migration or design note.

```text
[high] [go-design-review] internal/infra/postgres/orders.go:102
Issue: The repository now assumes a new `archived_at` column, but the migration source of truth did not change.
Impact: Code and database shape can drift across environments; tests using fakes may stay green while deployed queries fail.
Suggested fix: Add the migration and align repository code with the schema change, or remove the new assumption until the data design is approved.
Reference: `docs/repo-architecture.md` migration source-of-truth row.
```

## Non-Findings To Avoid
- Do not require extraction for one-off local logic that is not stable policy and has no second consumer.
- Do not flag duplicated test setup when it is intentionally local and avoids coupling tests to a shared fixture with hidden state.
- Do not flag generated code changes when the generator is the actual source and the diff is the regenerated output from a canonical input change.
- Do not ask for a global helper package to solve source-of-truth spread. Prefer the narrowest owning package.

## Smallest Safe Correction
- Update the canonical source, then regenerate or adapt derived consumers.
- Collapse repeated stable policy into a seam-named helper or type in the owning package.
- Delete local copies that reinterpret config, contract, migration, or classification rules.
- Cite the repository source of truth directly in the review finding so the fix is clear.

## Escalation Rules
- Escalate to `api-contract-designer-spec` when the canonical API contract itself needs a new resource, status, idempotency, async, or error decision.
- Escalate to `go-data-architect-spec` or `go-db-cache-spec` when schema ownership, transactions, projections, cache behavior, or data consistency must change.
- Escalate to `go-design-spec` when no existing package clearly owns the stable policy.
- Hand off to `go-qa-review` when missing tests are the only proof gap after the source-of-truth owner is correct.

## Exa Source Links
- [Organizing a Go module - The Go Programming Language](https://go.dev/doc/modules/layout)
- [Go Code Review Comments - Package Names](https://go.dev/wiki/CodeReviewComments)
- [arc42 Section 9 - Architecture Decisions](https://docs.arc42.org/section-9/)
- [Architecture Decision Record - Martin Fowler](https://martinfowler.com/bliki/ArchitectureDecisionRecord.html)
