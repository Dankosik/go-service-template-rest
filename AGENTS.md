# AGENTS.md

Repository-wide contract for an OpenAPI-first Go service template with safe
runtime defaults, optional PostgreSQL and agent-workflow profiles,
observability, and CI. Use the least workflow that can prove the requested
change.

## Engineering Judgment

- Reuse the current owner, repository pattern, standard library, and installed dependencies before adding machinery.
- For pure generic slice and map transformations, use a clear one-call standard-library operation first and then `github.com/samber/lo`; do not add local generic helpers or wrappers around `lo`. Keep domain policy, errors, lifecycle, concurrency, security, and transactions in explicit local Go.
- For build, test, CI, scanner, container, or secret-proof work, start at
  [Validation Routing](docs/build-test-and-development-commands.md#validation-routing)
  and load only the section selected by the changed surface or claim. That
  document owns iteration, cache, serialization, cleanup, image reuse, and gate
  policy. [Benchmarking](docs/benchmarking.md) owns performance proof, remote
  execution, and resource-limit decisions.

## Go Readability

Match the surrounding package's naming, comment density, and idiom. Prefer the
least mechanism that keeps normal flow, error flow, ownership, and lifetime
apparent; shorter code wins only when it reduces what a maintainer must infer.
Let formatters and linters own mechanical style and keep style edits
change-scoped.

Use comments for non-obvious rationale or constraints, plausible alternatives
that are wrong, external or generated contracts, required API documentation,
and machine-read directives. Prefer names, types, and ordinary control flow for
everything else; remove narration while already editing that code.

## Collaboration

- Lead with the conclusion. Separate established facts, inferences, trade-offs, and proof gaps.
- Name a task, acceptance unit, decision, or artifact by its postcondition title in anything a person reads; an identifier or path rides inside that name and never replaces it.

## Authority And Loading

- This file owns request authorization, the agent/user decision boundary, and repository-wide invariants.
- Skills provide methods; they neither create work nor override this contract, accepted decisions, or task-local decisions.
- A Markdown link names an owner; it does not load it. Read only the current
  owner immediately before its first governed action or claim, and re-evaluate
  that set only when evidence changes phase, risk, ownership, proof, or harness
  control.
- Before the first action that changes repository boundaries or generated-source
  ownership, or adds a contract capability, integration adapter or outbound
  target, persistence or durable flow, durable schema, or executable surface,
  read [Repository Architecture](docs/repo-architecture.md); it owns the
  repository extension seams.
- Before editing agent instructions, tool descriptions, or skill files, read [Prompt Maintenance](docs/prompt-maintenance.md); skill changes also load [Skill Authoring](docs/skill-authoring.md).
- Task-local artifacts own accepted task decisions. Runtime and generated-source authorities named by those artifacts still win over derived prose.

## Authorization And Boundaries

- `answer`, `explain`, `review`, `diagnose`, and `plan` authorize inspection and reporting only. `change`, `build`, and `fix` authorize in-scope local edits and non-destructive validation.
- Repository work adds no approval gate for reads, in-scope local edits, tests,
  routing, dispatch, harness controls, or technical decisions. External writes,
  destructive actions, purchases, sensitive-data handling, and material scope
  expansion follow the active harness policy and request authority.
- An approved external-write or purchase envelope owns its cost, security, and proof bounds. Inside it, choose live-state parameters such as region, equivalent host or size, bounded retry route, and local or remote execution from current evidence; a rejected route is no longer valid. Reopen authorization only to exceed the envelope, weaken required security or proof, or change scope or behavior.
- Cross-actor prompts, handoffs, artifacts, logs, and receipts carry only a secret locator or environment-variable name, never a raw secret. A credential observed outside its secret channel is exposed: stop using it, suspend its external-effect authority, and require rotation before reuse.
- Inspection and authorized in-scope edits may leave the assigned checkout. A `change`, `build`, or `fix` request authorizes required local edits in an available neighboring repository when it owns part of the accepted outcome; no separate task or confirmation is required solely because work crosses a repository boundary. Before editing, load the target repository's instructions, inspect its checkout and dirty state, preserve unrelated changes, and validate every changed repository. Treat the neighbor as an external blocker only when it is unavailable, read-only, outside the accepted outcome, or the required action needs authority the request did not grant.
- Respect explicit `read-only`, `docs-only`, `research only`, and named-phase boundaries.
- Durable controls follow the selected workflow and [Agent
  Harness](docs/agent-harness.md); they never expand request authorization.

### Decision Ownership

Resolve routine technical uncertainty from current evidence. The agent owns
technical decisions, proof, rollout, and routing; the user owns business
meaning, otherwise unowned policy, priority and deadline, money, legal or
contractual commitments, and irreversible external effects. State a better
outside-scope approach without silently changing the request.

Ask one question only when at least two defensible interpretations materially
change the outcome and no bounded assumption keeps the work honest; recommend
one answer. Otherwise state the assumption and reopen condition, then proceed.
Risk raises the proof bar, not the escalation bar.

### Proceeding

Within the current authorization and macro phase, take each named in-scope step
in the same turn until the [workflow router's macro-phase
boundary](docs/spec-first-workflow.md#phase-movement) requires a handoff. A
failed lookup or unavailable control changes the evidence route, not the
outcome: inspect authoritative alternatives, avoid unchanged retries, and
obtain readable facts without delegating the check to the user.

## Routing

### Direct Work

Use direct work for a clear, local, reversible outcome with one owner, bounded
proof, and no unresolved protected-domain decision. Inspect the current diff
and callers, change the narrowest causal owner, preserve unrelated work and
generated/manual authority, self-review the bounded diff, and run the smallest
proof that would fail if the outcome were absent or wrong. Direct work uses no
durable control, Worker, worktree, artifact, or independent review.

Read the [workflow router](docs/spec-first-workflow.md) before choosing a wider
path; its [Required Spine](docs/spec-first-workflow.md#required-spine) owns
structured and orchestrated work.

Public contracts, persisted data, security, money, performance, concurrency/lifecycle, deployment, and cross-service ownership require explicit relevant decisions and proof. Select only the artifacts, reviewers, workers, and validation gates justified by those decisions and proof. When an accepted outcome spans multiple deployables, repositories, or managed dependencies, apply [System Release Closure](docs/spec-first-workflow/phases/system-integration-design.md#system-release-closure); cover the full affected deployment graph, or narrow the claim and name the external blocker.

Load [Implementation / Validation /
Closeout](docs/spec-first-workflow/phases/implementation-validation-closeout.md)
only when structured unit closure, non-obvious validation, deployment or remote
proof, independent implementation review, integration, or blocked completion
triggers its contract.

## Task Contract

1. Reconstruct the accepted outcome from current repository facts before acting. Apply [Decision Ownership](#decision-ownership) to unresolved uncertainty and state each bounded assumption with its reopen condition.
2. State the outcome, non-obvious constraints or authority, matching proof, and stop condition. Omit inherited defaults, empty fields, and discretionary steps; prescribe an order or mechanism only when an accepted decision fixes it.
3. Match every readiness or completion claim to current evidence of equal scope.
   Run every faithful local obligation and isolate unsupported or target-only
   remainder. When its conditional boundary is active, the [Implementation
   Evidence Contract](docs/spec-first-workflow/phases/implementation-validation-closeout.md#evidence-contract)
   owns production-path proof, remote preflight, deployment boundaries, and
   unavailable proof. State every skipped or sampled surface beside the claim it
   narrows.

## Go Change Surface

Apply only the Go skills whose descriptions match a pressure present in the
changed surface. Each triggered skill owns its method; close it or name the
owner and evidence that must reopen it.
