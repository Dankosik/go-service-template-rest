# Input Bundle Sufficiency

## Behavior Change Thesis
When loaded for a thin or underspecified input bundle, this file makes the model return a missing-input approval blocker instead of likely mistake inventing candidate decisions, acceptance constraints, or fake validation expectations.

## When To Load
Load this when the orchestrator's bundle may be too thin to run an honest approval challenge. The right output may be "blocked by missing bundle evidence" rather than a list of guessed questions.

This reference helps distinguish missing input that blocks the clarification gate from missing implementation detail that can be deferred.

## Sufficiency Test
A challenge bundle is sufficient when it gives enough context to identify approval-changing seams:

- problem frame and intended outcome
- scope and non-goals
- candidate decisions
- constraints and validation expectations
- known assumptions and open questions
- relevant research links or lane outputs when a claim depends on evidence

If the bundle lacks candidate decisions or acceptance constraints, do not invent them. Say what is missing and classify the gap as blocking approval.

## Strong Vs Weak Questions

### Missing candidate decisions
Strong:

> The bundle gives a problem frame for async exports but no candidate decisions about job state persistence, artifact access, or failure semantics. Which candidate decisions is the orchestrator asking this gate to challenge, and should `spec.md` remain draft until those decisions exist?

Correct classification: `blocks_spec_approval`

Recommended next action: `answer_from_existing_evidence`

Weak:

> Please provide more details.

Why weak: it does not identify the missing approval surface.

### Missing validation expectations
Strong:

> The bundle says account-summary caching should reduce load but gives no validation expectation for stale data, fallback behavior, or cache key isolation. Which correctness checks must be true before approval, and are any of them already evidenced in research?

Correct classification: `blocks_spec_approval`

Recommended next action: `answer_from_existing_evidence` or `targeted_research` if the bundle points to a missing factual claim.

Weak:

> How will this be tested?

Why weak: it asks for a test plan rather than the validation expectation needed for spec approval.

### Missing scope/non-goal boundary
Strong:

> The bundle proposes admin deactivation but does not say whether reactivation, deletion, session revocation, and integration shutdown are in or out of scope. Which of these are non-goals versus approval constraints, and would a different answer change the accepted behavior?

Correct classification: `blocks_spec_approval`

Recommended next action: `answer_from_existing_evidence`

Weak:

> What is out of scope?

Why weak: it is generic and does not show why the boundary matters.

### Missing evidence link for a risky assumption
Strong:

> The bundle assumes tenant IDs are globally unique but gives no repo evidence or research link. Can the orchestrator answer that from the schema/ownership notes, or should a data lane reopen tenant keying before approval?

Correct classification: `blocks_specific_domain`

Recommended next action: `expert_subagent` for data, or `answer_from_existing_evidence` if the orchestrator already has the schema evidence.

Weak:

> Is account ID unique?

Why weak: it lacks the approval consequence: cache key isolation and tenant safety.

## Correct Result For A Thin Bundle
When the bundle is too thin, return a short blocker instead of pretending:

- `Clarification Summary`: "The gate cannot run honestly because candidate decisions and validation expectations are missing."
- `Questions`: one to three missing-input questions, each tied to approval impact.
- `Reopen / Rerun Recommendation`: "Repair the input bundle, then rerun this challenge once."
- `Confidence`: low for the challenge result, high for the input-gap diagnosis.

## Agent Traps
- Do not invent missing candidate decisions from the problem frame.
- Do not ask for implementation details when the missing item is an approval-level boundary.
- Do not classify a bundle as sufficient just because it contains many words; it must contain challengeable decisions and proof expectations.
