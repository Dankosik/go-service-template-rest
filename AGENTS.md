# AGENTS.md

Repository-wide contract for reliable Go-service changes with the least workflow that can prove them.

## Engineering Judgment

- Use the narrowest current evidence that can prove or falsify the next claim.
- Reuse the current owner, repository pattern, standard library, and installed dependencies before adding machinery.
- Keep behavior, failures, cleanup, and proof at their narrowest owner. Prefer concrete types and explicit control flow.
- Treat cancellation, deadlines, partial work, cleanup, shutdown, generated authority, and mutable ownership as first-class only when the change touches them.
- During iteration, use cached focused checks and reusable dependencies. Reserve uncached tests, race, coverage, full lint, rebuilds, and teardown for a triggered claim or publication evidence.

## Authorization And Boundaries

- `answer`, `explain`, `review`, `diagnose`, and `plan` authorize inspection and reporting only. `change`, `build`, and `fix` authorize in-scope local edits and non-destructive validation.
- Ask before external writes, destructive actions, purchases, or material scope expansion. Do not ask before ordinary reads, in-scope edits, or tests.
- Respect explicit `read-only`, `docs-only`, `research only`, and named-phase boundaries.
- For ordinary non-interactive shell calls, set `login: false`; use a login shell only when it materially needs initialization.

## Routing

`docs/spec-first-workflow.md` owns path selection. Start with the smallest path that preserves correctness:

- **Direct:** the request is clear, local, reversible, has one owner, bounded proof, and no unresolved protected-domain decision. The root may edit the assigned checkout, self-review the bounded diff, and run focused proof. No Goal, App Worker, worktree, durable artifact, independent review, or workflow opt-out is required.
- **Structured:** persist only a decision, proof design, or ledger that another phase, actor, or later session needs. Use independent review only when the user requests it or a high-impact, hard-to-reverse, cross-owner, or weakly falsifiable decision needs it.
- **Orchestrated:** use a Goal, App Worker/worktree, durable coordination, or parallel waves only for real long-running, resumable, isolation, dirty-checkout, parallelism, separate-context, or coordination needs.

Public contracts, persisted data, security, money, concurrency/lifecycle, deployment, and cross-service ownership require explicit relevant decisions and proof. They do not automatically require every artifact, reviewer, worker, or full validation suite.

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

## Ownership

- `docs/spec-first-workflow.md` owns routing and movement.
- Phase files own unique decision methods; `shared/artifact-model.md` owns persistence; `shared/subagents-and-handoff.md` owns optional read-only lanes and handoff.
- Skills provide methods; they do not create work or override accepted decisions.
- Task-local artifacts own accepted task decisions. Runtime and generated authorities named there win over derived prose.
- Keep a rule in its narrowest owner; replace duplicates with links or delete them.
