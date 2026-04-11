# Defer To Design Vs Block Spec

## When To Load
Load this when a question may be valid but might belong in `design/` rather than blocking `spec.md` approval.

The key distinction: `spec.md` must settle the decision, invariant, acceptance meaning, and proof obligation. `design/` can choose the mechanism when those are already stable.

## Decision Rule
Block spec approval when the answer could change:

- user-visible or API behavior
- domain acceptance semantics
- source-of-truth ownership
- data isolation, retention, or deletion policy
- failure, retry, or degradation behavior
- rollout or safety constraint
- validation proof expected from implementation

Defer to design when the spec already states the invariant and proof obligation, and only the mechanism remains.

## Strong Vs Weak Questions

### Cleanup mechanism after retention is already approved
Strong:

> The candidate spec already fixes seven-day export retention and requires cleanup proof. Is the exact cleanup mechanism a `design/` concern, while `spec.md` only needs to record the retention invariant and validation obligation?

Correct classification: `non_blocking_but_record`

Recommended next action: `defer_to_design`

Weak:

> Should we implement cleanup with a cron job?

Why weak: it writes design instead of clarifying the spec boundary.

### Retention period itself is not approved
Strong:

> The candidate spec chooses seven-day artifact retention for every tenant without evidence of product, contract, or privacy acceptance. Would a different retention period change scope, data policy, or validation enough to block approval?

Correct classification: `blocks_spec_approval`

Recommended next action: `answer_from_existing_evidence`, `targeted_research`, or `requires_user_decision` if the answer is external policy.

Weak:

> How should storage lifecycle be configured?

Why weak: it skips the policy decision and jumps to implementation.

### Cache implementation detail after tenant invariant is settled
Strong:

> If `spec.md` already requires tenant-scoped cache keys and stale-data tolerance, should hash format, Redis TTL command choice, and serialization shape be deferred to `design/`?

Correct classification: `non_blocking_but_record`

Recommended next action: `defer_to_design`

Weak:

> What should the Redis key look like exactly?

Why weak: it invites design authorship when the approval invariant is already settled.

### Cache tenant invariant not settled
Strong:

> The current cache key omits tenant identity because `account_id` is assumed globally unique. Does approval require source-of-truth evidence for that assumption before deciding whether tenant keying is mandatory?

Correct classification: `blocks_specific_domain`

Recommended next action: `expert_subagent` for data/cache, or `answer_from_existing_evidence` if the orchestrator has schema evidence.

Weak:

> Can design decide the cache key later?

Why weak: it may hide an isolation decision that belongs in `spec.md`.

### Audit sink detail versus audit requirement
Strong:

> If the spec already requires audit for admin deactivation, can the exact audit event schema and sink be deferred to `design/`, while the approval gate only records actor, target, decision, and proof expectations at the spec level?

Correct classification: `non_blocking_but_record`

Recommended next action: `defer_to_design`

Weak:

> What table should audit logs use?

Why weak: it asks for implementation planning.

## Exa Source Links
Exa MCP lookup and fetch were attempted before authoring on 2026-04-11, but the tool returned a 402 credit-limit error. When Exa is available, refresh against these links. Repo authorities remain controlling for gate placement and reconciliation.

- Repo authority: `/Users/daniil/Projects/Opensource/go-service-template-rest/AGENTS.md`
- Repo authority: `/Users/daniil/Projects/Opensource/go-service-template-rest/docs/spec-first-workflow.md`
- NASA, requirements should state what is needed rather than how to provide it: https://www.nasa.gov/reference/appendix-c-how-to-write-a-good-requirement/
- arc42, architecture decisions include rationale, consequences, and stakeholder retraceability: https://docs.arc42.org/section-9/
- ADR background and decision rationale: https://adr.github.io/
