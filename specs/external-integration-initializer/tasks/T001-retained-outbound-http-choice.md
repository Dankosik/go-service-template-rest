# T001 — Retained outbound HTTP choice

Outcome:
`make template-init` accepts exactly `OUTBOUND_HTTP=none|bounded`, records and readbacks the exact choice in `template.lock`, retains the existing bounded HTTP owner when selected or still required by another retained capability, and removes it only when no selected capability consumes it, without changing any existing profile result.

Consumes:
- [`../test-design-transition.md`](../test-design-transition.md) SHA-256 `8c4afa98856011370f897cdb5bcee903d985104d2d39cb2148e8b44ffd4dffc3` and its fixed inputs.
- [`../design/overview.md`](../design/overview.md) SHA-256 `ad02cc02cd79dae097850eb241cb8d0f04ce8ee399fc5b2882a68b0255d3c2ac`, especially Retained outbound HTTP selection and its `template-init-check.sh` cross-product proof owner.
- Accepted repository baseline `8967a4ac06d4fce0515703b15ffa5db35e5378ae`; overlapping dirty selector, HTTP-client, and harness edits are excluded from candidate authority.

Provides:
- The accepted `outbound_http` choice, lock identity, and existing bounded HTTP package retention surface consumed by T002.
- A green template-initialization baseline for every existing capability combination.

Boundary:
Change only the template initializer's outbound HTTP choice, lock write/readback, dependency-aware package retention/removal, concise public option documentation, and their existing template-init proof. Reuse the current `internal/infra/httpclient`; do not change its API or behavior here. Do not add an integration command, adapter, config section, generator, record, OAuth migration, provider behavior, dependency, or external action.

Mutable owners:
- Template initialization capability selection, `template.lock` write/readback, and bounded HTTP retention/removal
- Template-init capability cross-product proof and public option documentation

Exclusive locks:
- Template profile selection/removal and `template.lock` contract

Accept when:
- Claim: Omitted and explicit `none` are equivalent; `bounded` retains the accepted HTTP owner; empty/unknown values reject without changing the fixture; every existing capability combination retains or removes shared HTTP ownership only when its dependency graph requires it.
- Focused check: `make template-init-check`
- Observable: The command exits zero with non-empty outbound-HTTP cross-product coverage; each generated fixture has the exact `outbound_http` lock value and expected HTTP-owner presence/absence; invalid choices leave byte-identical fixtures; existing profile checks remain green.

Reopen if:
Reopen Technical Design only if the accepted template initializer cannot expose the exact choice, lock field, or dependency-aware retention through its current owner. Reopen Planning only if this result cannot land green and be consumed by T002 without unfinished companion work. Reopen Specification only under the fixed transition's explicit behavior/authority conditions.
