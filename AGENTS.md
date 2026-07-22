# AGENTS.md

Repository-wide contract for reliable Go-service changes with the least workflow that can prove them.

## Engineering Judgment

- Use the narrowest current evidence that can prove or falsify the next claim.
- Reuse the current owner, repository pattern, standard library, and installed dependencies before adding machinery.
- Keep behavior, failures, cleanup, and proof at their narrowest owner. Prefer concrete types and explicit control flow. Remove replaced code and adjacent stale artifacts unless current compatibility requires them.
- Treat cancellation, deadlines, partial work, cleanup, shutdown, generated authority, and mutable ownership as first-class only when the change touches them.
- During iteration, use cached focused checks and keep reusable local dependencies running. Reserve uncached tests, race, coverage, full lint, rebuilds, and teardown for a triggered claim or publication evidence; do not clean caches as a speed technique.
- For performance changes, follow [Benchmarking](docs/benchmarking.md): choose the narrowest matching Go, in-process HTTP, real-PostgreSQL, or external HTTP proof; define workload and budget before measuring; preserve raw baseline/candidate evidence where comparison applies; and keep correctness proof independent. Before executing a benchmark, prefer the DigitalOcean runner when `doctl` is installed and its selected context is authorized; use the matching local command only when that remote path is unavailable. Read the `digitalocean-benchmark-runner` skill before remote execution, and keep every paid lifecycle operation inside an explicitly authorized cost and lifecycle envelope. Run `make benchmark-infra-check` when benchmark tooling or scenario infrastructure changes. Persistent history or blocking automation requires a stable dedicated testbed and a named threshold owner.

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
- An approved external-write or purchase envelope owns its cost, security, and proof bounds. Inside it, choose live-state parameters such as region, equivalent host or size, bounded retry route, and local or remote execution from current evidence; a rejected route is no longer valid. Reopen authorization only to exceed the envelope, weaken required security or proof, or change scope or behavior.
- Respect explicit `read-only`, `docs-only`, `research only`, and named-phase boundaries.
- A durable execution control (a Codex Goal in the Codex App; the task list in Claude Code — see [Agent Harness](docs/agent-harness.md)) is for implementation only. Do not create or continue one during intake, research, specification, technical design, test design, planning, or their review and repair loops, even when those phases edit repository workflow artifacts.
- For ordinary non-interactive shell calls, avoid shell startup: set `login: false` when supported; otherwise set `shell: "/bin/bash"`. Use a login or interactive shell only when the command materially depends on its initialization or zsh-only syntax.

## Routing

`docs/spec-first-workflow.md` owns path selection. Choose the smallest path that can close the accepted outcome:

- **Direct:** the request is clear, local, reversible, has one owner, bounded proof, and no unresolved protected-domain decision. The root may edit the assigned checkout, self-review the bounded diff, and run focused proof. No durable execution control, native worker, worktree, durable artifact, independent review, or workflow opt-out is required.
- **Structured:** the normal non-trivial case. Keep only the `spec.md`, `tasks.md`, design, or test artifacts whose decisions must survive; the root self-reviews them unless current risk or the user requires independent review.
- **Orchestrated:** use durable coordination, parallel lanes, or optional durable-control and native worker/worktree execution only when broad or multi-owner scope, hard-to-reverse decisions, conflicting evidence, explicit multi-agent work, dirty-checkout isolation, separate context, or likely multi-session execution makes coordination real.

Public contracts, persisted data, security, money, concurrency/lifecycle, deployment, and cross-service ownership require explicit relevant decisions and proof. They do not automatically require every artifact, reviewer, worker, or full validation suite. When an accepted outcome spans multiple deployables, repositories, or managed dependencies, apply [System Release Closure](docs/spec-first-workflow/phases/system-integration-design.md#system-release-closure); cover the full affected deployment graph, or narrow the claim and name the external blocker.

### Required Spine

Structured and orchestrated work follows the workflow router's [Required Spine](docs/spec-first-workflow.md#required-spine), including its review, movement, and scoping-down rules.

Before structured or orchestrated work designs against an external platform, unfamiliar mechanism, new infrastructure or dependency, or non-trivial architecture choice, research current official documentation/source and credible real implementations or engineering writeups. Treat official sources as contract authority, real-world sources as operational evidence, and do not rely on model memory for current external behavior.

## Task Contract

1. Reconstruct the accepted outcome from current repository facts before acting. Ask the user only when an unresolved decision would materially change scope, behavior, ownership, safety, authorization, or proof; otherwise state a bounded assumption and continue.
2. State the outcome, non-obvious constraints or authority, matching proof, and stop condition. Omit inherited defaults, empty fields, and discretionary steps; prescribe an order or mechanism only when an accepted decision fixes it.
3. Match every readiness or completion claim to current evidence of equal scope. Report narrower or unavailable proof honestly and name the next useful check.

## Implementation And Evidence

- During implementation, follow the phase-owned [Acceptance Posture](docs/spec-first-workflow/phases/implementation-validation-closeout.md#acceptance-posture) and [Progress](docs/spec-first-workflow/phases/implementation-validation-closeout.md#progress). A real blocker ends implementation and is reported.
- The root may implement direct local work. Use the current harness's native implementation worker and isolated worktree ([Agent Harness](docs/agent-harness.md): a Codex App Worker with managed worktree, or a Claude Code background subagent with worktree isolation) only when the routing triggers above apply or the user explicitly delegates implementation.
- A durable execution control (Codex Goal or Claude Code task list) is for genuinely long-running, multi-step, or resumable implementation; one root control spans that outcome. Do not create one for ordinary direct work or non-implementation reasoning.
- Inspect the owning code, callers, siblings, tests, and generated/manual boundary before editing. Fix defects at the narrowest shared owner proved by the reproducer.
- Worker candidates pass the phase-owned [Scope Lock](docs/spec-first-workflow/phases/implementation-validation-closeout.md#scope-lock) before review and then follow [Monotonic Acceptance](docs/spec-first-workflow/phases/implementation-validation-closeout.md#monotonic-acceptance) plus its [Diagnostic Gate](docs/spec-first-workflow/phases/implementation-validation-closeout.md#diagnostic-gate). A correction finding identifies a candidate-caused regression, a concrete violation of an accepted criterion or repository-owned invariant, or proof missing from an accepted claim; other defects remain observations.
- Reuse successful proof only while the relevant content, environment or preconditions, claim scope, provenance, and risk surface are unchanged. Use a commit/tree identity only when proof crosses a checkout or integration boundary.

## Validation Matrix

Use the smallest matching check:

| Changed surface | Default proof |
| --- | --- |
| Docs or instructions | `git diff --check` |
| Local Go behavior | Focused package/test proof; changed-code lint when useful |
| Concurrency/lifecycle | Focused behavior plus race/liveness proof |
| Performance claim | The matching [benchmark level](docs/benchmarking.md), equivalent workload/testbed evidence, and independent correctness proof |
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
- [docs/agent-harness.md](docs/agent-harness.md) owns harness detection and the mapping from workflow concepts to native Codex App and Claude Code controls: durable execution controls, workers, subagent lanes, model selection, and reasoning effort.
- [Skill authoring](docs/skill-authoring.md) owns the lean behavioral-adapter contract.
- `docs/spec-first-workflow.md` owns routing and movement.
- Keep phase-specific method in `docs/spec-first-workflow/phases/`.
- `shared/artifact-model.md` owns persistence; `shared/subagents-and-handoff.md` owns built-in subagent delegation, triggered non-implementation review independence, convergence, and handoff.
- Skills provide methods; they do not create work or override accepted decisions.
- Task-local artifacts own accepted task decisions. Runtime and generated authorities named there win over derived prose.
- [Prompt Maintenance](docs/spec-first-workflow.md#prompt-maintenance) owns instruction phrasing, deduplication, and behavior-evaluation boundaries.
