# Secure coding instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Implementing or reviewing HTTP handlers, request decoding, validation, authz, DB access, filesystem access, or outbound clients
  - Working on file uploads, command execution, template rendering, serialization/deserialization logic
  - Changing timeout/limit policies, retry behavior, concurrency limits, or other abuse-resistance controls
  - Doing security review, threat-driven refactor, or hardening work
- Do not load when: The task is documentation-only or pure internal refactor with no trust-boundary/runtime behavior impact

## Purpose
- This document defines secure-by-default coding rules for Go services.
- Goal: prevent common vulnerability classes and reduce security regressions from generated code.
- This is an operational standard: defaults are mandatory unless an explicit exception is approved in review.

## Baseline assumptions
- Every external input is untrusted: path, query, headers, cookies, body, uploaded files, webhook payloads, and downstream service data.
- Internal network traffic is not trusted by default.
- The default service profile is JSON-over-HTTP API with SQL datastore and outbound HTTP integrations.
- Security controls must be enforced both at contract level and runtime level.
- Prefer standard library first; introducing third-party security libraries requires explicit justification.

## Required inputs before changing security-sensitive behavior
Resolve these first. If unknown, apply defaults and document assumptions.

- Trust boundary: external API, partner API, internal service, background worker.
- Data sensitivity: public, internal, confidential, regulated.
- Threat exposure: internet-facing, private network only, or mixed.
- Side effects and retry model: idempotent/non-idempotent, dedup strategy.
- Resource budget: max request size, max upload size, timeout budget, concurrency budget.
- Outbound access policy: allowed schemes/hosts/ports, redirect policy, egress restrictions.

## Security defaults by threat class

### Input validation
- Validate at the boundary before business logic.
- Use strict decoding for JSON request bodies:
  - apply `http.MaxBytesReader` before decode,
  - decode with `json.Decoder`,
  - call `DisallowUnknownFields()`,
  - reject trailing JSON tokens.
- Validate with explicit allowlists:
  - allowed enums, ranges, lengths, formats, and field-level mutability,
  - allowed sort keys and filter keys,
  - allowed state transitions.
- Reject black-list-only validation as primary control.
- Treat missing validation as a merge-blocking issue.

### Output encoding
- JSON responses:
  - set `Content-Type: application/json; charset=utf-8`,
  - use `json.NewEncoder(w).Encode(...)` (no manual string concatenation).
- HTML responses:
  - use `html/template` only,
  - never use `text/template` for HTML output.
- Never return raw internal errors to clients; return stable sanitized error payloads.
- Do not reflect untrusted input into headers without strict validation (CRLF-safe values only).

### Injection classes
- SQL injection:
  - always use parameterized queries,
  - never build SQL values with `fmt.Sprintf` or string concatenation,
  - dynamic identifiers (`ORDER BY`, column names) must use explicit allowlists.
- NoSQL/operator injection:
  - do not pass raw client JSON as datastore filters,
  - map request fields to typed DTOs and approved operators only.
- Command injection:
  - default policy is no OS command execution in request path,
  - if unavoidable, never invoke shell (`sh -c`, `bash -c`, `cmd /c`, `powershell -Command`) with user input.
- Template/script injection:
  - no direct interpolation into HTML/JS contexts outside safe template engine behavior.

### SSRF
- Any outbound URL influenced by untrusted input must pass SSRF policy:
  - allowlist schemes (`https` by default),
  - allowlist hosts/domains/ports,
  - block loopback, link-local, multicast, and private IP ranges after DNS resolution,
  - re-check every redirect target,
  - disable or tightly limit redirects by default.
- Do not use `http.DefaultClient` or `http.Get` for security-sensitive outbound traffic.
- Outbound HTTP clients must have explicit timeout budget and transport settings.
- Network-layer egress controls are mandatory defense-in-depth; code-only SSRF controls are insufficient.

### Path traversal
- For attacker-controlled file paths, use `os.OpenInRoot` / `os.Root` (Go 1.24+) as default.
- Never assume `filepath.Join(base, userPath)` is sufficient protection.
- Never use raw user-provided filenames as storage paths.
- Store uploaded/generated files outside public webroot by default.
- Enforce canonical path policy and ownership boundary per storage root.

### Deserialization
- Accept only explicitly supported formats by endpoint contract.
- Decode into typed structs; avoid `map[string]any` for security-critical decisions.
- Reject unknown fields by default.
- Bound input size before deserialization.
- Prefer deterministic parsers over permissive formats for security-sensitive flows.
- Do not deserialize untrusted data into executable/configurable structures without strict schema constraints.

### Resource exhaustion
- Define explicit limits for every entrypoint:
  - max headers (`MaxHeaderBytes`),
  - max URI length (gateway policy),
  - max JSON body (default `1 MiB`),
  - max multipart body (default `10 MiB` unless contract says otherwise),
  - max page size / filter complexity.
- Apply strict time budgets:
  - inbound request deadlines,
  - outbound HTTP timeouts,
  - DB query timeouts.
- Avoid unbounded memory patterns (`io.ReadAll` on untrusted streams).
- For fan-out or bulk work, enforce bounded concurrency (`semaphore`/worker pool).
- Protect expensive endpoints with rate limiting/quota controls and clear `429` behavior.

## Safe HTTP defaults
- Always construct `http.Server` explicitly; never rely on implicit zero-timeout defaults.
- Default server settings for JSON API profile:
  - `ReadHeaderTimeout: 2s`
  - `ReadTimeout: 5s`
  - `WriteTimeout: 10s`
  - `IdleTimeout: 60s`
  - `MaxHeaderBytes: 16 << 10`
- Enforce body limits before decoding via `http.MaxBytesReader`.
- Reject suspicious request framing:
  - if both `Transfer-Encoding` and `Content-Length` are present, return `400` and close connection.
- Set minimal response hardening headers for API responses:
  - `X-Content-Type-Options: nosniff`
  - avoid leaking unnecessary server fingerprinting headers.
- Return generic error messages to clients; log full details with request correlation IDs.

## Template escaping rules
- HTML generation must use `html/template`.
- `template.HTML`, `template.JS`, `template.URL`, `template.CSS` are dangerous-by-default:
  - allow only for trusted pre-sanitized content,
  - require explicit security review and documented sanitization path.
- Never bypass escaping to "fix rendering" without review.

## File upload handling rules
- Apply `MaxBytesReader` before multipart parsing.
- Prefer streaming via `MultipartReader`; avoid buffering full files in memory.
- Validate file type with both:
  - extension allowlist,
  - content sniffing (magic bytes).
- Generate storage filename on server side (UUID/random); do not trust client filename.
- Store outside webroot with restrictive permissions.
- If malware/content scanning is required, publish only after scan result.
- For large/bursty uploads, use direct object-storage upload flow (presigned URL + finalize).

## Command execution policy
- Default: forbidden in request handlers and business logic.
- If command execution is unavoidable:
  - isolate in dedicated adapter package,
  - execute fixed binary from allowlist,
  - pass arguments as explicit tokens (no shell),
  - validate each argument against allowlist/range/pattern,
  - use `exec.CommandContext` with strict timeout,
  - run with least privilege and controlled environment,
  - cap stdout/stderr collection size,
  - log command intent, not secrets.
- Any new command execution path requires explicit security review approval.

## `unsafe` usage policy
- Default: prohibited.
- Allowed only if all conditions are met:
  - measured performance need cannot be solved safely,
  - isolated package boundary with clear ownership,
  - design note explains memory-safety invariants,
  - targeted tests include race and edge cases,
  - reviewers with Go/runtime expertise approve.
- Never introduce `unsafe` for readability convenience or speculative micro-optimizations.

## Dangerous APIs and patterns requiring explicit review

| API / pattern | Risk | Required control |
|---|---|---|
| `http.Get`, `http.DefaultClient` | no enforced timeout, weak SSRF posture | dedicated client with timeout + SSRF policy |
| `http.ListenAndServe` default usage | missing server hardening defaults | explicit `http.Server` config |
| `json.Unmarshal` over unbounded body | memory exhaustion, weak input discipline | `MaxBytesReader` + strict `Decoder` |
| `Request.ParseMultipartForm` on large input | memory/resource pressure | stream with limits, avoid full buffering |
| `io.ReadAll` on untrusted stream | unbounded memory growth | streaming or bounded reader |
| `fmt.Sprintf` / string concatenation for SQL | SQL injection | parameterized SQL + identifier allowlist |
| raw NoSQL filter from client JSON | operator injection | typed DTO + operator allowlist |
| `filepath.Join(base, userPath)` + `os.Open` | path traversal, symlink escape | `os.OpenInRoot` / root-constrained FS |
| `exec.Command("sh", "-c", userInput)` | command injection | no shell, tokenized args, allowlists |
| `text/template` for HTML | XSS risk | `html/template` |
| `template.HTML` / `template.JS` etc. | escape bypass | trusted sanitized data + explicit review |
| `tls.Config{InsecureSkipVerify: true}` | MITM risk | full TLS validation or approved test-only exception |
| `context.Background()` in request flow | timeout/cancel bypass, stuck work | propagate request context and deadlines |
| `unsafe` package | memory safety violations | exceptional case process + expert review |

## Decision rules
Apply in order.

1. If request data is untrusted and parsed, enforce size limit before parsing.
2. If user input affects query, command, path, or URL target, use allowlist strategy first.
3. If operation is outbound network I/O, apply explicit timeout and SSRF policy.
4. If operation touches filesystem with user influence, constrain to root (`OpenInRoot`) and server-generated names.
5. If operation can amplify CPU/memory/connections, add bounded concurrency and request limits.
6. If a change needs an exception from defaults, document rationale and require explicit security review.

## Anti-patterns to reject
- Silent parsing with ignored decode errors.
- "Best effort" validation after side effects are started.
- Returning internal stack traces or dependency error strings to clients.
- Fire-and-forget goroutines in request path for critical work.
- Hidden retries of non-idempotent operations.
- Shipping new privileged command execution paths without review.
- Introducing `unsafe` without hard evidence and ownership.

## Security review criteria (merge gate)
- Trust boundary and threat assumptions are documented in PR description.
- All new inputs have explicit validation and size limits.
- Serialization/deserialization behavior is strict and bounded.
- Outbound requests have timeout, SSRF controls, and redirect policy.
- DB access is parameterized; dynamic query fragments are allowlisted.
- Filesystem operations are traversal-safe and avoid client-controlled paths.
- Upload flow has size/type/path controls and storage isolation.
- Error responses are sanitized; logs keep diagnostic detail with correlation IDs.
- Command execution and `unsafe` usage are absent or explicitly approved.
- Abuse controls exist for expensive paths: timeout + limit + concurrency + retry policy.
- Verification commands were run: `go test ./...`, `go test -race ./...` (for concurrency-sensitive changes), `go vet ./...`, `govulncheck ./...`.

## Good / bad examples

### 1) Secure-by-default HTTP handler

Bad:
```go
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	var req CreateUserRequest
	_ = json.Unmarshal(body, &req) // unknown fields ignored by default

	_ = h.svc.Create(context.Background(), req) // drops request cancellation
	w.Write([]byte("ok"))
}
```

Good:
```go
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	const maxBody = 1 << 20 // 1 MiB
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req CreateUserRequest
	if err := dec.Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if dec.More() {
		writeProblem(w, http.StatusBadRequest, "invalid_json", "trailing json data")
		return
	}
	if err := validateCreateUser(req); err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "validation_failed", "invalid input")
		return
	}

	if err := h.svc.Create(r.Context(), req); err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

### 2) Secure outbound HTTP client (SSRF-aware)

Bad:
```go
func FetchAvatar(ctx context.Context, rawURL string) ([]byte, error) {
	resp, err := http.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
```

Good:
```go
var outboundClient = &http.Client{
	Timeout: 5 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func FetchAvatar(ctx context.Context, rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if !isAllowedOutboundURL(u) {
		return nil, errors.New("url is not allowed")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := outboundClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Bound response size to prevent memory abuse.
	return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}
```

### 3) Secure DB access (SQL + NoSQL)

Bad:
```go
func FindUser(ctx context.Context, db *sql.DB, email string, order string) (*User, error) {
	q := fmt.Sprintf("SELECT id,email FROM users WHERE email='%s' ORDER BY %s", email, order)
	row := db.QueryRowContext(ctx, q)
	var u User
	if err := row.Scan(&u.ID, &u.Email); err != nil {
		return nil, err
	}
	return &u, nil
}
```

Good:
```go
func FindUser(ctx context.Context, db *sql.DB, email string, order string) (*User, error) {
	orderCol := "id"
	switch order {
	case "", "id":
		orderCol = "id"
	case "email":
		orderCol = "email"
	default:
		return nil, errors.New("invalid order")
	}

	q := "SELECT id, email FROM users WHERE email = $1 ORDER BY " + orderCol
	row := db.QueryRowContext(ctx, q, email)

	var u User
	if err := row.Scan(&u.ID, &u.Email); err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	return &u, nil
}

type UserSearchRequest struct {
	Email string `json:"email"`
}
// For NoSQL paths: decode strict DTO and map fields to fixed operators;
// never pass raw client JSON filter directly into driver query.
```

### 4) Secure filesystem interaction

Bad:
```go
func ReadUserFile(baseDir, userPath string) ([]byte, error) {
	fullPath := filepath.Join(baseDir, userPath)
	return os.ReadFile(fullPath)
}
```

Good:
```go
func ReadUserFile(baseDir, userPath string) ([]byte, error) {
	f, err := os.OpenInRoot(baseDir, userPath)
	if err != nil {
		return nil, fmt.Errorf("open in root: %w", err)
	}
	defer f.Close()

	// Bound read to protect memory in case of unexpected file size.
	return io.ReadAll(io.LimitReader(f, 1<<20))
}
```

## MUST / SHOULD / NEVER

### MUST
- MUST treat all boundary input as untrusted until validated.
- MUST enforce size/time/concurrency limits before expensive work.
- MUST use strict JSON decoding defaults and explicit validation.
- MUST parameterize datastore queries and allowlist dynamic fragments.
- MUST apply SSRF policy to untrusted outbound targets.
- MUST use traversal-safe filesystem APIs for user-influenced paths.
- MUST sanitize client-facing errors and preserve details only in logs.
- MUST require explicit review for command execution and `unsafe`.

### SHOULD
- SHOULD keep HTTP defaults explicit (`http.Server` with timeouts and header limits).
- SHOULD centralize validation and SSRF policy helpers to avoid per-handler drift.
- SHOULD use `html/template` and safe-by-default rendering primitives.
- SHOULD run security tooling (`govulncheck`, `go vet`, race tests where relevant) in CI.

### NEVER
- NEVER trust unknown JSON fields by default in mutable endpoints.
- NEVER use shell execution with user-controlled input.
- NEVER build SQL/NoSQL queries directly from raw client strings/maps.
- NEVER use client-controlled filenames/paths as storage paths.
- NEVER enable `InsecureSkipVerify` in production.
- NEVER add `unsafe` without measured need and explicit review.
