# Source-Of-Truth Seam Drift

## Behavior Change Thesis
When loaded for symptom "generated, config, migration, contract, or stable policy ownership split," this file makes the model route the fix through the canonical owner or a narrow owning-package seam instead of accepting local copies, hand edits to derived files, or a global helper package.

## When To Load
Load this when a diff edits derived/generated code, duplicates config or migration rules, spreads one stable policy across files, or creates competing owners for contracts, validation, classification, mapping, or normalization.

Prefer `accidental-complexity-and-helper-buckets.md` when the primary symptom is speculative abstraction rather than source ownership.

## Decision Rubric
- Generated output is acceptable only when it follows a canonical input change; hand edits to generated output are source-of-truth drift.
- Config precedence, validation, migration shape, and API contracts should have one canonical source before runtime code consumes them.
- Repeated stable policy deserves one seam-named helper or type in the owning package; one-off logic and intentionally local test setup do not.
- Do not solve source spread by adding `common`, `util`, or an owner-neutral package.
- If no package clearly owns the stable policy, request design escalation rather than picking an owner in the review comment.

## Imitate
```text
[critical] [go-design-review] internal/api/server.gen.go:219
Issue: The diff changes generated API code instead of the OpenAPI contract source.
Impact: Regeneration can discard the behavior, and reviewers cannot tell whether the REST contract intentionally changed.
Suggested fix: Update `api/openapi/service.yaml`, regenerate the bindings, then keep manual runtime mapping in `internal/infra/http`.
Reference: `docs/repo-architecture.md` source-of-truth table.
```

Copy this shape when derived output changed without its canonical input.

```text
[high] [go-design-review] cmd/service/internal/bootstrap/cache.go:44
Issue: Bootstrap now reinterprets env and flag precedence for cache settings instead of consuming the validated config snapshot.
Impact: The cache path can diverge from `internal/config` validation and secret policy, making startup behavior depend on two config owners.
Suggested fix: Add the field and validation to `internal/config`, then consume the typed value from bootstrap.
Reference: `docs/repo-architecture.md` config ownership.
```

Copy this shape when runtime code duplicates an already-owned policy.

```text
[medium] [go-design-review] internal/infra/http/widgets.go:73
Issue: Request limit policy is now split between shared middleware and endpoint-local parsing.
Impact: Future changes can update one path but not the other, producing endpoint-specific behavior not visible in the transport policy.
Suggested fix: Put the stable policy behind one seam-named helper in `internal/infra/http` or route it through the existing middleware owner.
Reference: task `design/ownership-map.md` if present; otherwise HTTP edge ownership in `docs/repo-architecture.md`.
```

Copy this shape when repeated policy is stable enough to deserve one local owner.

## Reject
```text
[medium] [go-design-review] internal/infra/http/widgets.go:73
Issue: This repeats code.
Suggested fix: Move it to `internal/common`.
```

Reject because it flags duplication without proving stable policy and proposes an owner-neutral bucket.

```text
[high] [go-design-review] internal/api/server.gen.go:219
Issue: Generated code changed.
Suggested fix: Do not commit generated code.
```

Reject because regenerated output is fine when the canonical input changed; the finding must distinguish hand edit from expected generation.

## Agent Traps
- Do not require extraction for one-off local logic.
- Do not flag duplicated tests when local setup is clearer and avoids hidden fixture state.
- Do not treat a fake or mock as the source of truth for durable schema, contract, or config behavior.

## Validation Shape
Check the canonical source and derived surfaces together: OpenAPI input plus generated code, migrations plus repository queries, config parser plus bootstrap consumers, and owning helper plus all repeated call sites. Proof means the canonical owner and consumers move in sync.
