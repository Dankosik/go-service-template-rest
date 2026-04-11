---
name: go-security-spec
description: "Design security requirements for Go services: trust boundaries, identity and access rules, tenant isolation, threat-class controls, abuse resistance, secure defaults, and testable security behavior."
---

# Go Security Spec

## Purpose
Define security requirements before coding so trust boundaries, identity rules, authorization, tenant isolation, threat controls, data protection, privacy, abuse resistance, and negative-path proof are explicit and testable.

## Specialist Stance
- Treat security as explicit trust boundaries, identity rules, denial behavior, and abuse controls.
- Separate authentication, authorization, tenant isolation, sensitive-data handling, privacy, and resource-exhaustion defenses.
- Prefer fail-closed, least-privilege, standard-library-friendly controls with concrete negative-path proof.
- Hand off architecture, API, physical schema, reliability, observability, or delivery policy when they stop being security-owned decisions.

## Scope
- define trust boundaries, attacker paths, security assumptions, and threat exposure for affected flows
- define identity and access rules, including caller/subject separation, tenant binding, object-level authorization, and property-level authorization
- define secure-by-default controls for untrusted input, outbound access, secrets, sensitive data, privacy, and telemetry redaction
- define abuse-resistance behavior: limits, bounded concurrency, timeout policy, rate-limit semantics, and safe degradation
- define fail-closed behavior for critical security paths and degraded dependencies
- define verification obligations for negative and abuse-path security behavior
- surface hidden security decisions instead of leaving them to implementation guesses

## Boundaries
Do not:
- redesign general service architecture, ownership topology, or distributed coordination model as the primary output
- take ownership of full API resource modeling or physical schema design outside their security impact
- prescribe low-level middleware, handler, repository, or CI wiring as the main result
- treat observability, reliability, or delivery policy as the primary domain unless they materially affect security behavior

## Core Defaults
- Use a zero-trust baseline: external, partner, internal service, async worker, and third-party API traffic are untrusted unless explicitly justified otherwise.
- Keep authentication, authorization, tenant isolation, data protection, privacy, and abuse resistance as separate decision blocks.
- Prefer deny-by-default and least privilege. Missing policy means deny.
- Prefer standard library and vetted existing platform controls; security libraries require explicit justification.
- Missing trust-boundary facts, identity-model facts, data-classification facts, or enforcement ownership are blockers, not details to improvise later.

## Reference Files
Load only the files needed for the security requirement question. Use these references before writing security examples or coding-facing requirements.

- `references/trust-boundary-threat-modeling.md`: boundary maps, attacker paths, STRIDE-style prompts, threat responses, and repo-local boundary anchors.
- `references/authentication-authorization-tenant-isolation.md`: identity model, bearer/JWT expectations, caller/subject separation, object/function/property authorization, and tenant isolation.
- `references/input-output-injection-and-ssrf-controls.md`: request validation, JSON strictness, injection controls, SSRF and outbound allowlists, response sanitization, and Go parser/API caveats.
- `references/api-facing-security-semantics.md`: REST security semantics, `401` vs `403`, `405`, `413`, `415`, `429`, `503`, idempotency, CORS, security headers, and sanitized problem responses.
- `references/async-distributed-security.md`: async envelope authenticity, replay windows, token propagation rules, third-party API consumption, webhook callbacks, and distributed recovery security.
- `references/data-cache-and-secret-handling.md`: sensitive data minimization, privacy, cache isolation, DB access controls, secret-source policy, key management, and logging/redaction.
- `references/security-negative-path-verification.md`: negative-path and abuse-path test obligations, auth matrix testing, JWT tampering, BOLA, tenant-crossing, injection/SSRF, resource exhaustion, and CI security gates.

## Design Method
- Start from the affected flow and name the security decision that is still implicit.
- Load the relevant reference files and reuse their selected/rejected controls, fail-closed examples, and testable requirement patterns.
- Produce requirements, not implementation guesses: state the threat scenario, selected control, rejected alternative, enforcement point, failure behavior, and verification obligation.
- Tie security requirements to repo-local sources of truth when present, such as `api/openapi/service.yaml`, `internal/infra/http`, `internal/config`, `SECURITY.md`, and CI security targets.
- If a control depends on an identity provider, gateway, service mesh, secret manager, cache, queue, or third-party API not present in the repo, record the assumption or blocker.

## Decision Quality Bar
Major security recommendations should make the following explicit:
- trust boundary and attacker path
- selected control and at least one rejected alternative when nontrivial
- enforcement point and owner
- fail-closed behavior and degraded-dependency behavior
- sensitive-data and privacy impact
- negative-path and abuse-path verification
- residual risk and reopen conditions

Security claims without enforcement and verification are incomplete.

## Deliverable Shape
Return security work in a compact, reviewable form:
- `Security Decisions`
- `Threat-Control Matrix`
- `Identity, Authorization, And Tenant Rules`
- `Sensitive Data, Privacy, And Redaction Rules`
- `Abuse Resistance And Fail Behavior`
- `Verification Obligations`
- `Assumptions And Residual Risks`

## Escalate When
Escalate if:
- trust boundaries or identity model are ambiguous
- object-level authorization, property-level authorization, or tenant isolation lacks an explicit enforcement point
- untrusted input lacks threat-class control coverage
- retry-unsafe behavior has no idempotency contract
- async paths lack authenticity, replay, or dedup rules
- sensitive-data handling lacks minimization, sanitization, or redaction rules
- abuse-prone paths have no bounded timeout, limit, or concurrency strategy
- runtime hardening assumptions materially affect safety but remain undefined
