# Reference Selector

State the expected decision effect and load one matching reference by default.

| Pressure | Load | Required effect |
| --- | --- | --- |
| Bearer-token verification, an identity header, or what a handler reads about its caller | [verified-principal.md](verified-principal.md) | Derive identity from the verifier this repository already runs, instead of writing a second credential path or authorizing on a claim it never populates. |
| A new operation, a changed `security:`, or a surface becoming reachable by a new caller class | [contract-exposure.md](contract-exposure.md) | Justify an opt-out against the contract's default-deny requirement, instead of reading an unprotected operation as undecided. |
| Caller-influenced data reaching a decoder, SQL, a template, a subprocess, or a filesystem path | [untrusted-input.md](untrusted-input.md) | Bound the value where the interpreter's own defaults are lax, instead of trusting a decoded struct or a lexical path clean. |
| This service dialing a provider, delivering a webhook, or fetching a caller- or tenant-supplied URL | [outbound-egress.md](outbound-egress.md) | Reuse the pinned-authority client and its post-DNS gate, and treat per-request destinations as a new decision. |
| A caller able to scale work, repeat attempts, or drive cost | [abuse-and-cost-bounds.md](abuse-and-cost-bounds.md) | Name the identity a budget is charged to, instead of reading shedding and timeouts as fairness. |
| A secret, credential, personal field, or internal detail able to reach a response, log, trace, metric label, config file, or the repository | [secrets-and-disclosure.md](secrets-and-disclosure.md) | Name the sink and its readers and keep the repository's config and scan boundaries intact, instead of restating "do not log secrets". |

Record an absent identity provider, edge, secret store, queue, or third-party
guarantee as an assumption or a blocker rather than an implied control.
