# Test Design transition

status: superseded
superseded_by: 2026-08-23 named-only OAuth maintainability repair
owner: Test Design
result: `specs/external-integration-initializer/test-plan.md` SHA-256 `4e5d409a6cc3f153d29740163817b2a01ebf64e67aca8a8dc4ac90259ca499d3`
review: `specs/external-integration-initializer/test-design-review.md` SHA-256 `ee8fb27a050fd9cb537c95f33811b9c6225f4671ff6adb2dd7fdb719299d742b` — fresh independent Test Design Review `PASS` after one bounded delta recheck
movement_evidence: The fixed 25-row Test Plan V1 reconciles in both directions with all eight accepted proof expectations. Every non-residual row names a wrong observable, controlled trigger, independent oracle, proving boundary, runnable command or exact procedure, mandatory input, proof owner, and smallest reopen owner. The initial independent review exposed one silent ignored-path access false-pass; the bounded Test Design repair added mandatory Linux file-syscall proof, structural helper containment, and silent-open/follow mutants, and the same reviewer returned PASS on the current exact candidate. No actual `.env` was inspected, no behavioral proof was claimed, and no Planning, Implementation, provider, network, deployment, credential, or other external effect was performed.
reopen_owner: Test Design
next_owner: Planning

Reopen Test Design for a false-pass, nondeterministic control, infeasible
proving layer, missing mandatory input, or command that does not execute its
named row. Reopen only the affected Technical Design owner when a reviewed
mechanism, placement, or proving surface cannot realize a row. Reopen
Specification only if the fixed fail-closed behavior, developer or operator
custody boundary, byte-restoration rules, or initial/refresh consequences
cannot be preserved without changing accepted behavior or authority.

Planning is named only as the next owner. It remains unopened; Implementation
and every external effect remain unauthorized by this receipt.
