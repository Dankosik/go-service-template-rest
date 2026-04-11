# Required Design Artifact Examples

Use these examples to shape the required core design bundle for non-trivial work. Keep final decisions in `spec.md`; keep task-local technical context in `design/`; keep execution order for the later planning session.

## When To Load
- Load when creating or repairing `design/overview.md`, `design/component-map.md`, `design/sequence.md`, or `design/ownership-map.md`.
- Load when existing design prose is scattered across workflow files or chat and needs to be normalized into the design bundle.
- Load when you need examples of good and bad artifact boundaries before handing off to planning.

## Good Design-Session Outputs
- `design/overview.md`: records the selected technical approach, links the required and triggered artifacts, lists unresolved seams, and states whether planning can start next.
- `design/component-map.md`: names affected packages or modules, what changes in each, and what remains stable.
- `design/sequence.md`: describes request, async, startup, shutdown, or recovery flow with call order, side effects, failure points, and sync/async boundaries.
- `design/ownership-map.md`: states source-of-truth ownership, allowed dependency direction, generated-code authority, adapter responsibility, and what must not own the behavior.

## Bad Design-Session Outputs
- A single `design.md` that mixes overview, package map, sequence, ownership, rollout, and implementation steps.
- `workflow-plans/technical-design.md` containing the real component map while `design/component-map.md` stays empty.
- `design/sequence.md` saying "handler calls service and saves data" without failure points, side effects, or async boundary.
- `design/ownership-map.md` listing packages but not source-of-truth ownership or allowed dependency direction.
- A design bundle that tells the coder to "decide during implementation" for a planning-critical ownership or contract question.

## Conditional Artifact Examples
- If `design/sequence.md` reveals durable state transitions or schema evolution, add `design/data-model.md`.
- If `design/component-map.md` introduces new dependency direction or generated-code flow, add `design/dependency-graph.md`.
- If `design/ownership-map.md` identifies changed OpenAPI, event, generated, or internal interface contracts, add `design/contracts/`.
- If the required artifacts prove validation or rollout is too large to summarize cleanly later, trigger `test-plan.md` or `rollout.md` and record that status in workflow control.

## Blocked Handoff Examples
- Required artifacts exist but disagree on which layer owns the source of truth.
- The overview says "planning-ready" while sequence or ownership still contains a blocker.
- The component map requires a new package boundary but the dependency direction is not designed.
- The sequence depends on a contract behavior not approved in `spec.md`.

## Exa Source Links
Exa MCP search and fetch were attempted before writing these examples, but the provider returned a 402 credits-limit error. Treat these fallback-verified links as calibration only; repo-local files remain authoritative.

- C4 model: https://c4model.com/
- arc42 documentation: https://docs.arc42.org/home/
- Google Cloud Architecture Framework: https://docs.cloud.google.com/architecture/framework
- Google Cloud Architecture decision records: https://docs.cloud.google.com/architecture/architecture-decision-records

