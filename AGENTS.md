# AGENTS.md

Repository-wide contract for reliable Go-service changes with the least workflow that can prove them.

## Engineering Judgment

- Use the narrowest current evidence that can prove or falsify the next claim.
- Reuse the current owner, repository pattern, standard library, and installed dependencies before adding machinery.
- Keep behavior, failures, cleanup, and proof at their narrowest owner. Prefer concrete types and explicit control flow. Remove replaced code and adjacent stale artifacts unless current compatibility requires them.
- Treat cancellation, deadlines, partial work, cleanup, shutdown, generated authority, and mutable ownership as first-class only when the change touches them.
- During iteration, use cached focused checks and keep reusable local dependencies running. Reserve uncached tests, race, coverage, full lint, rebuilds, and teardown for a triggered claim or publication evidence; do not clean caches as a speed technique.
- Never overlap broad Go or Docker gates on the same host. Continue with focused checks or wait for the active gate; preserve repository-owned aggregate and linter serialization even when `MAKEFLAGS` is inherited.
- Keep tool cache ownership aligned end to end: local scripts inherit canonical tool paths, and CI restores and saves those same paths. A repository-local override requires measured need and matching persistence proof.
- Let mandatory lint own `govet` for the current repository. Repeated unit, race, coverage, fuzz, flake, and OpenAPI test commands use [`go test -vet=off`](https://pkg.go.dev/cmd/go#hdr-Testing_flags); retain default vet for disposable generated profiles and integration-tag paths not covered by the current-tree lint run.
- Persist expensive scanner downloads between local runs. The Trivy container uses one shared named cache volume; remove it only for diagnosed corruption or an explicit disk-reclamation decision, never as routine cleanup.
- Concurrent tests synchronize on owned events, not wall-clock sleeps. Use [`testing/synctest`](https://pkg.go.dev/testing/synctest) when the behavior is time-driven; keep real-time deadlines only as outer failure diagnostics.
- CI change-scope routing is fail-closed: an empty, unknown, or mixed path set receives the full gate. Docs-only changes may skip runtime and dependency analysis, but repository integrity, secret scanning, and the stable aggregate remain active.
- Build a production runtime image once per local or CI aggregate and reuse the exact tag for migration and security proof. A second build is justified only when the build inputs or required artifact identity differ.
- Keep temporary Docker resources owned by their creator: use [`--rm`](https://docs.docker.com/reference/cli/docker/container/run/#rm) for throwaway containers, register [Compose `down -v --remove-orphans`](https://docs.docker.com/reference/cli/docker/compose/down/) before ephemeral work, and register [Testcontainers cleanup](https://golang.testcontainers.org/features/garbage_collector/#terminate-function) immediately after acquisition. A host janitor is recovery only; opt in solely with `codex.cleanup=auto`, `codex.data=ephemeral`, a non-empty `codex.owner`, and a Unix-seconds `codex.expires_at`. Never label persistent data as ephemeral or rely on age, an unused state, or host cleanup as primary lifecycle proof.
- Keep change-scoped and full-history secret proof distinct. Local/PR checks cover the reviewable worktree and every base-to-HEAD commit; main and release retain full history. Nightly does not repeat an unchanged commit's deterministic merge gates, and performance never authorizes suppressing findings.
- Add CPU or memory limits only from a representative peak measurement with headroom; a limit that creates throttling, swapping, OOMs, or flaky proof is not an optimization.
- For a performance claim, follow [Benchmarking](docs/benchmarking.md); it owns proof level, workload and budget, evidence, remote execution, and completion policy.

## Go Readability

For behaviorally equivalent Go choices, optimize for the next maintainer. Make
what the code does and why it does it apparent through names, ordinary control
flow, and comments that record non-obvious rationale, constraints, or public
contracts.

Resolve style trade-offs in this order: clarity, simplicity, concision,
maintainability, then local consistency. Prefer the least mechanism that
preserves the accepted behavior; shorter code wins only when it also reduces
what a reader must remember.

Use the [Google Go Style Guide](https://google.github.io/styleguide/go/guide.html)
for judgment priority and the relevant
[Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) for Go idioms.
Let formatters and linters own mechanical style, and keep style edits
change-scoped. Changed Go is readable when its normal path, error path,
ownership or lifetime, and non-obvious policy are clear without reconstructing
hidden modes or speculative abstractions.

## Collaboration

- Lead with the conclusion. Separate established facts, inferences, trade-offs, and proof gaps.
- Challenge a design with concrete consequences and a viable smaller alternative; when choices are comparable, prefer clearer ownership, failure signals, and recovery.

## Authority And Loading

- Explicit user, system, and developer instructions win.
- This file owns request authorization, the agent/user decision boundary, and repository-wide invariants.
- Skills provide methods; they neither create work nor override this contract, accepted decisions, or task-local decisions.
- [docs/spec-first-workflow.md](docs/spec-first-workflow.md) is the workflow router. Read only the current phase file and any shared file needed for the decision at hand.
- Task-local artifacts own accepted task decisions. Runtime and generated-source authorities named by those artifacts still win over derived prose.

## Authorization And Boundaries

- `answer`, `explain`, `review`, `diagnose`, and `plan` authorize inspection and reporting only. `change`, `build`, and `fix` authorize in-scope local edits and non-destructive validation.
- Confirm only before an irreversible external effect: an external write that another owner or the public can observe, a destructive action, or a purchase. Everything else — reads, in-scope edits, tests, routing, scope judgement, worker or lane dispatch, model or effort selection, artifact set, and every technical decision under [Decision Ownership](#decision-ownership) — is the agent's to decide and act on.
- An approved external-write or purchase envelope owns its cost, security, and proof bounds. Inside it, choose live-state parameters such as region, equivalent host or size, bounded retry route, and local or remote execution from current evidence; a rejected route is no longer valid. Reopen authorization only to exceed the envelope, weaken required security or proof, or change scope or behavior.
- Inspection and authorized in-scope edits may leave the assigned checkout. A `change`, `build`, or `fix` request authorizes required local edits in an available neighboring repository when it owns part of the accepted outcome; no separate task or confirmation is required solely because work crosses a repository boundary. Before editing, load the target repository's instructions, inspect its checkout and dirty state, preserve unrelated changes, and validate every changed repository. Treat the neighbor as an external blocker only when it is unavailable, read-only, outside the accepted outcome, or the required action needs authority the request did not grant.
- Respect explicit `read-only`, `docs-only`, `research only`, and named-phase boundaries.
- A durable execution control (a Codex Goal in the Codex App; `/goal` plus the task list in Claude Code — see [Agent Harness](docs/agent-harness.md)) is for implementation only. Do not create or continue one during intake, research, specification, technical design, test design, planning, or their review and repair loops, even when those phases edit repository workflow artifacts.
- For ordinary non-interactive shell calls, avoid shell startup: set `login: false` when supported; otherwise set `shell: "/bin/bash"`. Use a login or interactive shell only when the command materially depends on its initialization or zsh-only syntax.

## Decision Ownership

The agent owns every technical decision. Architecture, boundaries, mechanism, API and schema shape, error and failure semantics, package and file placement, dependency and tooling choice, naming, test strategy, proof level, rollout mechanics, and workflow routing are agent-owned: decide each from current evidence and record it with its reopen condition. A technical branch leaves this agent as a decision, never as a question or a menu.

The user owns only what cannot be derived from the repository, its tools, or a named external owner: the intended outcome and its business meaning, a business rule or policy value with no repository or external authority, priority and deadline, money, legal or contractual commitments, and any irreversible external effect.

Escalate a user-owned item only when all of these hold:

- the answer changes the accepted outcome, not the implementation;
- at least two options remain defensible on current evidence and none dominates on correctness, reversibility, ownership clarity, or cost;
- no bounded assumption stated with its reopen condition would keep the delivered work honest and useful if it turns out wrong.

When one option dominates, choose it and state the choice with its reopen condition. When a bounded assumption is honest, state it and continue without waiting. One escalation carries one question: the decision it changes, the defensible options, and the recommended answer. Uncertainty, desire for confidence, blast radius, and protected domains raise the proof bar, not the escalation bar.

### Proceeding

Proceeding is not a decision the user owns. Inside the current authorization, never ask whether to begin, continue, or widen inspection, analysis, diagnosis, or research; whether to open a read-only lane; or whether to take the next in-scope task, lane, wave, or phase. Take the action current evidence supports and report the result. A stated intention is not a result: state the next step in the same turn that takes it, and treat a step too vague to name as the missing decision rather than a plan. Exactly two questions may end a turn: the single user-owned escalation above, and the confirmation required before an irreversible external effect.

Resolve doubt by looking, not by asking. Repository inspection, current external sources, and additional read-only lanes are authorized by every request, so when more evidence could change or strengthen a conclusion, gather it before answering and stop only when another source is unlikely to change the decision. A bounded assumption with its reopen condition, a named blocker, and a progress or scope note are statements that carry the work forward; none of them waits for a reply.

## Routing

`docs/spec-first-workflow.md` owns path selection; choose the smallest path that can close the accepted outcome. Direct work — clear, local, reversible, one owner, bounded proof, and no unresolved protected-domain decision — requires no durable execution control, native worker, worktree, durable artifact, independent review, or workflow opt-out: the root edits the assigned checkout, self-reviews the bounded diff, and runs focused proof. Read the router before choosing any wider path.

Public contracts, persisted data, security, money, performance, concurrency/lifecycle, deployment, and cross-service ownership require explicit relevant decisions and proof. They do not automatically require every artifact, reviewer, worker, or full validation suite. When an accepted outcome spans multiple deployables, repositories, or managed dependencies, apply [System Release Closure](docs/spec-first-workflow/phases/system-integration-design.md#system-release-closure); cover the full affected deployment graph, or narrow the claim and name the external blocker.

### Required Spine

Structured and orchestrated work follows the workflow router's [Required Spine](docs/spec-first-workflow.md#required-spine), including its research, review, movement, and scoping-down rules.

## Task Contract

1. Reconstruct the accepted outcome from current repository facts before acting. Resolve every open decision from current evidence and state the bounded assumption you chose; [Decision Ownership](#decision-ownership) owns which decisions are yours and when to escalate.
2. State the outcome, non-obvious constraints or authority, matching proof, and stop condition. Omit inherited defaults, empty fields, and discretionary steps; prescribe an order or mechanism only when an accepted decision fixes it.
3. Match every readiness or completion claim to current evidence of equal scope. Report narrower or unavailable proof honestly and name the next useful check.

## Implementation And Evidence

- During implementation, follow the phase-owned [Acceptance Posture](docs/spec-first-workflow/phases/implementation-validation-closeout.md#acceptance-posture) and its selected execution branch. Implementation / Validation / Closeout owns contract closure, bounded review, claim-scoped proof, acceptance, and integration; skills and execution branches supply methods and carrier-specific mechanics only. A real blocker ends implementation and is reported.

## Validation Matrix

Use the smallest matching check:

| Changed surface | Default proof |
| --- | --- |
| Docs or instructions | `git diff --check` |
| Local Go behavior | Focused package/test proof; changed-code lint when useful |
| Concurrency/lifecycle | Focused behavior plus race/liveness proof |
| Performance claim | The matching [benchmark level](docs/benchmarking.md), equivalent workload/testbed evidence, and independent correctness proof |
| OpenAPI, sqlc, migration, generated source | Canonical generation/drift and affected runtime proof |
| Defect crossing a service, client, or managed-dependency boundary | Correlated evidence from each implicated side: what the caller emitted, what this service observed and returned, and what the next hop recorded for the same correlation id |
| Security, deployment, cross-service or release | The matching protected-domain and integrated proof |
| Publication, CI parity, or broad cross-cutting change | `check-full`, `ci-local`, `pr-check`, container, or security suites only when the claim needs them |

`make check` is a broad local baseline, not the default edit loop. Do not run service tests for docs-only work or broad suites merely because they exist.

## Go Change Surface

When work can change Go, classify only the triggered pressures: package owner,
import direction, composition, or exported surface; reader-visible control flow,
naming, helper shape, or test diagnostics; method sets, nil/zero, errors, or
context; resource or transaction lifetime; mutable ownership, aliasing,
concurrency, or lifecycle; hot-path work amplification, resource cost, or
capacity; canonical, generated, or hand-written authority; and repository-native
proof. Activate only the matching existing Go methods; untriggered categories
create no work. Close every triggered category with its phase or skill owner, or
name the owner and condition that must reopen it.

## Instruction Ownership

- Keep global rules here.
- Never record repository-specific content — service name, module path, deployment target, owner, or service-specific invariant — in a template-owned instruction path. `template-owned.paths` lists those paths and [Template Sync](docs/template-sync.md) mirrors them verbatim into every derived repository, so anything service-specific left there is overwritten on the next sync and meanwhile misleads every other repository. That document names the repository-owned records which receive such content instead.
- [docs/agent-harness.md](docs/agent-harness.md) owns harness detection and the mapping from workflow concepts to native Codex App and Claude Code controls: durable execution controls, workers, subagent lanes, model selection, and reasoning effort.
- Where an instruction describes an external tool's behavior, it is a summary of a vendor contract this repository does not own. Link the vendor page beside the claim. Read that page before relying on the claim: a summary that drops a load-bearing clause still reads as complete, so the gap surfaces as an invented workaround rather than as a missing fact. Never infer unstated external behavior from a summary.
- [Skill authoring](docs/skill-authoring.md) owns the lean behavioral-adapter contract.
- Keep phase-specific method in `docs/spec-first-workflow/phases/`.
- `shared/artifact-model.md` owns persistence; `shared/subagents-and-handoff.md` owns built-in subagent delegation, triggered non-implementation review independence, convergence, and handoff.
- [Prompt Maintenance](docs/spec-first-workflow.md#prompt-maintenance) owns instruction phrasing, deduplication, and behavior-evaluation boundaries.
