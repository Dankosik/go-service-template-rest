# Reference Selector

Each row names a pressure where this repository's implemented controls or a
current standard overrides the obvious answer. State the expected behavior
change before loading, and load one reference by default.

Password storage, account recovery, cookie sessions, CSRF tokens, uploads, and
archive extraction have no reference here: this service authenticates bearer
tokens issued elsewhere and serves no browser documents, uploaded files, or
archives. A service that adds one decides it against
[`auth-access-control`](../../../../docs/universal-disciplines/auth-access-control/SKILL.md)
and current OWASP guidance; adding a reference back requires a decision it would
change.

Two neighbours own security questions that arrive through this skill. Deny-path
falsifiers, permission-model shape, tenant partitioning, and revocation windows
belong to `auth-access-control`. Inbound message authenticity — a webhook or
queue payload whose producer must be proved before a side effect, and the replay
window that follows — belongs to
[`reliable-messaging`](../../../../docs/universal-disciplines/reliable-messaging/SKILL.md)
and
[`durable-background-jobs`](../../../../docs/universal-disciplines/durable-background-jobs/SKILL.md),
with the identity carried as durable principal context rather than a live token.

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
