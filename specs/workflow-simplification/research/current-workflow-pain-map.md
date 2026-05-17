# Current Workflow Pain Map

## Scope

This note maps the current `go-service-template-rest` workflow against the user's concern: the phases carry real value, but the real template usage is too heavy for modern LLM agents.

## Local Evidence

| Evidence | What It Shows |
| --- | --- |
| `AGENTS.md:35-53` | Durable quality invariants: orchestrator-owned decisions, task-ledger-gated implementation, spec clarification before approval, advisory reviews, and fresh validation evidence. |
| `AGENTS.md:60-63` | Existing execution shapes already distinguish direct path, lightweight local, and full orchestrated work; the redesign renames/promotes lightweight local as `lean local` while preserving compatibility with the older term. |
| `AGENTS.md:55` and `docs/spec-first-workflow.md:302-308` | Current default is one named phase per session for non-trivial work unless an upfront waiver exists. |
| `docs/spec-first-workflow.md:79-85` | Current non-trivial model requires master workflow state, phase-local workflow plans, design, tasks, implementation readiness, and optional research. |
| `docs/spec-first-workflow.md:108-117` | The artifact matrix shows useful ownership boundaries, but also spreads context across many files. |
| `docs/spec-first-workflow.md:222-225` | Technical design is split into overview, component map, sequence, and ownership map by default. |
| `docs/spec-first-workflow.md:437-476` | `tasks.md` and implementation readiness encode strong executable handoff and proof obligations. |
| `docs/spec-first-workflow.md:829-839` | The detailed workflow already admits direct-path and lightweight-local collapse, but the exception path is buried late in the document. |
| `specs/railway-auto-migrations/workflow-plan.md:1-10` | A real successful template change used lightweight local, collapsed phases, and locally reconciled challenge gates. |
| `specs/railway-auto-migrations/spec.md:1-54` | The spec captured context, scope, constraints, decisions, validation, and outcome compactly. |
| `specs/railway-auto-migrations/design/*.md` | The real design bundle answered useful questions, but across four short files. |
| `specs/railway-auto-migrations/tasks.md:1-24` | The task ledger was concise and carried strong execution value: dependencies and proof per task. |

## Current Quality Carriers

These should be preserved unless replaced by a stronger mechanism:

1. Orchestrator accountability.
   - Current invariant: final decisions and synthesis remain with the orchestrator, not subagents or skill output.
   - Quality value: prevents lane consensus from becoming accidental architecture.

2. Canonical `spec.md`.
   - Current invariant: stable decisions live in `spec.md`.
   - Quality value: later design, planning, implementation, review, and validation have one source of truth.

3. Execution-shape distinction.
   - Current workflow already has `direct path`, `lightweight local`, and `full orchestrated`.
   - Quality value: lets small work avoid process load while preserving a path for complex changes.
   - Simplification move: make `lean local` the first-class bounded non-trivial default and treat `lightweight local` as the compatibility alias.

4. Design questions.
   - Component, sequence, and ownership questions are the real value of the current design bundle.
   - Quality value: prevents implementation from inventing ownership, flow, and stable surfaces.

5. Executable task ledger.
   - `tasks.md` carries task IDs, dependencies, surfaces, and proof expectations.
   - Quality value: modern agents implement better from bounded, verifiable slices.
   - Simplification move: for lean implementation, shift working discipline into `tasks.md` instead of phase-control files.

6. Fresh validation evidence.
   - Completion claims require fresh evidence, and the Railway bundle closed with real commands.
   - Quality value: prevents "looks done" from replacing proof.

7. Read-only advisory subagents.
   - Current contract keeps subagents advisory and read-only.
   - Quality value: preserves safe delegation without moving final judgment away from the orchestrator.

## Current Overhead Drivers

1. Phase-control duplication.
   - `workflow-plan.md` and `workflow-plans/<phase>.md` often carry overlapping status, blockers, next action, and artifact states.
   - Cost: agents must read and keep synchronized multiple routing surfaces before doing the actual engineering work.

2. Mandatory-looking phase separation.
   - One phase per session is valuable for high-risk work, but too heavy for bounded template changes.
   - Cost: real work uses waivers, which means the documented default and practical usage diverge.

3. Split design bundle for small changes.
   - The component, sequence, and ownership questions are valuable; four separate files are not always valuable.
   - Cost: short files create navigation and synchronization overhead without adding insight.

4. Challenge gates as artifact ceremony.
   - Workflow adequacy and spec clarification are useful when ambiguity is high or multiple lanes exist.
   - Cost: for local bounded work, they become local reconciliation notes rather than real independent challenge.

5. Optional artifacts described as default-looking.
   - `research/`, `test-plan.md`, `rollout.md`, review phase files, and validation phase files are conditional, but the full layout makes them feel expected.
   - Cost: agents may create files "for completeness" instead of because a trigger exists.

6. Late visibility of lightweight path.
   - The direct/local exception exists, but the prominent default path is still large.
   - Cost: agents start in full ceremony mode unless explicitly pushed toward lean local.

7. Status drift surface.
   - The more status files exist, the more likely `current phase`, `next action`, artifact status, and validation claims diverge.
   - Cost: future sessions spend tokens reconciling stale state instead of making progress.

## Real Bundle Lessons From `railway-auto-migrations`

The Railway bundle is strong evidence that the desired simplification is evolutionary:

- The actual successful flow used `lightweight local`, not full orchestrated.
- The useful target term is `lean local`: the same bounded-local concept, but framed as a normal mode rather than a waiver.
- Research stayed local even though external docs were consulted.
- Challenge gates were locally reconciled, not delegated.
- The spec was compact and useful.
- The design questions were useful, but the four-file split produced very short artifacts.
- `tasks.md` was high-value because it gave bounded implementation slices and proof expectations.
- Validation evidence was concrete and should remain non-negotiable.

## Practical Interpretation

The workflow should preserve the quality carriers and reduce the forced surface area:

- Keep decision ownership, canonical spec, design questions, task ledger, and proof.
- Make phase-local workflow plans conditional.
- Merge design questions into `spec.md` or one design artifact for lean local work.
- Make challenge gates trigger-based and record local self-checks when risk is low.
- Promote direct path and lean local trigger rules near the top of the docs and skills.
- Treat full orchestrated flow as the high-risk path, not the default mental model for every non-trivial task.
- Use an inline `Risk Challenge` in lean local to decide `PASS`, `CONCERNS`, or `FULL_REQUIRED` before coding.
- Prefer delta-style `ADDED / MODIFIED / REMOVED` behavior sections in lean specs so brownfield changes do not become full service re-documentation.
