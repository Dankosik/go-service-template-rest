# AGENTS.md

Repository-wide contract for reliable Go-service changes with the least workflow that can prove them.

## Engineering Judgment

- Use the narrowest current evidence that can prove or falsify the next claim.
- Reuse the current owner, repository pattern, standard library, and installed dependencies before adding machinery.
- Keep behavior, failures, cleanup, and proof at their narrowest owner. Prefer concrete types and explicit control flow.
- Treat cancellation, deadlines, partial work, cleanup, shutdown, generated authority, and mutable ownership as first-class only when the change touches them.
- During iteration, use cached focused checks and keep reusable local dependencies running. Reserve uncached tests, race, coverage, full lint, rebuilds, and teardown for a triggered claim or publication evidence; do not clean caches as a speed technique.

## Collaboration

- Lead with the conclusion. Separate established facts, inferences, trade-offs, and proof gaps.
- Challenge a design with concrete consequences and a viable smaller alternative; when choices are comparable, prefer clearer ownership, failure signals, and recovery.

## Authority And Loading

- Explicit user, system, and developer instructions win.
- This file owns request authorization and repository-wide invariants.
- Skills provide methods; they do not override this contract or task-local decisions.
- [docs/spec-first-workflow.md](docs/spec-first-workflow.md) is the workflow router. Read only the current phase file and any shared file needed for the decision at hand.
- Task-local artifacts own accepted task decisions. Runtime and generated-source authorities named by those artifacts still win over derived prose.

## Authorization And Boundaries

- `answer`, `explain`, `review`, `diagnose`, and `plan` authorize inspection and reporting only. `change`, `build`, and `fix` authorize in-scope local edits and non-destructive validation.
- Ask before external writes, destructive actions, purchases, or material scope expansion. Do not ask before ordinary reads, in-scope edits, or tests.
- Respect explicit `read-only`, `docs-only`, `research only`, and named-phase boundaries.
- A Codex Goal is an execution control for implementation only. Do not create or continue one during intake, research, specification, technical design, test design, planning, or their review and repair loops, even when those phases edit repository workflow artifacts.
- For ordinary non-interactive shell calls, set `login: false`; use a login shell only when it materially needs initialization.

## Routing

`docs/spec-first-workflow.md` owns path selection. Choose the smallest path that can close the accepted outcome:

- **Direct:** the request is clear, local, reversible, has one owner, bounded proof, and no unresolved protected-domain decision. The root may edit the assigned checkout, self-review the bounded diff, and run focused proof. No Goal, App Worker, worktree, durable artifact, independent review, or workflow opt-out is required.
- **Structured:** the normal non-trivial case. Keep a reviewed `spec.md` and reviewed `tasks.md`; create design and test artifacts only when their decisions must survive.
- **Orchestrated:** use durable coordination, parallel lanes, or optional Goal and App Worker/worktree execution only when broad or multi-owner scope, hard-to-reverse decisions, conflicting evidence, explicit multi-agent work, dirty-checkout isolation, separate context, or likely multi-session execution makes coordination real.

Public contracts, persisted data, security, money, concurrency/lifecycle, deployment, and cross-service ownership require explicit relevant decisions and proof. They do not automatically require every artifact, reviewer, worker, or full validation suite. When an accepted outcome spans multiple deployables, repositories, or managed dependencies, apply [System Release Closure](docs/spec-first-workflow/phases/system-integration-design.md#system-release-closure); cover the full affected deployment graph, or narrow the claim and name the external blocker.

### Required Spine

Structured and orchestrated work evaluates the phase router in order:

1. establish the accepted outcome at intake;
2. resolve decision-changing evidence, or state why research is unnecessary;
3. complete specification and independent specification review;
4. complete system and Go-ownership design when implementation would otherwise choose mechanism or placement, then independently review the design;
5. complete test design when proof is non-obvious, then obtain independent QA review;
6. complete `tasks.md` and independent task review/readiness;
7. enter [Implementation / Validation / Closeout](docs/spec-first-workflow/phases/implementation-validation-closeout.md) with one direct outcome or the next ready planned ledger wave.

Scoping down research, design, or test design needs one concrete reason in the current artifact or handoff. Specification, planning, and their independent review gates remain required for structured and orchestrated work.

Before each required review of Specification, combined Technical Design, Test Design, Planning, or an explicit `research only` boundary, run one autonomous read-only challenge probe after the whole candidate meets its authoring bar and before a different independent reviewer. Direct work, supporting steps, and Implementation / Validation / Closeout do not run this probe. [Autonomous Pre-Review Challenge](docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md) owns the protocol.

Before structured or orchestrated work designs against an external platform, unfamiliar mechanism, new infrastructure or dependency, or non-trivial architecture choice, research current official documentation/source and credible real implementations or engineering writeups. Treat official sources as contract authority, real-world sources as operational evidence, and do not rely on model memory for current external behavior.

## Working Contract

1. Reconstruct the intended outcome before acting. Inspect repository facts instead of asking the user for facts the repository can answer. Ask only for a decision that would materially change scope, behavior, ownership, safety, or proof; otherwise state a bounded assumption and continue.
2. Describe the outcome, constraints, success criteria, and stop conditions. Do not prescribe steps the model can choose reliably, repeat rules across files, or create artifacts solely to prove that process happened.
3. Choose the smallest path that preserves correctness. Direct work may proceed without workflow artifacts; otherwise use [the workflow router](docs/spec-first-workflow.md), which owns path selection, phase order, review gates, and movement rules. Respect a user-named phase boundary.
4. Public contracts, persisted data, security, money, concurrency/lifecycle, deployment, and cross-service ownership require explicit relevant decisions and proof, but not automatically full-depth work or a durable artifact in every phase. Apply [System Release Closure](docs/spec-first-workflow/phases/system-integration-design.md#system-release-closure) when the accepted outcome spans multiple deployables, repositories, or managed dependencies.
5. Evidence before invention. Prefer current Go stdlib and established repository patterns. For structured or orchestrated work, research current external contracts and credible operational evidence before choosing an unfamiliar platform, mechanism, infrastructure, dependency, or non-trivial architecture.
6. Keep ownership explicit. Put substantial code in the narrow owning package/file, preserve generated-source discipline, and remove replaced code and adjacent stale artifacts unless current compatibility evidence justifies retention.
7. Skills define method. [Subagents And Handoff](docs/spec-first-workflow/shared/subagents-and-handoff.md) owns built-in subagent eligibility, mandatory non-implementation review independence, and handoff; [Autonomous Pre-Review Challenge](docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md) owns its protocol; the implementation phase retains its current local/direct and optional Worker execution contract. The root owns synthesis and decisions.
8. Do not claim ready, complete, fixed, or covered without current evidence matched to the claim. Report unavailable or narrower proof honestly and name the next useful check.

## Implementation And Evidence

- The root may implement direct local work. Use a native Codex App Worker and managed worktree only when the routing triggers above apply or the user explicitly delegates implementation.
- A Codex Goal is for genuinely long-running, multi-step, or resumable implementation; one root Goal spans that outcome. Do not create one for ordinary direct work or non-implementation reasoning.
- Inspect the owning code, callers, siblings, tests, and generated/manual boundary before editing. Fix defects at the narrowest shared owner proved by the reproducer.
- Review only changed and transitively affected surfaces. Return all evidence-backed compatible findings together; do not create ceremonial re-review loops.
- Map every completion claim to current proof of equal scope. Proof for an immutable commit/tree remains valid after a byte-identical fast-forward; rerun only when the tree, relevant environment or preconditions, claim scope, provenance, or risk surface changed.
- Never claim complete, fixed, ready, or covered beyond current evidence. State unavailable proof and the reopen owner.

## Validation Matrix

Use the smallest matching check:

| Changed surface | Default proof |
| --- | --- |
| Docs or instructions | `git diff --check` and the relevant instruction gate |
| Local Go behavior | Focused package/test proof; changed-code lint when useful |
| Concurrency/lifecycle | Focused behavior plus race/liveness proof |
| OpenAPI, sqlc, migration, generated source | Canonical generation/drift and affected runtime proof |
| Security, deployment, cross-service or release | The matching protected-domain and integrated proof |
| Publication, CI parity, or broad cross-cutting change | `check-full`, `ci-local`, `pr-check`, container, or security suites only when the claim needs them |

`make check` is a broad local baseline, not the default edit loop. Do not run service tests for docs-only work or broad suites merely because they exist.

## Go Change Surface

When work can change Go, classify only the triggered pressures: package owner,
import direction, composition, or exported surface; method sets, nil/zero,
errors, or context; resource or transaction lifetime; mutable ownership,
aliasing, concurrency, or lifecycle; canonical, generated, or hand-written
authority; and repository-native proof. Activate only the matching existing Go
methods; untriggered categories create no work. Close every triggered category
with its phase or skill owner, or name the owner and condition that must reopen
it.

## Instruction Ownership

- Keep global rules here.
- [Skill authoring](docs/skill-authoring.md) owns the lean behavioral-adapter contract.
- `docs/spec-first-workflow.md` owns routing and movement.
- Keep phase-specific method in `docs/spec-first-workflow/phases/`.
- `shared/artifact-model.md` owns persistence; `shared/subagents-and-handoff.md` owns built-in subagent delegation, mandatory non-implementation review independence, convergence, and handoff; `shared/autonomous-pre-review-challenge.md` owns the autonomous challenge protocol.
- Skills provide methods; they do not create work or override accepted decisions.
- Task-local artifacts own accepted task decisions. Runtime and generated authorities named there win over derived prose.
- Keep a rule in its narrowest owner; replace duplicates with links or delete them.
