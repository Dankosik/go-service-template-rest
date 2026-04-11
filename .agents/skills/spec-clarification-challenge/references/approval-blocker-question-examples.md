# Approval Blocker Question Examples

## When To Load
Load this when the challenge needs examples of questions that can make `spec.md` approval dishonest. Use it for hidden assumptions that could change scope, acceptance semantics, ownership, failure behavior, rollout, or validation.

Do not load it just to pad a challenge. A blocker question earns its place because a different answer changes approval.

## Strong Vs Weak Questions

### Backend idempotency hidden behind UI behavior
Strong:

> The candidate decision says the UI disables the export button, so backend idempotency is not needed. What happens when an HTTP retry, browser refresh, or second client creates the same export request, and would that change the approved API semantics or validation for `POST /v1/exports`?

Correct classification: `blocks_spec_approval`

Recommended next action: `answer_from_existing_evidence` if retry/idempotency policy already exists; otherwise `expert_subagent` for an API or reliability lane.

Weak:

> Should exports use idempotency?

Why weak: it asks for a design preference without tying the question to the candidate decision or approval impact.

### Staleness assumption in user-visible cache
Strong:

> The candidate decision accepts a 10-minute account-summary TTL without invalidation. Which user-visible fields may be stale for that long, and would support or tenant-facing correctness require a shorter TTL, explicit stale-data disclosure, or invalidation constraint before approval?

Correct classification: `blocks_spec_approval`

Recommended next action: `answer_from_existing_evidence` if stale-data tolerance is documented; otherwise `expert_subagent` for domain/data or `requires_user_decision` only if tolerance is a product/support policy outside repo evidence.

Weak:

> Is the cache TTL good?

Why weak: it invites tuning instead of asking whether the spec can be approved with the current stale-data semantics.

### Security intent contradicted by delayed revocation
Strong:

> The problem frame includes compromised accounts, but the candidate decision lets existing sessions expire naturally. Does deactivation need immediate session, token, or integration revocation to satisfy the stop-abuse intent, and would validation need to prove that before approval?

Correct classification: `blocks_spec_approval`

Recommended next action: `expert_subagent` for security/reliability when repo policy can decide; `requires_user_decision` only if support/product policy owns the compromise response.

Weak:

> What about sessions?

Why weak: it names a topic but not the hidden assumption or approval consequence.

### Tenant boundary hidden in artifact delivery
Strong:

> The candidate decision uses signed URLs for export downloads after a tenant lookup. What prevents a valid signed URL from being reused outside the tenant context before expiry, and would the answer change the approved download contract, storage keying, URL lifetime, or validation proof?

Correct classification: `blocks_specific_domain`

Recommended next action: `expert_subagent` for security or API contract review.

Weak:

> Are signed URLs secure?

Why weak: it is a generic security checklist question.

### Record-only example, not a blocker
Strong:

> The spec already decides seven-day export retention and requires cleanup proof, but does not choose the cleanup mechanism. Should `spec.md` record that cleanup implementation belongs to `design/` while preserving the seven-day retention invariant?

Correct classification: `non_blocking_but_record`

Recommended next action: `defer_to_design`

Weak:

> How will cleanup work?

Why weak: it asks for design authorship after the approval invariant is already present.

## Classification Guardrails
- Use `blocks_spec_approval` when the answer could change the approved problem, scope, acceptance semantics, cross-domain invariant, rollout, or validation proof.
- Use `blocks_specific_domain` when one bounded expert lane can answer the reopened seam without rewriting the whole spec.
- Use `non_blocking_but_record` when approval stays honest if the assumption, deferral, or risk is explicitly recorded in `spec.md`.

## Exa Source Links
Exa MCP lookup and fetch were attempted before authoring on 2026-04-11, but the tool returned a 402 credit-limit error. When Exa is available, refresh against these links. Repo authorities remain controlling for gate placement and reconciliation.

- Repo authority: `/Users/daniil/Projects/Opensource/go-service-template-rest/AGENTS.md`
- Repo authority: `/Users/daniil/Projects/Opensource/go-service-template-rest/docs/spec-first-workflow.md`
- NASA, requirement validation and assumption confirmation: https://www.nasa.gov/reference/appendix-c-how-to-write-a-good-requirement/
- NASA, validated requirements and stakeholder agreement before baselining: https://www.nasa.gov/reference/4-2-technical-requirements-definition/
- Requirements-smell research for ambiguity/testability: https://arxiv.org/abs/2403.17479
