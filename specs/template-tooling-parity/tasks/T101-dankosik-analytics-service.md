# T101 — migrate Dankosik/analytics-service

Outcome:
The canonical `Dankosik/analytics-service` repository adopts the accepted T001 portable tooling without losing repository-owned policy or guessing ambiguous profile state, and returns one independently valid accepted candidate or bounded blocked receipt.

Consumes:
- Accepted T001 portable tooling baseline at `493e61ae6abc311df89a58d7cba64f407311ff41`.
- Current canonical `Dankosik/analytics-service` repository instructions, `origin/main`, profile evidence, and existing local-owner boundaries.

Provides:
- One reviewed consumer candidate or exact blocked receipt for `Dankosik/analytics-service`.

Boundary:
Mutate only `Dankosik/analytics-service` template-owned paths, generated views, mechanically required identity metadata, and the smallest service-owned Make extraction or explicit compatibility shim. Preserve unrelated work and duplicate checkouts. Do not push, open a consumer pull request, deploy, change required checks, acquire credentials, or weaken repository-owned policy.

Mutable owners:
- `Dankosik/analytics-service` canonical repository template-adoption surface.

Exclusive locks:
- `Dankosik/analytics-service` canonical Git identity and template-owned paths.

Accept when:
- Claim: `Dankosik/analytics-service` retains repository-owned commands and policy, records only mechanically proved profile choices, adopts the T001 surface, and reaches one independently reviewable accepted candidate or exact blocker.
- Focused check: run the accepted T001 `scripts/template-sync.sh --check` against the canonical checkout.
- Integrated check: run `make verify` in the canonical checkout on the fixed consumer candidate.
- Observable: repeat sync reports zero drift and focused/integrated proof passes, or one bounded blocked receipt names the unresolved owner and candidate identity.

Reopen if:
T001 changes, the canonical Git identity or profile evidence becomes ambiguous, or acceptance requires a behavior, policy, deployment, or authority decision outside template adoption.

