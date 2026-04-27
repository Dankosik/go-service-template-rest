---
name: go-security-spec
description: "Design security requirements for Go services: trust boundaries, identity and access rules, tenant isolation, threat-class controls, abuse resistance, secure defaults, and testable security behavior."
---

# Go Security Spec

## Purpose
Define security requirements before coding so trust boundaries, identity rules, authorization, tenant isolation, threat controls, data protection, privacy, abuse resistance, and negative-path proof are explicit and testable.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist. Follow exact order only when this skill or the repository contract makes the sequence an invariant.
- Use the minimum context, references, tools, and validation loops that can change the deliverable; stop expanding when the quality bar is met.
- Before acting, resolve prerequisite discovery, lookup, or artifact reads that the outcome depends on; parallelize only independent evidence gathering and synthesize before the next decision.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change correctness, ownership, safety, or scope.
- When evidence is missing or conflicting, retry once with a targeted strategy or label the assumption, blocker, or reopen target instead of treating absence as proof.
- Finish only when the requested deliverable is complete in the required shape and verification or a clearly named blocker/residual risk is recorded.

## Specialist Stance
- Treat security as explicit trust boundaries, identity rules, denial behavior, and abuse controls.
- Separate authentication, authorization, tenant isolation, sensitive-data handling, privacy, and resource-exhaustion defenses.
- Prefer fail-closed, least-privilege, standard-library-friendly controls with concrete negative-path proof.
- Hand off architecture, API, physical schema, reliability, observability, or delivery policy when they stop being security-owned decisions.
- If another domain is only affected, return the consequence as `constraint_only`, `proof_only`, or explicit `no new decision required` instead of widening the design.

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
References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Load at most one reference by default; load multiple only when the task clearly spans independent security decision pressures, such as tenant authorization plus SSRF, async replay plus secret handling, or REST status semantics plus abuse budgeting.

Before loading a reference, name the symptom and the behavior change you need. If the skill body already steers the decision, skip the reference.

| Symptom | Load | Behavior Change |
| --- | --- | --- |
| Trust boundaries, attacker paths, or boundary ownership are implicit. | `references/trust-boundary-threat-modeling.md` | Choose named boundary, attacker-path, enforcement, and proof requirements instead of generic "use auth" or "validate input" advice. |
| Identity, authorization, tenant, object, property, or admin rules are in scope. | `references/authentication-authorization-tenant-isolation.md` | Choose caller/subject/tenant-bound access rules instead of role-only checks, untrusted headers, or `subject_id == path_id` shortcuts. |
| JSON/input parsing, SQL or interpreter use, outbound URLs, webhooks, or sanitized errors are in scope. | `references/input-output-injection-and-ssrf-controls.md` | Choose strict parser, allowlist, SSRF dial, and sanitized output requirements instead of denylist validation, late validation, or raw error relay. |
| REST/OpenAPI status codes, CORS, problem responses, method policy, request limits, retry/idempotency, or browser-visible headers are in scope. | `references/api-facing-security-semantics.md` | Choose contract-visible fail-closed semantics instead of ad hoc status codes, `200` error bodies, permissive CORS, or retry ambiguity. |
| Queues, workers, outbox/inbox, webhooks, callbacks, cross-service calls, third-party APIs, or retries are in scope. | `references/async-distributed-security.md` | Choose authenticity, replay, scoped credential, and step-authorization rules instead of trusting internal queues or propagating raw bearer tokens. |
| Sensitive data, privacy, cache keys, DB access, secrets, config source policy, logging, redaction, or telemetry fields are in scope. | `references/data-cache-and-secret-handling.md` | Choose classification, minimization, cache scoping, secret-source, and redaction requirements instead of shared caches, secret config, or log leakage. |
| Rate limits, expensive operations, bulk work, provider-cost triggers, repeated attempts, or resource exhaustion are in scope. | `references/resource-abuse-and-cost-controls.md` | Choose principal/tenant-scoped budgets and cheap pre-side-effect gates instead of one global rate limit or reliability-only overload wording. |
| Security decisions need proof obligations, test matrices, abuse-path checks, or scanner-vs-test separation. | `references/security-negative-path-verification.md` | Choose concrete negative-path and no-side-effect proof obligations instead of "covered by integration tests" or scanner-only confidence. |

## Design Method
- Start from the affected flow and name the security decision that is still implicit.
- Load at most one relevant reference by default and reuse its rubric, examples, traps, and proof shape. Load another only for an independent pressure the first file does not cover.
- Produce requirements, not implementation guesses: state the threat scenario, selected control, rejected alternative, enforcement point, failure behavior, and verification obligation.
- Tie security requirements to repo-local sources of truth when present, such as `api/openapi/service.yaml`, `internal/infra/http`, `internal/config`, `SECURITY.md`, and CI security targets.
- If a control depends on an identity provider, gateway, service mesh, secret manager, cache, queue, or third-party API not present in the repo, record the assumption or blocker.

## Decision Quality Bar
Major security recommendations should make the following explicit:
- trust boundary and attacker path
- whether a real `live fork` exists
- when a `live fork` exists, the selected control and at least one rejected alternative
- enforcement point and owner
- fail-closed behavior and degraded-dependency behavior
- sensitive-data and privacy impact
- negative-path and abuse-path verification
- downstream decision/proof consequences only when another domain must act before the current artifact is usable
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
- `Downstream Decision Or Proof Consequences`
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
