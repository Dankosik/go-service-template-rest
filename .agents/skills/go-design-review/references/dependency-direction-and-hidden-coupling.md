# Dependency Direction And Hidden Coupling

## When To Load
Load this when imports, callbacks, global registries, package init side effects, test helper reuse, or adapter wiring change who depends on whom.

Start from approved task-local design and `docs/repo-architecture.md`. Use the external links only to calibrate Go package direction, interface ownership, and decision-record escalation.

## Concrete Review Examples
Finding example: `internal/app/billing` imports `internal/infra/postgres` to call a repository concrete type.

```text
[critical] [go-design-review] internal/app/billing/service.go:31
Issue: The app package now imports the concrete Postgres adapter, reversing the approved inward dependency direction.
Impact: Business behavior becomes coupled to datastore mechanics, making alternate adapters, tests, and future workers inherit Postgres lifecycle concerns.
Suggested fix: Have bootstrap construct the Postgres repository and pass it into the app service; introduce a consumer-owned app/domain interface only if multiple implementations or tests need it.
Reference: `docs/repo-architecture.md` stable dependency direction; Go review guidance on interfaces belonging with the consumer.
```

Finding example: an infra package registers itself through `init()` so bootstrap no longer wires it explicitly.

```text
[high] [go-design-review] internal/infra/queue/register.go:14
Issue: Adapter registration now happens through package initialization instead of the explicit bootstrap composition root.
Impact: Runtime behavior depends on import side effects, so dependency admission, config, and shutdown ownership become harder to audit.
Suggested fix: Remove the registration side effect and wire the adapter explicitly from bootstrap.
Reference: task `design/sequence.md` if present; otherwise `docs/repo-architecture.md` startup path.
```

Finding example: a test helper under `internal/infra/http` is imported by app tests because it contains useful fixture setup.

```text
[medium] [go-design-review] internal/app/health/service_test.go:12
Issue: App tests now depend on HTTP adapter fixtures, making transport setup a hidden dependency of app behavior.
Impact: Future changes to HTTP middleware or generated handler setup can break app tests for reasons unrelated to the use case.
Suggested fix: Move the fixture to an app-owned test helper or build the app input directly in the test.
Reference: approved app/HTTP boundary in `docs/repo-architecture.md`.
```

## Non-Findings To Avoid
- Do not flag dependency direction that is explicitly allowed for bootstrap; the composition root is supposed to know concrete adapters.
- Do not flag an interface just because it exists. Flag producer-owned or premature interfaces when they blur the consuming package's needs.
- Do not flag generated imports solely for looking unusual; first verify whether they are derived from the repository's canonical contract or generator.
- Do not treat every callback as hidden coupling. The issue is whether lifecycle, ownership, or dependency direction becomes implicit.

## Smallest Safe Correction
- Move concrete construction to bootstrap or the approved composition root.
- Replace side-effect registration with explicit wiring.
- Put small interfaces in the consuming package when abstraction is necessary.
- Keep test helpers near the layer they model; pass plain values instead of importing a higher-level adapter helper.

## Escalation Rules
- Escalate when the diff needs a new dependency inversion point, not just a relocated import.
- Hand off to `go-concurrency-review` when callbacks or registrations add goroutines, channels, worker pools, or shutdown behavior.
- Hand off to `go-reliability-review` when hidden coupling affects timeouts, retries, fallback, or lifecycle admission.
- Hand off to `go-security-review` when hidden coupling weakens a trust boundary or authorization point.

## Exa Source Links
- [Organizing a Go module - The Go Programming Language](https://go.dev/doc/modules/layout)
- [Go Code Review Comments - Interfaces](https://go.dev/wiki/CodeReviewComments)
- [arc42 Section 9 - Architecture Decisions](https://docs.arc42.org/section-9/)
- [Decision record template by Michael Nygard](https://github.com/joelparkerhenderson/architecture-decision-record/blob/main/locales/en/templates/decision-record-template-by-michael-nygard/index.md)
