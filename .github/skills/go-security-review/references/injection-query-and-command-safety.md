# Injection, Query, And Command Safety Review

## Behavior Change Thesis
When loaded for symptom "attacker-influenced data reaches an interpreter," this file makes the model trace value versus syntax boundaries and choose bind/allowlist/no-shell fixes instead of likely mistake giving generic injection warnings or recommending escaping without context.

## When To Load
Load this when changed Go code builds SQL, search filters, datastore operators, shell commands, subprocess invocations, templates, HTML responses, URL/query strings for interpreters, or command-like mini-languages from caller-influenced data.

If the issue is only that a selector should have been rejected before reaching internal logic, load the trust-boundary reference. If it has reached SQL, shell, template, expression, or another interpreter, load this file.

## Decision Rubric
- Identify the interpreter and the attacker-controlled segment before writing a finding.
- Use placeholders for SQL values and code-owned allowlists for dynamic identifiers, operators, sort directions, table names, and filter names.
- Treat `fmt.Sprintf`, string concatenation, raw filters, and user-controlled datastore operators as unsafe until a fixed grammar proves otherwise.
- Prefer Go APIs over subprocesses. If a subprocess is required, hardcode the executable, pass operands as separate args, use `exec.CommandContext`, and bound the context.
- Reject `sh -c`, `cmd /c`, user-controlled executable names, and user-controlled option names unless an approved design says the command surface is intentional.
- Use `html/template` for HTML and avoid typed safe wrappers such as `template.HTML` unless trusted code or an approved sanitizer produced the value.

## Imitate
```text
[critical] [go-security-review] internal/infra/postgres/search.go:39
Issue: Axis: Injection And Query Safety; `fmt.Sprintf` inserts the request `filter` directly into the SQL `WHERE` clause.
Impact: An authenticated caller can change the predicate and read rows outside the intended search.
Suggested fix: Replace the raw filter string with typed filter fields, use driver placeholders for values, and map dynamic operators from a fixed allowlist.
Reference: SQL query construction boundary.
```

Copy this shape when user-controlled text becomes interpreter syntax.

```text
[high] [go-security-review] internal/app/export.go:74
Issue: Axis: Injection And Command Safety; the export command runs `sh -c` with a filename derived from the request.
Impact: A caller who controls the filename can influence shell parsing under the service account.
Suggested fix: Avoid the shell, hardcode the executable, pass validated operands as separate `exec.CommandContext` args, and use the request context.
Reference: subprocess execution boundary.
```

Copy this shape when the problem is shell parsing, not just "exec with user input."

```text
[medium] [go-security-review] internal/web/render.go:31
Issue: Axis: Template Injection/XSS; the change wraps user-authored profile HTML in `template.HTML`, bypassing contextual escaping.
Impact: A profile editor can inject active markup into every viewer's page.
Suggested fix: Store the value as plain text or run it through an approved sanitizer before marking it trusted.
Reference: HTML rendering boundary.
```

Copy this shape when a safe template API is bypassed.

## Reject
```text
Issue: This might be SQL injection.
```

Reject because it lacks the attacker-controlled value, interpreter context, and concrete correction.

```text
Suggested fix: Escape the string before concatenating it into the command.
```

Reject because shell and query escaping is context-sensitive and usually the wrong local control in Go review.

## Agent Traps
- Do not suggest parameterizing SQL identifiers; values can use placeholders, identifiers need allowlisted mapping.
- Do not treat `exec.Command(name, args...)` as safe when `name` or option names are caller-controlled.
- Do not mistake `cmd.String()` for safe shell quoting or use it as command input.
- Do not flag `text/template` by name alone; flag it when the output context is HTML, JS, shell, SQL, or another interpreter.
- Do not overlook telemetry leaks of raw SQL, command output, or interpreter diagnostics when they include sensitive values.

## Validation Shape
- Add tests that special characters remain data, not syntax, for SQL values and command arguments.
- Add tests for unsupported dynamic identifiers, operators, and sort keys.
- Add context-timeout tests for subprocess paths.
- Add rendering tests that assert user text is escaped and typed safe wrappers are not used on untrusted data.

## Repo-Local Anchors
- This repo uses sqlc-generated query files under `internal/infra/postgres/sqlcgen`; handwritten query changes deserve extra scrutiny.
- `Makefile` exposes `make go-security` for `govulncheck` and `gosec` and `make sqlc-check` for generated query drift.
