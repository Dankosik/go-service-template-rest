# Shared Specialist Contract

This file is a non-triggerable instruction owner loaded by specialist specification and review skills. Apply the role-labelled mechanics below together with the loading skill's domain rules; do not use this contract as a workflow, execution, or domain owner.

## Common

- Select one primary specialist by the decision or violated contract, not by a shared symptom. Stay on that primary axis and preserve the loading skill's domain invariants, evidence distinctions, and stop conditions.
- Load one symptom-matched reference by default. Load more only for independent pressures that materially affect the result.
- Describe alternatives only when a live fork has at least two viable choices. Record only handoffs forced by the current decision or defect, and do not duplicate the neighboring owner's work.

## Specification

- Decide only the loading skill's owned policy axis. Stop on an unset policy owned by another specialist instead of inventing it.
- Use `constraint_only` when another owner must preserve a consequence, `proof_only` when only validation work remains, and `no_new_decision_required` when accepted policy already resolves the domain.
- Return no new decision without padding when the accepted behavior is already complete; otherwise state the owned decision, forced consequences, proof obligation, and unresolved blocker.

## Review

- Review against accepted behavior and current evidence. Do not invent missing policy; escalate it to the named specification owner.
- Report only evidence-backed failure paths, the smallest safe correction, and focused proof. Return no findings without padding when no material defect exists.
- Use the repository's [shared review finding envelope](../../docs/subagent-contract.md#shared-review-finding-envelope), preserving any domain-specific severity or evidence rules from the loading skill.

## Primary-Axis Arbitration

These tables route defects against accepted behavior; a missing or changed policy goes to the named specification owner and must not be invented by a reviewer.

| Decisive defect axis | Primary review owner | Unset or changed decision owner |
| --- | --- | --- |
| Go language or stdlib contract: panic, nil/zero value, aliasing, receiver/method set, context API/lifetime, error, or resource semantics | `go-idiomatic-review` | nearest affected domain spec; use `go-implementation-ownership-spec` only for placement/ownership |
| Behavior is correct, ownership is local, but control flow, predicates, naming, or helper shape is hard to understand or change | `go-language-simplifier-review` | owning behavior spec if simplification would change behavior |
| Explicit harsh/whole-diff review finds cross-file structural overbuild, abstraction cost, mixed-responsibility files, or missed deletion | `go-structural-quality-review` | `go-implementation-ownership-spec` when responsibility or placement must change; otherwise the owning behavior spec |
| Wrong package/file owner, dependency direction, source of truth, or implementation seam | `go-implementation-ownership-review` | `go-implementation-ownership-spec`; use `go-system-architecture-spec` when topology or system authority must change |

| Decisive defect axis | Primary review owner | Unset or changed decision owner |
| --- | --- | --- |
| Concrete happens-before, shared-state, goroutine, channel, timer, cancellation-unblock, or join protocol | `go-concurrency-review` | `go-reliability-spec` for missing service lifecycle policy; `go-implementation-ownership-spec` only for mechanism placement |
| Violation of accepted end-to-end timeout/deadline, retry, overload, degradation, readiness, startup, drain, shutdown, recovery, or rollout behavior | `go-reliability-review` | `go-reliability-spec` |
| Replay, partial failure, ordering, compensation, redrive, or reconciliation across a durable boundary | `go-distributed-review` | `go-distributed-spec` |

Route by the violated contract, not the shared symptom. Replacing caller context on a blocking dependency path is reliability when it breaks an accepted end-to-end deadline/cancellation contract; a goroutine that cannot observe cancellation and join is concurrency; storing a call-scoped context or accepting an invalid nil context is idiomatic Go. A WaitGroup/channel shutdown hang is concurrency, readiness/drain/rolling-shutdown behavior is reliability, and resume/replay after durable process loss is distributed. Local readability stays with simplification; whole-diff overbuild stays with structural quality; wrong ownership or dependency direction stays with implementation ownership.

Neighboring skills link to these tables instead of restating overlapping ownership. Reliability retains local lifecycle-policy checks but does not duplicate concurrency mechanics or distributed policy.
