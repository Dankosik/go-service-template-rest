# Security Negative-Path Verification

## Behavior Change Thesis
When loaded for validation obligations, this file makes the model choose concrete negative-path, abuse-path, and no-side-effect proof requirements instead of likely mistake: "covered by integration tests", scanner-only confidence, or status-code-only assertions.

## When To Load
Load this when turning security requirements into validation obligations before coding, when a spec needs negative-path tests, when auth matrices are involved, or when abuse, tenant-crossing, JWT tampering, injection, SSRF, secret leakage, or resource exhaustion needs proof.

## Decision Rubric
- Every security decision should yield at least one positive proof and one negative proof. For high-risk access decisions, prefer a small matrix over a single example.
- Test authentication failures separately from authorization failures: missing credentials, malformed credentials, invalid signature, wrong issuer/audience, expired token, insufficient scope, wrong tenant, wrong object, and wrong property.
- Test BOLA with two accounts or tenants, multiple HTTP methods, object IDs in path/query/header/body, and bulk/list endpoints when those surfaces exist.
- Test no-side-effect behavior, not only status code: repository mutation, cache write, job enqueue, provider call, emitted secret, telemetry field, and audit/security event when relevant.
- Use scanners for known-vulnerability and secret-detection classes, not as proof of authorization, tenant isolation, business-flow abuse, or privacy rules.
- Tie code/config changes to repo gates when applicable: unit/integration tests, `make go-security`, `make secrets-scan`, OpenAPI checks, and targeted contract tests.

## Imitate
- "For each protected endpoint, missing auth returns `401`; valid auth with wrong tenant/object/property returns `403` or approved concealment; assertions prove no repository mutation, cache write, or job enqueue occurred." Copy the denial plus no-side-effect proof.
- "JWT tests include altered payload, `alg: none`, wrong audience, wrong issuer, expired `exp`, future `nbf`, unknown key ID, missing signature, and wrong token type." Copy the tamper matrix.
- "SSRF tests assert disallowed destinations fail before dial; allowed destinations still enforce timeout, response-size, media-type, and sanitized error behavior." Copy the pre-dial proof.

## Reject
- "Integration tests cover it." This says nothing about negative security cases.
- "Run `gosec`." Scanners do not prove tenant isolation, object authorization, property filtering, replay safety, or abuse semantics.
- "Assert status is `403`." Status without no-side-effect and no-leak checks can miss the real defect.
- "Manual test the matrix." Stable authorization matrices should be encoded when the rules are durable.

## Agent Traps
- Do not test only one HTTP method when another method can reach the same object or mutate state.
- Do not skip list and bulk endpoints; BOLA often hides in enumeration and collection filters.
- Do not assume a denial response proves no cache, telemetry, provider, or async side effect occurred.
- Do not make proof obligations broader than the spec. Name the smallest test layer that can catch the security failure.

## Validation Shape
- Auth matrix: unauthenticated -> malformed -> authenticated wrong role/scope -> wrong tenant -> wrong object -> wrong property -> allowed.
- Abuse matrix: body/batch/page/concurrency/retry/provider-cost dimensions -> limit -> expected denial -> no-side-effect assertion.
- Secret/privacy proof: config rejection, raw-secret redaction, no secrets in logs or problem responses, and no sensitive data in URL/query strings.
- Repo gates: `make go-security` for `govulncheck` and `gosec`; `make secrets-scan` for `gitleaks`; targeted Go/OpenAPI tests for behavior.

## Repo-Local Anchors
- `internal/infra/http/router_test.go` includes fail-closed CORS preflight, security header, request framing, request ID, and body-limit tests.
- `internal/config/config_test.go` includes secret policy and raw-secret redaction tests.
- `Makefile` provides `go-security`, `secrets-scan`, `openapi-check`, `test`, and `test-race` proof commands.
- `scripts/ci/required-guardrails-check.sh` tracks required guardrails including security policy and CI checks.
