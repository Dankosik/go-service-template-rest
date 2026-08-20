# Research Branches

Load only the branch selected by [Research](research.md#conditional-methods).

## Current-State Or Semantic Baseline

When current state can change accepted behavior or a later decision, establish the smallest decision-relevant baseline across every current surface that can independently change the answer, including as applicable repository authority, generated-source authority and artifact drift, observed runtime behavior, caller or operator evidence, persisted or deployed state, external consumers, mixed-version constraints, and the first unsupported edge. Record a required surface that cannot be observed as an unknown with its decision effect. Trace an affected value or contract from origin through persistence, transport, consumer, and derived output only as far as the decision requires; preserve grain or cardinality, identifiers, units, absence or error states, version authority, and lossy transformations. Matching names, historical intent, or receiver capability do not prove equivalent semantics or runtime use. Inspect ADRs, issues, incidents, or runbooks only when replacing unexplained brownfield behavior and only for a still-current constraint; history alone is not a current contract.

## Current External Contract

For an external platform, unfamiliar mechanism, new infrastructure/dependency, or non-trivial design choice, search current web and repository sources before design. Verify each decision-changing contract or behavior claim against its current primary authority: official provider documentation or contract, the applicable standard, or maintainer source. Resolve version and applicability explicitly. When practical fit, integration behavior, operability, or failure modes can change the decision, complete contract authority with applicable implementation evidence (maintainer code or credible real implementations) and credible engineering or operational evidence; record local applicability, context differences, and unavailable evidence as exact gaps.

When a decision depends on Go, toolchain, standard-library, module, runtime, or
dependency behavior, establish the current repository Go version and consult
current authoritative sources for only that decision. Do not expand this into
a full Go review.

## Solution Discovery Evidence

When the way to realize a behavior remains unresolved, do not anchor discovery on a user-proposed pattern, product, or implementation. Derive vendor-neutral search terms from the required behavior, invariants, workload, failure and recovery model, quality scenarios, constraints, and unacceptable trade-offs. Use pattern catalogs and reference architectures to expand vocabulary, not as proof of local production fit. Name the responsibility or decision slot each candidate occupies and classify it as a substitute, prerequisite, complement, or defense-in-depth mechanism. Characterize candidate evidence only to expose decision implications among substitutes at the same live decision level. Research may eliminate a candidate contradicted by authority or infeasible under a hard constraint; it does not compose, rank, or select the target architecture. System / Integration Design owns deriving task-specific drivers, constructing viable target-state mechanisms from this evidence, comparing them, and selecting one architecture. Record separately any product, library, managed service, transport, or topology roles used to realize the candidate.

For a live solution choice, scan only the relevant rungs: existing repository or organization reuse, Go stdlib, native database/broker/platform/provider capability, already approved and operated infrastructure, managed service, mature maintained OSS, and custom implementation. Do not force every rung or a fixed candidate count. Tutorials, examples, and reference architectures may demonstrate an integration shape, but do not alone establish production fit. For each surviving substitute, collect only applicable evidence that can change local approval: accountable owner, support, and on-call; availability, entitlement, region, quota, and SLA; compatibility, boundary fit, and provisioning; security guidance, advisories, unsafe defaults, and data custody; pricing unit and material cost at the accepted workload; lifecycle, deprecation, and upgrade policy; adoption and migration feasibility; and portability, exit, and failback limitations. For external code also require current evidence for maintenance/releases, license, security or vulnerability posture, API stability, transitive cost, domain adoption, and repository/boundary fit. If required evidence is unavailable, block or carry the exact proof gap.

Stop only when further searches by problem, decision force, failure mode, and known alias are unlikely to reveal a materially different viable candidate at the live decision level; source depth alone does not establish candidate-space saturation. Test saturation against every candidate already surfaced, including the ones already eliminated, rather than against the surviving set. A candidate that reappears in a later sweep after being rejected on evidence is not a new candidate, and re-characterizing it is how a sweep stops converging.

## Empirical Claim Or Probe

For empirical or runtime evidence, record the version or configuration when relevant, method or command, environment, workload or sample, timestamp, result, and known limits. When a decision depends on a performance, capacity, reliability, cost, quality, or version-specific empirical claim, also establish a comparable current baseline and representative workload or data envelope, including only distributions that can change the result; do not infer causality from a single snapshot or profile. If authoritative sources cannot resolve such a claim or establish its applicability to the current version, configuration, workload, or data envelope, and an authorized safe representative surface is available, run the smallest reversible or read-only probe that can discriminate the live implications; otherwise carry the exact blocker or later proof obligation.

## Conflict Or Freshness

When decision-changing research depends on conflicting sources, freshness-sensitive external behavior, or an approval-critical claim for a hard-to-reverse choice, obtain an independent read-only semantic challenge of the synthesis before specification or design consumes it. Evidence gathering alone cannot issue that verdict. If authoritative evidence cannot resolve a material conflict, block or return it to the evidence owner instead of choosing by confidence.

For freshness-sensitive evidence, record `valid as of` plus an objective refresh trigger or earliest downstream checkpoint.

## Downstream Input Closure

When research identifies a concrete external input required downstream, record
its owner, authoritative source, required shape, availability, and earliest
required checkpoint without inventing later design inputs. When an implication
requires proof, name the current observable or proving surface and setup
availability/derivability, or the missing-proof owner; Test Design still owns
scenario and proof-level selection.
