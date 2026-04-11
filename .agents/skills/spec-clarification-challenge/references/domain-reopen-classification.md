# Domain Reopen Classification

## When To Load
Load this when a question seems important but you need to decide whether it blocks the whole spec, reopens one expert domain, or should only be recorded.

Use this reference to keep the clarification pass advisory and routed. Do not run the expert analysis yourself inside this skill.

## Classification Rules
- `blocks_spec_approval`: the unresolved point could change the approved problem, scope, acceptance semantics, cross-domain invariant, rollout, or validation proof.
- `blocks_specific_domain`: the spec might still be near-approvable, but one bounded domain lane must answer a specific seam first.
- `non_blocking_but_record`: the spec can be approved if it records the assumption, deferral, or accepted risk.

## Strong Vs Weak Domain Reopen Questions

### API contract lane
Strong:

> The export create endpoint returns `202 Accepted`, but duplicate request behavior is justified only by UI disablement. Should an API lane reopen idempotency/retry semantics before approval because clients and network retries can bypass the UI?

Correct classification: `blocks_specific_domain`

Recommended next action: `expert_subagent` with an API-contract skill, or `answer_from_existing_evidence` if the repo has a settled retry policy.

Weak:

> Should the API be RESTful?

Why weak: it is generic and does not target an approval-changing candidate decision.

### Data/cache lane
Strong:

> The cache key omits tenant identity based on an assumption that `account_id` is globally unique. Should a data lane confirm identifier ownership and tenant isolation before approving the cache key invariant?

Correct classification: `blocks_specific_domain`

Recommended next action: `expert_subagent` with a data/cache skill.

Weak:

> Is the Redis key good?

Why weak: it asks for taste and skips the source-of-truth issue.

### Security lane
Strong:

> Admin deactivation is internal-only, but audit logging is deferred. Should a security lane determine whether admin mutation audit is a mandatory trust-boundary requirement before `spec.md` can approve a one-click destructive action?

Correct classification: `blocks_specific_domain`

Recommended next action: `expert_subagent` with a security skill.

Weak:

> Are there security concerns?

Why weak: it is a broad checklist prompt.

### Reliability lane
Strong:

> Redis outage fallback goes directly to Postgres. Should a reliability lane reopen overload behavior before approval because the fallback path could undo the stated DB-load objective during a cache outage?

Correct classification: `blocks_specific_domain`

Recommended next action: `expert_subagent` with a reliability skill.

Weak:

> What happens if Redis is down?

Why weak: it does not say whether the answer changes the spec or only the design.

### Domain/policy lane versus user decision
Strong:

> The spec lets support decide paid-customer deactivation per case without recorded product policy. Can repo evidence define that policy, or is this an external business decision that should block or partially draft approval?

Correct classification: `blocks_spec_approval`

Recommended next action: `requires_user_decision` only if repo evidence cannot answer; otherwise `answer_from_existing_evidence`.

Weak:

> What does support want?

Why weak: it asks the human by default instead of checking whether the orchestrator can reconcile from evidence.

## Reopen/Rerun Guidance
Recommend rerunning this clarification challenge once only when the domain answer materially changes candidate decisions or reopens a major seam. If the domain answer only adds a constraint already compatible with the spec, record it and continue.

## Exa Source Links
Exa MCP lookup and fetch were attempted before authoring on 2026-04-11, but the tool returned a 402 credit-limit error. When Exa is available, refresh against these links. Repo authorities remain controlling for gate placement and reconciliation.

- Repo authority: `/Users/daniil/Projects/Opensource/go-service-template-rest/AGENTS.md`
- Repo authority: `/Users/daniil/Projects/Opensource/go-service-template-rest/docs/spec-first-workflow.md`
- NASA, requirements management and traceability across design/test plans: https://www.nasa.gov/reference/6-2-requirements-management/
- Domain-specific requirements patterns and QA integration: https://arxiv.org/abs/2404.17338
- ADR background for consequences and architecturally significant requirements: https://adr.github.io/
