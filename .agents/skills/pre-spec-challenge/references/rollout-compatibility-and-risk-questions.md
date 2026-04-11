# Rollout Compatibility And Risk Questions

## When To Load
Load this when the candidate synthesis mentions rollout, migration, feature flags, canary, backward compatibility, mixed versions, destructive state changes, data backfill, or rollback. Do not use it to force rollout ceremony onto tiny reversible local work.

## Strong Vs Weak Questions
| Strong | Weak |
| --- | --- |
| "Can old and new binaries read the same persisted state during the rollout, and what breaks if rollback happens after new writes?" | "Is rollback safe?" |
| "Which metric or guardrail would stop the canary before broad exposure, and is the sample representative enough to catch the failure mode?" | "Should we canary?" |
| "If the feature flag is turned off after partial exposure, which data or async side effects remain and who cleans them up?" | "Can we use flags?" |
| "Does the migration require expand-contract ordering, or can the candidate plan deploy code and schema together safely?" | "How do we migrate?" |

## Blocker Classifications
- `blocks_planning`: rollback, migration order, or mixed-version compatibility can change implementation phases or require an expand-contract path.
- `blocks_specific_domain`: only delivery, data, or reliability evidence is missing, and the candidate path is otherwise stable.
- `non_blocking`: rollout detail is useful but the change is local, reversible, and validation can prove it before release.

## Next-Action Examples
- `answer`: "Use existing rollout policy or prior migration pattern if the repository already has a compatible precedent."
- `re-research`: "Reopen delivery/reliability research when guardrail metrics, rollback trigger, or canary population are not evidenced."
- `ask_user`: "Ask only if rollout cohort or launch timing is a business choice."
- `defer`: "Defer flag cleanup naming if the temporary flag owner and removal trigger are already explicit."
- `accept_risk`: "Accept no canary only for low-blast-radius work with a named fallback and fresh validation path."

## Exa Sources
- [Martin Fowler: Canary Release](https://martinfowler.com/bliki/CanaryRelease.html) for limiting production risk with gradual exposure and rollback by rerouting.
- [Martin Fowler: Feature Toggles](https://martinfowler.com/articles/feature-toggles.html) for separating deployment from release and classifying toggle lifetimes.
- [Google Cloud: Reliable releases and rollbacks](https://cloudplatform.googleblog.com/2017/03/reliable-releases-and-rollbacks-CRE-life-lessons.html) for treating rollback readiness as part of release safety.
- [ACM Queue: Canary Analysis Service](https://queue.acm.org/detail.cfm?id=3194655) for representative canary populations, clear metrics, and pass/fail evaluation.
