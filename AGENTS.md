# AGENTS.md

Repository-wide contract for reliable Go-service changes with the least workflow that can prove them.

## Engineering Judgment

- Use the narrowest current evidence that can prove or falsify the next claim.
- Reuse the current owner, repository pattern, standard library, and installed dependencies before adding machinery.
- For pure generic slice and map transformations, use a clear one-call standard-library operation first and then `github.com/samber/lo`; do not add local generic helpers or wrappers around `lo`. Keep domain policy, errors, lifecycle, concurrency, security, and transactions in explicit local Go.
- Keep behavior, failures, cleanup, and proof at their narrowest owner. Prefer concrete types and explicit control flow. Remove replaced code and adjacent stale artifacts unless current compatibility requires them.
- Treat cancellation, deadlines, partial work, cleanup, shutdown, generated authority, and mutable ownership as first-class only when the change touches them.
- During iteration, use cached focused checks and keep reusable local dependencies and caches. Run uncached tests, race, coverage, full lint, rebuilds, and teardown only for a triggered claim or publication evidence; cache clearing belongs only to diagnosis, disk reclamation, or proof that requires a cold state.
- Run at most one broad Go or Docker gate on the host. While one runs, use focused checks or wait; preserve repository-owned aggregate and linter serialization even when `MAKEFLAGS` is inherited.
- Keep tool cache ownership aligned end to end: local scripts inherit canonical tool paths, and CI restores and saves those same paths. A repository-local override requires measured need and matching persistence proof.
- Let mandatory lint own `govet` for the current repository. Repeated unit, race, coverage, fuzz, flake, and OpenAPI test commands use [`go test -vet=off`](https://pkg.go.dev/cmd/go#hdr-Testing_flags); retain default vet for disposable generated profiles and integration-tag paths not covered by the current-tree lint run.
- Persist expensive scanner downloads between local runs. The Trivy container uses one shared named cache volume retained until diagnosed corruption or an explicit disk-reclamation decision requires removal.
- Concurrent tests synchronize on owned events. Use [`testing/synctest`](https://pkg.go.dev/testing/synctest) when the behavior is time-driven; keep real-time deadlines only as outer failure diagnostics.
- CI change-scope routing is fail-closed: an empty, unknown, or mixed path set receives the full gate. Docs-only changes may skip runtime and dependency analysis, but repository integrity, secret scanning, and the stable aggregate remain active.
- Build a production runtime image once per local or CI aggregate and reuse the exact tag for migration and security proof. A second build is justified only when the build inputs or required artifact identity differ.
- Keep temporary Docker resources owned by their creator: use [`--rm`](https://docs.docker.com/reference/cli/docker/container/run/#rm) for throwaway containers, register [Compose `down -v --remove-orphans`](https://docs.docker.com/reference/cli/docker/compose/down/) before ephemeral work, and register [Testcontainers cleanup](https://golang.testcontainers.org/features/garbage_collector/#terminate-function) immediately after acquisition. A host janitor is recovery only; opt in solely with `codex.cleanup=auto`, `codex.data=ephemeral`, a non-empty `codex.owner`, and a Unix-seconds `codex.expires_at`. Never label persistent data as ephemeral or rely on age, an unused state, or host cleanup as primary lifecycle proof.
- Keep change-scoped and full-history secret proof distinct. Local/PR checks cover the reviewable worktree and every base-to-HEAD commit; main and release retain full history. Nightly does not repeat an unchanged commit's deterministic merge gates, and performance never authorizes suppressing findings.
- Add CPU or memory limits only from a representative peak measurement with headroom; a limit that creates throttling, swapping, OOMs, or flaky proof is not an optimization.
- For a performance claim, follow [Benchmarking](docs/benchmarking.md); it owns proof level, workload and budget, evidence, remote execution, and completion policy.

## Go Readability

For behaviorally equivalent Go choices, optimize for the next maintainer. Make
what the code does apparent through names and ordinary control flow, and keep
what only a comment can carry under [Comments](#comments).

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

### Comments

Write a comment only where a competent reader of the final code would still
reach a wrong conclusion: a rationale or constraint the code cannot show, an
alternative that looks correct and is not, an external or generated contract, or
a machine-read directive. Anything a name, a type, or ordinary control flow can
carry is a rename or a restructure, not a comment.

A justified comment records that fact once, in the words needed to act on it.
Delete one that restates its identifier, its signature, or the next line;
narrates the normal path; repeats what another owner already states; argues for
code that is already accepted; or supplies background a reader does not need in
order to change this file. Deleting a comment that fails this test belongs to
whatever change is already editing that code, not to a separate cleanup.

## Collaboration

- Lead with the conclusion. Separate established facts, inferences, trade-offs, and proof gaps.
- Challenge a design with concrete consequences and a viable smaller alternative; when choices are comparable, prefer clearer ownership, failure signals, and recovery.
- Name a task, acceptance unit, decision, or artifact by its postcondition title in anything a person reads; an identifier or path rides inside that name and never replaces it.

## Authority And Loading

- System and developer instructions remain authoritative; explicit user requests override this repository file.
- This file owns request authorization, the agent/user decision boundary, and repository-wide invariants.
- Skills provide methods; they neither create work nor override this contract, accepted decisions, or task-local decisions.
- A Markdown link names an owner; it does not load that owner's instructions. Before the first phase-governed action, load the smallest current read set: every `change`, `build`, or `fix` loads [Implementation / Validation / Closeout](docs/spec-first-workflow/phases/implementation-validation-closeout.md); an open path or phase, or any structured/orchestrated route, first loads the [workflow router](docs/spec-first-workflow.md) and then its current phase. Load the router's conditional owner before its governed action.
- Before the first action that changes repository boundaries or generated-source
  ownership, or adds a contract capability, integration adapter or outbound
  target, persistence or durable flow, durable schema, or executable surface,
  read [Repository Architecture](docs/repo-architecture.md); it owns the
  repository extension seams.
- Before editing agent instructions, tool descriptions, or skill files, read [Prompt Maintenance](docs/spec-first-workflow.md#prompt-maintenance); skill changes also load [Skill Authoring](docs/skill-authoring.md).
- Instruction loading is a gate: read the matching owner before the first edit, durable-artifact mutation, native-control dispatch, or readiness/completion claim it governs. Re-evaluate the read set only when evidence changes the phase, risk, ownership, proof, or harness control; retain only the current phase and triggered conditional owners.
- Orchestrated Implementation roles and their loading, dispatch, lane, and return
  contracts are owned by the [Execution Role
  Tree](docs/spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree);
  a native carrier name never grants role authority.
- Task-local artifacts own accepted task decisions. Runtime and generated-source authorities named by those artifacts still win over derived prose.

## Authorization And Boundaries

- `answer`, `explain`, `review`, `diagnose`, and `plan` authorize inspection and reporting only. `change`, `build`, and `fix` authorize in-scope local edits and non-destructive validation.
- Confirm only before an irreversible external effect: an external write that another owner or the public can observe, a destructive action, or a purchase. Reads, in-scope edits, tests, routing, dispatch, harness controls, and technical decisions remain agent-owned.
- An approved external-write or purchase envelope owns its cost, security, and proof bounds. Inside it, choose live-state parameters such as region, equivalent host or size, bounded retry route, and local or remote execution from current evidence; a rejected route is no longer valid. Reopen authorization only to exceed the envelope, weaken required security or proof, or change scope or behavior.
- Cross-actor prompts, handoffs, artifacts, logs, and receipts carry only a secret locator or environment-variable name, never a raw secret. A credential observed outside its secret channel is exposed: stop using it, suspend its external-effect authority, and require rotation before reuse.
- Inspection and authorized in-scope edits may leave the assigned checkout. A `change`, `build`, or `fix` request authorizes required local edits in an available neighboring repository when it owns part of the accepted outcome; no separate task or confirmation is required solely because work crosses a repository boundary. Before editing, load the target repository's instructions, inspect its checkout and dirty state, preserve unrelated changes, and validate every changed repository. Treat the neighbor as an external blocker only when it is unavailable, read-only, outside the accepted outcome, or the required action needs authority the request did not grant.
- Respect explicit `read-only`, `docs-only`, `research only`, and named-phase boundaries.
- Use durable execution controls only from Implementation through Closeout;
  earlier macro phases use none. An explicitly requested Ledger Orchestrator may
  own reversible native routing while Implementation is suspended, but it never
  owns phase or unit decisions or expands authority for an irreversible effect,
  purchase, or missing business meaning. [Agent Harness](docs/agent-harness.md)
  and the [Execution Role
  Tree](docs/spec-first-workflow/phases/implementation-worker-execution.md#execution-role-tree)
  own the exact controls and protocol.
- Run ordinary non-interactive shell calls without shell startup: set `login: false` when supported; otherwise set `shell: "/bin/bash"`. Use a login or interactive shell only when the command materially depends on its initialization or zsh-only syntax.

## Decision Ownership

Complete the requested outcome at its intended scope. Resolve routine technical
uncertainty from the repository, tools, and current sources, and make ordinary
judgment calls yourself. If a better approach exists outside that scope, state
it briefly and continue; never silently narrow, widen, or transform the request.

The agent owns architecture, boundaries, mechanisms, contracts, errors,
placement, dependencies, naming, tests, proof, rollout, and workflow routing.
The user owns only intended outcome and business meaning, an otherwise unowned
business policy value, priority and deadline, money, legal or contractual
commitments, and irreversible external effects.

Ask one question only when different interpretations materially change the
accepted outcome, at least two remain defensible, and no bounded assumption
keeps the work honest and useful; recommend one answer. Otherwise state the
assumption and its reopen condition, then proceed. Risk raises the proof bar,
not the escalation bar.

### Proceeding

Within the current authorization and macro phase, take each named in-scope next
step in the same turn unless the [workflow router's macro-phase
boundary](docs/spec-first-workflow.md#phase-movement) requires a handoff and
stop. A failed lookup or unavailable first control changes the evidence route,
not the outcome: inspect authoritative alternatives, do not repeat unchanged
attempts, and do not ask the user to perform a read-only check an authorized
source can perform. [Bottom-Up Obstacle
Resolution](docs/spec-first-workflow/phases/implementation-worker-execution.md#bottom-up-obstacle-resolution)
owns orchestrated Implementation returns and blockers.

## Routing

`docs/spec-first-workflow.md` owns path selection; choose the smallest path that can close the accepted outcome. Direct work — clear, local, reversible, one owner, bounded proof, and no unresolved protected-domain decision — requires no durable execution control, native worker, worktree, durable artifact, independent review, or workflow opt-out: the root edits the assigned checkout, self-reviews the bounded diff, and runs focused proof. Read the router before choosing any wider path.

Public contracts, persisted data, security, money, performance, concurrency/lifecycle, deployment, and cross-service ownership require explicit relevant decisions and proof. Select only the artifacts, reviewers, workers, and validation gates justified by those decisions and proof. When an accepted outcome spans multiple deployables, repositories, or managed dependencies, apply [System Release Closure](docs/spec-first-workflow/phases/system-integration-design.md#system-release-closure); cover the full affected deployment graph, or narrow the claim and name the external blocker.

### Required Spine

Structured and orchestrated work follows the workflow router's [Required Spine](docs/spec-first-workflow.md#required-spine), including its research, review, movement, and scoping-down rules.

## Task Contract

1. Reconstruct the accepted outcome from current repository facts before acting. Resolve every open decision from current evidence and state the bounded assumption you chose; [Decision Ownership](#decision-ownership) owns which decisions are yours and when to escalate.
2. State the outcome, non-obvious constraints or authority, matching proof, and stop condition. Omit inherited defaults, empty fields, and discretionary steps; prescribe an order or mechanism only when an accepted decision fixes it.
3. Match every readiness or completion claim to current evidence of equal scope.
   Run every faithful local obligation and isolate unsupported or target-only
   remainder. [Implementation / Validation /
   Closeout](docs/spec-first-workflow/phases/implementation-validation-closeout.md)
   owns production-path proof, remote preflight, deployment boundaries, and the
   `implementation complete; verification incomplete` stop. State every skipped
   or sampled surface beside the claim it narrows.

## Implementation And Evidence

- [Implementation / Validation /
  Closeout](docs/spec-first-workflow/phases/implementation-validation-closeout.md)
  owns contract closure, one proof owner, bounded review, the Validation Matrix,
  claim-scoped proof, acceptance, integration, and blocker handling; skills and
  execution branches supply methods only.

## Go Change Surface

When work can change Go, classify only the triggered pressures: package owner,
import direction, composition, or exported surface; reader-visible control flow,
naming, helper shape, or test diagnostics; method sets, nil/zero, errors, or
context; resource or transaction lifetime; mutable ownership, aliasing,
concurrency, or lifecycle; algorithmic complexity, hot-path work amplification,
resource cost, or capacity; canonical, generated, or hand-written authority;
and repository-native proof. For a second present path, treat behavior as one
policy only when it
shares the same accepted authority, invariant, and policy-level failure
semantics; similar code shape alone is not evidence. Activate only the matching
existing Go methods; untriggered categories create no work. Close every
triggered category with its phase or skill owner, or name the owner and condition
that must reopen it.
