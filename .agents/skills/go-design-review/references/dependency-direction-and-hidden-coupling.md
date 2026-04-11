# Dependency Direction And Hidden Coupling

## Behavior Change Thesis
When loaded for symptom "imports, callbacks, registration, or test helpers changed who depends on whom," this file makes the model review the dependency mechanism and composition root explicitly instead of treating the change as a stylistic import issue or asking for interfaces everywhere.

## When To Load
Load this when a diff changes import direction, concrete adapter wiring, package `init` registration, global registries, callbacks, or cross-layer test helper reuse.

Prefer `boundary-and-ownership-drift.md` when the main problem is misplaced behavior even without a new coupling mechanism.

## Decision Rubric
- Flag inward packages importing concrete outward adapters unless bootstrap or another approved composition root is doing the wiring.
- Flag package-init registration and global registries when they make runtime dependency admission, config, or shutdown ownership implicit.
- Do not flag explicit registration in an approved composition root solely because it wires concrete adapters there; bootstrap is where that dependency admission belongs.
- Treat package `init` or blank-import registration as suspect outside `main`, tests, or stateless/idempotent format-style registration; even in bootstrap, ask whether it hides lifecycle, config, shutdown, or test-isolation ownership.
- Flag test helper imports when a lower-level or app test now depends on adapter setup that can break for unrelated reasons.
- Do not flag generated imports solely because they look unusual; first check whether they follow the canonical generator path.
- Do not require an interface by default. Use a small consumer-owned interface only when the consuming package needs inversion.

## Imitate
```text
[critical] [go-design-review] internal/app/billing/service.go:31
Issue: The app package now imports the concrete Postgres adapter, reversing the approved inward dependency direction.
Impact: Business behavior becomes coupled to datastore mechanics, so alternate adapters, workers, and app tests inherit Postgres lifecycle concerns.
Suggested fix: Have bootstrap construct the Postgres repository and pass it into the app service; introduce a consumer-owned app/domain interface only if the use case needs inversion.
Reference: `docs/repo-architecture.md` stable dependency direction.
```

Copy this shape when the import itself is the evidence of a dependency direction change.

```text
[high] [go-design-review] internal/infra/queue/register.go:14
Issue: Adapter registration now happens through package initialization instead of the explicit bootstrap composition root.
Impact: Runtime behavior depends on import side effects, so dependency admission, config, and shutdown ownership become harder to audit.
Suggested fix: Remove the registration side effect and wire the adapter explicitly from bootstrap.
Reference: task `design/sequence.md` if present; otherwise `docs/repo-architecture.md` startup path.
```

Copy this shape when the coupling is hidden behind side effects rather than a direct import.

## Reject
```text
[medium] [go-design-review] internal/app/billing/service.go:31
Issue: This import looks wrong.
Suggested fix: Add `BillingRepository` interface to the postgres package.
```

Reject because it names neither the dependency direction nor the consumer-owned abstraction rule.

```text
[low] [go-design-review] cmd/service/internal/bootstrap/app.go:22
Issue: Bootstrap imports Postgres, which couples the binary to the database.
```

Reject because bootstrap is normally the approved composition root for concrete adapters.

## Agent Traps
- Do not confuse dependency direction with package name preference; the risk is hidden lifecycle, ownership, or adapter coupling.
- Do not call every callback hidden coupling. Ask whether the callback smuggles lifecycle, config, authorization, retry, or shutdown policy across owners.
- Do not file both this and a boundary finding for the same line unless there are two independent merge risks.

## Validation Shape
Verify the dependency path with imports, construction sites, registration side effects, and tests that now import across layers. The strongest proof is that concrete adapter construction and lifecycle admission remain explicit in the approved composition root.
