# Bootstrap Review Workflow Plan

Task: Review `/Users/daniil/Projects/Opensource/go-service-template-rest/cmd/service/internal/bootstrap` for Go idiomaticity, maintainability, readability, and design cleanliness.

Current phase: `review-phase-1`
Phase status: `in_progress`
Execution shape: `full orchestrated` because the user explicitly requested subagents and the target package spans bootstrap lifecycle, startup dependencies, network policy, probes, shutdown, and tests.
Session boundary reached: `no`
Ready for next session: `no`
Next session starts with: continue `review-phase-1` until findings are synthesized and reported.

Scope:
- Read-only review of `cmd/service/internal/bootstrap/**`.
- Include directly relevant repository context from `docs/repo-architecture.md` and `go.mod`.
- Focus on idiomatic Go, maintainability, readability, source-of-truth seams, and package design shape.

Non-goals:
- No code edits.
- No generated artifacts beyond this review workflow-control pair.
- No broad redesign of bootstrap or repository architecture.
- No review of unrelated packages except when needed to understand a bootstrap call boundary.

Artifact status:
- `spec.md`: not expected; standalone review request with no implementation decision record.
- `design/`: not expected; review only.
- `plan.md`: not expected; review only.
- `tasks.md`: not expected; review only.
- `test-plan.md`: not expected.
- `rollout.md`: not expected.
- `workflow-plans/review-phase-1.md`: active.

Workflow plan adequacy challenge:
- Required because work is agent-backed.
- Status: complete; no blocking task-specific handoff gaps surfaced.

Review lanes:
- Lane A: `quality-agent`, question: identify idiomatic Go issues with merge-risk impact in `cmd/service/internal/bootstrap`, skill: `go-idiomatic-review`.
- Lane B: `quality-agent`, question: identify readability and simplification issues that raise real maintenance risk in `cmd/service/internal/bootstrap`, skill: `go-language-simplifier-review`.
- Lane C: `architecture-agent`, question: identify design, boundary, source-of-truth, and accidental-complexity drift in `cmd/service/internal/bootstrap`, skill: `go-design-review`.

Fan-in rule:
- The orchestrator compares subagent claims against local code evidence before reporting.
- Findings that are only taste, duplicate claims, or lack concrete maintenance or correctness risk are dropped.
- Final review output leads with findings ordered by severity and includes file/line references.

Validation / evidence:
- Fresh local reads of the reviewed files are required.
- Run `go test ./cmd/service/internal/bootstrap` if findings depend on test compile behavior or if there are enough suspicious issues that a package-level test signal is useful.

Blockers:
- None known.

Stop rule:
- Stop after final synthesized review is reported to the user.
