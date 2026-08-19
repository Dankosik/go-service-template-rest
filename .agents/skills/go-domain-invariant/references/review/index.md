# Reference Selector

Select the reference that changes the finding, then cite local authority or state the bounded inference. The [shared finding envelope](../../../../../docs/spec-first-workflow/shared/review-findings-and-convergence.md#finding-envelope) owns the finding's shape — anchor, impact, classification, smallest action or reopen owner — so these references own only what makes a domain finding true.

| Symptom | Load | Distinction preserved |
| --- | --- | --- |
| Construction, mutation, save, guard, status update, or transition table may admit a state the domain forbids. | [invalid-state-and-transition-review.md](invalid-state-and-transition-review.md) | Prove the reachable bad state or the rejected move, not aggregate reshaping or a demand for a formal state machine. |
| An external or durable effect can happen before acceptance, happen twice, or overwrite newer state. | [effect-escape-and-duplication-review.md](effect-escape-and-duplication-review.md) | Prove the escaped or repeated effect before proposing a saga, an outbox, or dedupe. |
| A command, error, no-op, duplicate, event, or renamed domain term changes what the result means. | [acceptance-and-rejection-semantics.md](acceptance-and-rejection-semantics.md) | Preserve accepted acceptance meaning and load-bearing vocabulary, not error-style or naming taste. |
| Changed business behavior has no assertion that would fail on the regression. | [domain-test-traceability.md](domain-test-traceability.md) | Name the regression that can pass green, not generic coverage work. |
