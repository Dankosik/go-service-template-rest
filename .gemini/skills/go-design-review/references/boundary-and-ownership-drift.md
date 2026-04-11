# Boundary And Ownership Drift

## Behavior Change Thesis
When loaded for symptom "behavior moved across a component or package boundary," this file makes the model identify the owning boundary and request the smallest move back to that owner instead of giving a generic layering lecture or redesigning the system.

## When To Load
Load this when behavior, policy, or construction moves across app, domain, infra, HTTP, config, bootstrap, telemetry, or migration boundaries.

Prefer narrower references when the primary symptom is import direction, generated/config source drift, or helper abstraction shape.

## Decision Rubric
- Make a finding when the diff changes who owns a responsibility, not merely where code looks tidy.
- Anchor the owner in task `design/ownership-map.md`, `spec.md`, or `docs/repo-architecture.md`; if no owner exists, make a design escalation instead of inventing one.
- Keep fixes local: move behavior back to the owner, pass already validated values through bootstrap, or keep transport mapping at the transport edge.
- Do not require a new abstraction just because a boundary exists; concrete adapters wired by bootstrap are often the simpler approved shape.
- Treat a small same-package helper as fine unless it hides owner drift, source-of-truth spread, or dependency reversal.

## Imitate
```text
[high] [go-design-review] internal/app/orders/service.go:42
Issue: The app layer now builds HTTP problem responses, moving transport response policy into the transport-agnostic app boundary.
Impact: Future non-HTTP callers inherit HTTP semantics and must account for adapter concerns that should stay at the edge.
Suggested fix: Return an app/domain error shape from `internal/app/orders` and keep HTTP response mapping in `internal/infra/http`.
Reference: task `design/ownership-map.md` if present; otherwise `docs/repo-architecture.md` app and HTTP ownership rows.
```

Copy this shape when a local convenience crosses a clear component owner: name the owner, the crossed behavior, and the smallest move back.

```text
[high] [go-design-review] internal/infra/http/widgets.go:88
Issue: The HTTP handler now reads env/config directly, taking over config precedence that belongs to `internal/config` and bootstrap wiring.
Impact: Request handling can diverge from startup validation even while endpoint tests stay green.
Suggested fix: Pass the already validated config value through the composition root or add an approved option owned by bootstrap.
Reference: `docs/repo-architecture.md` config and bootstrap ownership rows.
```

Copy this shape when the wrong package owns a policy lookup rather than just a helper call.

## Reject
```text
[medium] [go-design-review] internal/app/orders/service.go:42
Issue: This violates clean architecture.
Suggested fix: Add an interface.
```

Reject because it skips the repository owner, the concrete merge risk, and the smallest safe correction.

```text
[low] [go-design-review] internal/app/orders/errors.go:12
Issue: This helper should be in a shared package.
```

Reject because extraction is not a design finding unless ownership or source-of-truth behavior changed.

## Agent Traps
- Do not object to `internal/` packages by default; server-internal code is normal when not exported as a module API.
- Do not turn boundary review into a package-layout essay. Tie the comment to one crossed responsibility.
- Do not keep reviewing inside this file when the real issue is router semantics, data lifecycle, security, or reliability depth; make the boundary finding and hand off the specialist part.

## Validation Shape
Use fresh evidence from the diff plus the nearest approved owner artifact. Green tests do not prove boundary integrity; proof is that construction, policy lookup, and mapping still happen at the approved owner.
