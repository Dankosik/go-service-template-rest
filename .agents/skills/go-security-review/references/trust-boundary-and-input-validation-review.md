# Trust Boundary And Input Validation Review

## When To Load
Load this when changed Go code accepts or normalizes data from HTTP requests, config files, environment variables, generated API handlers, async messages, partner feeds, CLI flags, cache payloads, or database records crossing back into business logic. Use it when the question is "is this data trusted enough to act on?"

## Attacker Preconditions
- The attacker can influence at least one inbound field, header, route parameter, query parameter, body field, config file, env value, message payload, or external feed.
- The changed code parses or stores that value before a strict server-side boundary check.
- The value can affect a lookup, state transition, side effect, security decision, resource use, or downstream interpreter.

## Review Signals
- Validation happens after writes, outbound calls, cache mutation, publish, or expensive work.
- Validation is denylist-only, partial, client-side only, or relies on a UI dropdown without a server-side allowlist.
- Code accepts unknown mutable fields or passes raw maps into domain logic.
- Body, header, multipart, array, or filter complexity limits are missing.
- Repository config hardening is weakened: non-local config path policy, max config file size, symlink rejection, allowed roots, or secret-like key rejection.

## Bad Finding Examples
- "Validate this input better."
- "This handler should sanitize the request."
- "Consider rejecting bad values here."

These are weak because they do not name the trust boundary, attacker control, affected asset, or smallest correction.

## Good Finding Examples
- "[high] [go-security-review] internal/api/orders.go:47 Axis: Trust Boundary And Input Validation; the handler decodes `sort` from the query and passes it into repository filtering before checking it against the supported sort keys. An authenticated caller can select unsupported operators that change the database predicate shape. Reject unknown sort keys at the HTTP boundary and map accepted values to internal enum constants before calling the repository."
- "[medium] [go-security-review] internal/config/load_koanf.go:141 Axis: Trust Boundary And Input Validation; this change removes the 1 MiB config-file read cap for non-local config. A compromised deployment config path can force startup memory pressure before secret-source policy runs. Keep the `io.LimitReader` cap and add a regression test for an oversized config file."

## Smallest Safe Correction
- Parse into a typed request/config struct at the boundary.
- Enforce syntactic checks such as type, length, enum, and shape before semantic checks.
- Enforce semantic checks such as tenant, state, and allowed transition before side effects.
- Map user-controlled selectors to code-owned constants; do not pass raw strings as internal policy.
- Bound body size with `http.MaxBytesReader` or equivalent middleware before reading, and keep multipart and array limits explicit.
- For repo config, preserve absolute-path policy outside local env, allowed-root checks, symlink rejection, group/world-writable rejection, max file size, and secret-like key rejection.

## Validation Ideas
- Add table tests for accepted values, unknown values, boundary lengths, empty values, duplicate fields, and unsupported operators.
- Add negative HTTP tests that verify validation fails before side effects or repository calls.
- Add regression tests for request bodies over the configured limit and config files over the max size.
- Run `make test` for local behavior and `make go-security` when the change touches a security boundary.

## Repo-Local Anchors
- `internal/infra/http/middleware.go` uses `RequestFramingGuard` and `RequestBodyLimit` with `http.MaxBytesReader`.
- `internal/config/load_koanf.go` enforces config file size, allowed roots, symlink policy, and secret-like key rejection.
- `docs/configuration-source-policy.md` documents non-secret YAML, secret ENV, and hardened non-local config files.

## Exa Source Links
- OWASP Input Validation Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html
- OWASP Secure Code Review Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Secure_Code_Review_Cheat_Sheet.html
- Go `net/http` package docs: https://pkg.go.dev/net/http
- Go `path/filepath` package docs: https://pkg.go.dev/path/filepath
