---
name: thermo-nuclear-code-quality-review
description: "Thermo-nuclear review: Use only on an explicit ask for a thermo-nuclear, thermonuclear, harsh, or deep code-quality audit of one fixed candidate. Own read-only structural simplification and maintainability findings; Skip ordinary review and implementation."
metadata:
  invocation: user
  kind: workflow
disable-model-invocation: true
---

# Thermo-Nuclear Code Quality Review

Review one fixed candidate read-only. Be ambitious only where repository
evidence supports deleting or collapsing structure without changing accepted
behavior; do not edit the candidate.

Resolve the requested diff from repository context, then read its full changed
sources, direct callers, canonical owners, and accepted proof boundary. Do not
blend materially different candidates. Apply the shared [Review
contract](../../../docs/spec-first-workflow/shared/review.md).

Load only the matching method:

- added abstractions, layers, files, shims, or parallel paths: [`go-structural-quality`](../go-structural-quality/SKILL.md)
- opaque control flow, helpers, predicates, or deduplication: [`go-language-simplifier`](../go-language-simplifier/SKILL.md)
- package, source, dependency, or proof-placement ambiguity: [`go-implementation-ownership`](../go-implementation-ownership/SKILL.md)
- an independent Go contract, lifecycle, security, data, or performance pressure: the corresponding Go skill

For every affected responsibility and added structure, record its anchor,
current owner, deletion or collapse attempt, observables that must survive,
next realistic change cost, available proof, and disposition. A code-judo
finding survives only with a concrete smaller replacement and an accounted
behavior boundary. Preserve error identity, status, nil/empty meaning, mutation
authority, cleanup and side-effect order, and dependency direction when they
apply.

Treat file length, branch count, duplication, and sequential execution as
search signals, not findings. Do not prescribe extraction, a new abstraction,
parallelism, or atomicity without evidence that it reduces present ownership
or reader cost while preserving behavior.

Return findings first, ordered by outcome impact. Each finding names its
anchor, impact, [classification and verdict](../../../docs/spec-first-workflow/interfaces/review-result-v1.md),
smallest repair owner, evidence boundary, and compact replacement. Omit
cosmetic nits. If no material finding survives, say so explicitly.

Complete when every affected responsibility and added structure has a
disposition, then return one Review Result V1 for the fixed candidate.
