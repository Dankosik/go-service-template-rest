# Clarification Anti-Patterns

## Behavior Change Thesis
When loaded for a bloated or answer-heavy draft, this file makes the model prune checklist noise, design authorship, human-escalation shortcuts, and approval theater instead of likely mistake returning a second spec or generic review.

## When To Load
Load this when a draft clarification pass feels bloated, generic, answer-heavy, or too eager to approve. Use it to prune output before returning the deliverable.

The clarification gate is not a checklist, design review, or second spec. It is a compact approval-risk pass.

Use this as a challenge/smell-triage reference, not primary design guidance. If a narrower positive reference matches the current uncertainty, load that instead.

## Strong Vs Weak Clarification Rewrites

### Generic checklist question
Weak:

> What about security?

Better:

> The signed URL decision relies on tenant lookup before URL issuance. What prevents URL reuse outside tenant context before expiry, and would that change the approved download contract or validation proof?

Correct classification: `blocks_specific_domain`

Recommended next action: `expert_subagent`

### Answering instead of asking
Weak:

> The spec should add immediate session revocation, audit logs, and integration shutdown.

Better:

> The problem frame includes compromised accounts, but the candidate decision lets sessions and integrations continue naturally. Which side effects must deactivation stop immediately for the spec to satisfy its stated intent?

Correct classification: `blocks_spec_approval`

Recommended next action: `answer_from_existing_evidence` or `expert_subagent`

### Design authorship
Weak:

> Implement export cleanup with a nightly job and store cleanup status in Postgres.

Better:

> The spec already fixes seven-day retention but not cleanup mechanics. Should the gate record cleanup as a `design/` concern while preserving retention and validation obligations in `spec.md`?

Correct classification: `non_blocking_but_record`

Recommended next action: `defer_to_design`

### Human escalation by default
Weak:

> Ask the user whether audit logs are required.

Better:

> Can repo security/admin-control evidence determine whether audit is mandatory for internal destructive actions, or is this an external policy decision that should keep `spec.md` draft?

Correct classification: `blocks_specific_domain` unless repo evidence proves it is external policy.

Recommended next action: `expert_subagent` first; `requires_user_decision` only if evidence cannot answer.

### Defer-to-design escape hatch
Weak:

> Tenant keying can be decided in design.

Better:

> The cache key omits tenant identity because `account_id` is assumed globally unique. Does approval require source-of-truth evidence for that assumption before design chooses the key shape?

Correct classification: `blocks_specific_domain`

Recommended next action: `expert_subagent`

### Record-only noise
Weak:

> Should we document every rejected export storage provider in `spec.md`?

Better:

> If the candidate spec only needs object storage retention and scoped download semantics, should rejected provider choices stay out of `spec.md` unless they change constraints or validation?

Correct classification: `non_blocking_but_record`

Recommended next action: `accept_risk` or `defer_to_design`

### Quota padding
Weak:

> Does this need observability? Does this need docs? Does this need metrics? Does this need tests?

Better:

> No question survives the approval-impact filter beyond the tenant-scoped download and duplicate-request seams; the gate should return fewer questions rather than padding.

Correct classification: no question, or `non_blocking_but_record` only for a specific missing spec note.

Recommended next action: `accept_risk` if the evidence boundary is explicit.

## Pruning Checklist
Drop a question when it:

- could not change approval
- asks for a best practice without a candidate-decision seam
- asks the human before checking repo evidence or expert research
- writes the answer, design, tasks, or implementation handoff
- duplicates another question with different wording only
- turns every uncertainty into a blocker

## Agent Traps
- Do not rewrite the candidate spec or design bundle.
- Do not add "security, docs, observability, tests" as category-only questions.
- Do not produce exactly 10 questions when only two survive the approval-impact filter.
- Do not mark the gate clear unless the evidence boundary is explicit.
