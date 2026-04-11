# Falsifying Candidate Assumptions

## When To Load
Load this when candidate synthesis depends on convenience claims like "clients will not retry," "TTL cleanup is enough," "operators can fix this manually," "UUID secrecy is sufficient," or "v1 can ignore the edge case." Use it to produce questions, not counter-designs.

## Strong Vs Weak Questions
| Strong | Weak |
| --- | --- |
| "What breaks if the client retries after a timeout and the first request commits after the response is lost?" | "What about retries?" |
| "If TTL cleanup lags by 24 hours, which user-visible or operator-visible state becomes wrong?" | "Is TTL safe?" |
| "Which metric or repository fact would falsify the claim that current traffic volume is too small to matter?" | "Do we have enough scale?" |
| "If the manual workaround is skipped during an incident, what invariant is violated?" | "Can ops handle it?" |

## Blocker Classifications
- `blocks_planning`: the assumption can invalidate the chosen path, change data shape, or require a different API/retry contract.
- `blocks_specific_domain`: the assumption only needs targeted security, reliability, data, or API evidence before planning can continue.
- `non_blocking`: the assumption is real but isolated, reversible, and can be guarded by validation or an explicit accepted risk.

## Next-Action Examples
- `answer`: "Use existing trace or test evidence if it already proves duplicate requests are rejected."
- `re-research`: "Ask a reliability lane to inspect retry and timeout behavior if no local evidence proves idempotency."
- `ask_user`: "Ask only if the risk turns on business appetite, such as accepting delayed cleanup visible to admins."
- `defer`: "Defer a rare, reversible operator UI inconvenience to downstream design if it cannot change implementation order."
- `accept_risk`: "Accept a low-frequency edge condition only when the plan names the blast radius and validation check."

## Exa Sources
- [ProductTalk: Assumption Testing](https://www.producttalk.org/2023/10/assumption-testing/) for isolating single assumptions and focusing on the ones that could torpedo the idea or cause harm.
- [Gary Klein: Premortem](https://www.gary-klein.com/premortem) for starting from plausible failure rather than optimism.
- [HBR: Performing a Project Premortem](https://hbr.org/2007/09/performing-a-project-premortem) for making dissent safe before planning hardens.
