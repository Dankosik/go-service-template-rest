# Shared Specialist Contract

Non-triggerable common owner for domain specialists; it does not own workflow, execution, or domain policy. Select the domain by the decision or violated contract, then select **decision** when policy is absent or changing and **review** when changed code or artifacts must conform to accepted policy. Load one reference by default and another only for an independent pressure.

A decision returns `constraint_only`, `proof_only`, or `no_new_decision_required` when applicable; otherwise it states the decision, forced consequences, proof, and blocker. A review uses current evidence and the [shared finding envelope](../../docs/spec-first-workflow/shared/subagents-and-handoff.md#review-finding-envelope) to report the smallest safe correction and focused proof, or no findings without padding. Missing policy re-enters the same domain's decision branch when it exists, otherwise the nearest affected domain decision owner. Load [specialist arbitration](specialist-arbitration.md) only for ambiguous overlap.

During [Implementation](../../docs/spec-first-workflow/phases/implementation-validation-closeout.md), `go-coder` owns the change and the phase's Candidate Acceptance section owns specialist review composition and finding fan-in.
