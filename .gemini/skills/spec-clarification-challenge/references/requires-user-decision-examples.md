# Requires User Decision Examples

## Behavior Change Thesis
When loaded for a possible external policy question, this file makes the model reserve `requires_user_decision` for non-repo-answerable product, business, support, contractual, policy, or legal choices instead of likely mistake asking the human for technical facts or inventing policy.

## When To Load
Load this when a question may require external product, business, policy, contractual, legal, or support judgment that repo evidence and safe engineering assumptions cannot answer.

Use `requires_user_decision` sparingly. First ask whether the orchestrator can answer from existing repo evidence, research notes, or a read-only expert lane.

## Correct Use
Use `requires_user_decision` when all are true:

- the question is approval-changing or must remain explicit in `spec.md`
- repo evidence cannot answer it
- targeted technical research cannot answer it
- a safe engineering default would invent product/business/legal policy
- approval should remain blocked, partial, or explicitly risk-accepted until the user decision exists

## Strong Vs Weak Questions

### Paid-customer deactivation policy
Strong:

> The spec lets support deactivate paid customers per case, but no repo evidence records whether manager approval, customer notice, or billing/legal review is required. Is this an external support/business policy decision that must be made before approving destructive account semantics?

Correct classification: `blocks_spec_approval`

Recommended next action: `requires_user_decision`

Weak:

> Should support be allowed to deactivate paid customers?

Why weak: it asks the human broadly without explaining why repo evidence is insufficient or why approval changes.

### Retention period owned by product or compliance
Strong:

> The spec chooses seven-day export artifact retention for all tenants. If no repo policy or contract evidence exists, does product/compliance need to decide the allowed retention window before approval?

Correct classification: `blocks_spec_approval`

Recommended next action: `requires_user_decision`

Weak:

> Is seven days okay?

Why weak: it omits the policy owner and approval consequence.

### Abuse response severity threshold
Strong:

> The problem frame says the admin action stops abusive or compromised accounts, but no policy defines when to use deactivation versus investigation-only. Is the trigger threshold an external product/support policy decision, or can existing abuse-response docs answer it?

Correct classification: `blocks_spec_approval` if the threshold changes allowed behavior; otherwise `non_blocking_but_record`.

Recommended next action: `answer_from_existing_evidence` first; `requires_user_decision` if evidence is absent.

Weak:

> What is the abuse policy?

Why weak: it is too broad and does not tell the orchestrator what to reconcile.

### Not a user decision: audit for internal mutation
Strong:

> Audit logging is deferred for an internal destructive action. Can security or admin-control repo evidence determine whether audit is mandatory before approval?

Correct classification: `blocks_specific_domain`

Recommended next action: `expert_subagent` for security, not `requires_user_decision` unless the expert lane finds an external policy owner.

Weak:

> Ask the user whether audit logging is required.

Why weak: it skips repo evidence and expert research.

### Not a user decision: implementation mechanism
Strong:

> If the spec already requires immediate session revocation, should the exact revocation mechanism be deferred to `design/`?

Correct classification: `non_blocking_but_record`

Recommended next action: `defer_to_design`

Weak:

> Ask the user which table stores sessions.

Why weak: it asks the human for a repo-answerable implementation detail.

## Output Guidance
When using `requires_user_decision`, say why the spec should stay blocked, partially draft, or risk-accepted. Do not ask the human to decide ordinary technical facts.

## Agent Traps
- Do not use `requires_user_decision` to avoid targeted research.
- Do not invent product, support, contractual, or legal policy to approve a spec.
- Do not ask the human for repo-answerable technical facts.
