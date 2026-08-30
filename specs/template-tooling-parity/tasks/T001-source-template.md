# T001 — publish the portable source-template tooling

Outcome:
The source template publishes one profile-independent Make, script, and pinned-tool surface through the existing sync engine while repository-owned policy and executable CI workflows remain outside verbatim adoption.

Consumes:
- Accepted `spec.md`, `design/overview.md`, and `test-plan.md`.

Provides:
- Accepted portable tooling baseline on template `main` at `493e61ae6abc311df89a58d7cba64f407311ff41`.
- The fixed input consumed independently by T101 through T129.

Boundary:
Own only the source template tooling, propagation manifest, sync engine, portable validation scripts, and pinned tools. Consumer migrations, consumer policy, pushes, deployments, credentials, and control-plane changes remain outside this unit.

Mutable owners:
- Source-template portable tooling and `template-owned.paths`.

Exclusive locks:
- Source-template propagation manifest and shared Make contract.

Accept when:
- Claim: a disposable initialized service retains local owners, adopts portable tooling byte-for-byte, resolves the pinned tools, refuses dirty or ambiguous adoption before writes, and repeats at zero drift.
- Focused check: `make template-owned-purity-check`
- Integrated check: `ALLOW_HEAVY=1 make verify`
- Observable: PR #256 and #257 exact-head required contexts succeed, PR #264 preserves classifier coverage, and template `main` contains the accepted surface.

Reopen if:
The accepted template baseline changes the propagation contract or a consumer proves that repository-owned policy can no longer survive adoption.

