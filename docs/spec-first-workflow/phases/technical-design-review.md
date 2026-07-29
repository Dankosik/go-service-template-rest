# Technical Design Review

Apply the shared [Review Independence](../shared/subagents-and-handoff.md#review-independence) contract. This adapter adds only the technical-design falsification kernel and design-specific verdict threshold; it does not define another workflow phase.

## Read When

- The shared [independent-review trigger](../shared/subagents-and-handoff.md#review-independence) applies to a completed fixed technical design.
- The user requests a standalone independent review of a completed fixed technical design.

## Inputs

- Ready spec and current fixed system/integration design plus Go ownership design when present, or their current diff.
- Relevant repository architecture, current runtime/generated authorities, and affected consumer surfaces.
- Dispositioned accepted risks/downstream proof obligations and still-valid prior findings.

## Outputs

Use the shared [Review Finding Envelope](../shared/subagents-and-handoff.md#review-finding-envelope). Add the earliest unsupported design edge and its smallest reopen owner.

Design-specific verdict threshold:

- `PASS`: every reconstructed material trace and triggered lens is supported within the stated evidence boundary; no downstream owner must invent behavior, mechanism, authority, or ownership.
- `CONCERNS`: only bounded downstream risks or proof obligations dispositionable under the shared rule remain; no behavior, mechanism, authority, ownership, or required-input choice remains open.
- `FAIL`: an earliest unsupported edge requires Specification, System / Integration Design, or Go Code / Ownership Design to close or change a decision, or a named external owner to supply a blocking input before movement.

## Review Method

Independently reconstruct every material behavior trace from the ready spec, current runtime/generated evidence, and affected consumer surfaces. Candidate omission does not establish that a trace or lens is untriggered.

For each trace, follow accepted behavior and explicit design drivers -> viable same-level substitutes or evidence that no real fork remains -> selected architecture -> ordered material flow and finality -> contract or data authority when relevant -> system owner and package/file owner when Go ownership is triggered -> implementation input required by the next owner -> proving surface when non-obvious proof is triggered. The first edge without compatible evidence is the earliest unsupported edge and the finding anchor. Suppress consequences downstream of that edge, but continue every unaffected trace and triggered lens before issuing the verdict.

At each triggered implementation-input edge, verify only whether the input is canonical, mechanically derivable without a semantic choice, or available from a named external owner under [Implementation-Input Closure](../../spec-first-workflow.md#implementation-input-closure). Record the first unmet input and its reopen owner. Materialization and semantic choice remain with the owning phase. For byte- or signature-sensitive behavior, verify that the design names the canonical golden vector and its authority; reproduction remains with the proof owner.

Activate only falsification lenses triggered by the reconstructed affected surface:

- Flow and authority: could two reasonable implementations follow the candidate yet differ in caller- or operator-visible behavior, ordering/finality, contract or data authority, failure/recovery, or rollout?
- Cross-flow coherence: do shared components, contracts, stores, authorities, state transitions, and failure/recovery policies have one compatible meaning across all material traces, or does one trace silently require a different owner, ordering, finality, synchronization, degradation, or recovery boundary?
- Current evidence: does a runtime, generated, provider, or consumer authority contradict the selected mechanism, contract, owner, or transition?
- Ownership, when placement, dependency direction, generated authority, or cleanup is material: does every selected system decision entering Go map exactly once to a feasible responsibility and package/file owner, does that owner fit the current import and composition graph, and is every live ownership fork closed?
- Performance, when a path is scale-sensitive or has an accepted objective: can work amplification exceed the workload or budget, or can a simpler established mechanism satisfy the same decision?
- Synthesis and selection: when current evidence leaves a real fork, are materially distinct viable substitutes compared at the same decision level against the same explicit drivers and evidence, does the selected architecture dominate, and was a simpler deletion, reuse, native, or established mechanism omitted? When no fork is recorded, does current evidence actually collapse it?
- Proof, when the mechanism creates non-obvious proof or an accepted downstream proof obligation: is the named proving surface feasible from available observables and inputs without choosing design?
- Machinery, when the candidate adds an abstraction, dependency, or mechanism: what present requirement or accepted constraint makes it necessary?

A reviewer may name a concrete evidence-backed simpler substitute only to falsify the selected architecture and anchor a System / Integration Design reopen; composing or selecting the replacement remains with that owner.

Implementation-local naming and task order remain downstream details unless they change a contract, ownership, or proof feasibility. Output ends at findings, reopen owners, and verdict; task breakdown, test design, implementation artifacts, and repair remain with their owning phases.

## Stop Rule

Stop after the complete compatible finding set and verdict. Apply the shared disposition, convergence, fresh-review, and standalone-review boundaries.
