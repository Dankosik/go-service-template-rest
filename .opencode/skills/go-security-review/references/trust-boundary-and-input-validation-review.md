# Trust Boundary And Input Validation Review

## Behavior Change Thesis
When loaded for symptom "external or boundary-crossing data is being accepted before action," this file makes the model choose boundary-first typed parsing, allowlists, limits, and pre-side-effect rejection instead of likely mistake "sanitize later" or trusting generated/internal callers.

## When To Load
Load this when changed Go code accepts or normalizes data from HTTP requests, generated API handlers, config files, environment variables, CLI flags, async messages, partner feeds, cache payloads, or database records crossing back into business logic.

Use this for the question "is this value trusted enough to act on?" If the primary danger is interpreter syntax, filesystem escape, or outbound target selection, load the narrower injection, path, or SSRF reference instead.

## Decision Rubric
- Name the boundary first: caller, config source, queue, cache, database re-entry, or partner feed.
- Prove attacker influence before writing a finding; do not flag every parse site.
- Require strict shape, enum, length, count, and semantic checks before writes, outbound calls, cache mutation, publish, or expensive work.
- Map caller selectors to code-owned constants; raw sort/filter/operator strings should not become internal policy.
- Treat unknown mutable fields, raw maps, and "validated by UI/generated type" as insufficient when they can change state or query shape.
- Preserve repo config hardening when touched: non-local config path policy, max file size, symlink rejection, allowed roots, permission checks, and secret-like key rejection.

## Imitate
```text
[high] [go-security-review] internal/api/orders.go:47
Issue: Axis: Trust Boundary And Input Validation; the handler decodes `sort` from the query and passes it into repository filtering before checking it against supported sort keys.
Impact: An authenticated caller can select unsupported operators that change the database predicate shape.
Suggested fix: Reject unknown sort keys at the HTTP boundary and map accepted values to internal enum constants before calling the repository.
Reference: request boundary and repository filter contract.
```

Copy this shape when the value is not SQL text yet, but boundary validation decides whether unsafe query behavior becomes possible.

```text
[medium] [go-security-review] internal/config/load_koanf.go:141
Issue: Axis: Trust Boundary And Input Validation; this change removes the config-file read cap for non-local config.
Impact: A compromised deployment config path can force startup memory pressure before secret-source policy runs.
Suggested fix: Keep the bounded read and add a regression test for an oversized config file.
Reference: `docs/configuration-source-policy.md`.
```

Copy this shape when the defect is a repo-specific config boundary, not generic "validate input" advice.

## Reject
```text
Issue: Validate this input better.
```

Reject because it does not name the boundary, attacker control, asset, or smallest correction.

```text
Suggested fix: Sanitize the request in the service layer.
```

Reject because security validation after business logic or side effects keeps the boundary ambiguous.

## Agent Traps
- Do not count OpenAPI/generated structs as complete validation when raw strings still encode policy.
- Do not let "internal queue" or "from cache" imply trusted data; name the producing trust contract or treat it as boundary-crossing.
- Do not duplicate the injection reference when the only issue is a missing SQL placeholder; load the injection reference for interpreter syntax.
- Do not recommend broad validation middleware when the safe fix is a resource-specific enum or typed request rule.

## Validation Shape
- Add table tests for accepted values, unknown values, boundary lengths, empty values, duplicate fields, and unsupported operators.
- Add negative HTTP tests that prove rejection occurs before side effects or repository calls.
- Add regression tests for oversized request bodies or config files when limits are touched.

## Repo-Local Anchors
- `internal/infra/http/middleware.go` uses request framing guards and `http.MaxBytesReader`.
- `internal/config/load_koanf.go` hardens non-local config file loading and rejects secret-like YAML keys.
- `docs/configuration-source-policy.md` separates non-secret YAML from secret environment variables.
