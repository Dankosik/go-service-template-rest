# Railway Full Infrastructure Technical Design Phase

Phase: technical-design
Status: complete
Pass type: reopen/repair
Date: 2026-06-02
Owner: orchestrator

## Design Outcome

The stale app-only design bundle was replaced with a review-ready full
production infrastructure design bundle for the approved `spec.md`.

This phase produced design context only. It did not run technical design
review, create or approve `tasks.md`, implement code, deploy, or mutate Railway
resources.

## Inputs Used

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/technical-design-session/SKILL.md`
- `.agents/skills/go-design-spec/SKILL.md`
- `.agents/skills/go-design-spec/references/design-bundle-assembly.md`
- `.agents/skills/technical-design-session/references/workflow-plan-technical-design-updates.md`
- `docs/repo-architecture.md`
- `workflow-plan.md`
- `workflow-plans/specification.md`
- approved `spec.md`
- all research notes under `research/`
- `railway.toml`
- `docs/railway-deployment-profile.md`
- `env/config/default.yaml`
- `env/.env.example`
- `docs/configuration-source-policy.md`
- `build/docker/Dockerfile`
- `docs/build-test-and-development-commands.md`
- current app-only `design/`, `test-plan.md`, `rollout.md`, and `tasks.md`
  as historical baseline only
- read-only code inspection of billing-service config, service-auth,
  worker/runtime, Docker, and OpenAPI route scopes
- read-only sibling `gonka-proxy` inspection for current provider-contract
  limits
- read-only Railway template lookup for Kafka-compatible broker candidates

## Artifact Status

| Artifact | Status | Notes |
| --- | --- | --- |
| `design/overview.md` | review-ready | Replaced app-only entrypoint with full infrastructure design and review handoff. |
| `design/component-map.md` | review-ready | Covers app, Postgres, backups/PITR, broker, worker, image, auth, proxy, and validation components. |
| `design/sequence.md` | review-ready | Defines dependency-gated order, failure handling, rollback, and fail-closed behavior. |
| `design/ownership-map.md` | review-ready | Names source-of-truth owners and rejected non-owners for money authority. |
| `design/data-model.md` | review-ready | Triggered by dedicated Postgres, migrations, restore, reconciliation, and rollback data semantics. |
| `design/dependency-graph.md` | review-ready | Triggered by app/worker/Postgres/broker/proxy dependency shape and pre-mutation gates. |
| `design/contracts/service-auth-and-broker.md` | review-ready | Triggered by RS256/JWKS, route scopes, proxy handoff, broker topics, producer identity, and lag proof. |
| `test-plan.md` | review-ready | Triggered by multi-layer repo/Railway/database/broker/worker/auth/proxy/rollback validation. |
| `rollout.md` | review-ready | Triggered by stateful deployment, restore, mixed-version, worker drain, authority, rollback, and failback choreography. |
| `tasks.md` | historical app-only only | Not changed; not a full rollout implementation handoff. |
| `workflow-plans/technical-design-review.md` | stale app-only PASS | Historical review only. A fresh review is mandatory for this packet. |

## Design Decisions Closed In This Phase

- Full rollout uses split design artifacts plus triggered `test-plan.md` and
  `rollout.md`.
- Dedicated `billing-service-postgres` remains the database target.
- `billing-service-kafka` remains the broker service target.
- Railway template code `kafka` is selected as the current Kafka-compatible
  candidate, with a hard pre-mutation read-back/reopen gate because no verified
  Kafka template was found.
- The app and worker share the canonical Dockerfile lineage, which must be
  repaired to include `/billing-worker`.
- `billing-worker` stays private with no public HTTP health surface; readiness
  proof is through non-zero failure, bounded startup/task evidence, broker/db
  read-backs, lag/backlog buckets, admission-control freshness, and shutdown
  proof.
- App paid readiness requires Postgres and broker readiness, RS256/JWKS service
  auth, private networking, worker readiness, and proxy proof; app health alone
  is insufficient.
- `gonka-proxy` is a clean provider-contract prerequisite. Current dirty draft
  evidence is not approved.
- Public billing ingress and public `/metrics` remain rejected by default.

## Blockers

No blockers remain for entering technical design review.

Production paid readiness and Railway mutation remain blocked until later
phases produce a fresh review verdict, approved task ledger, and secret-free
proof.

## Reopen Conditions

Reopen specification if:

- no private persistent Kafka-compatible Railway broker can meet topic,
  retention, lag, and read-back requirements;
- a clean `gonka-proxy` provider contract cannot meet RS256/JWKS, scope,
  private URL, producer, child-debit lineage, and no-fallback requirements;
- public billing ingress or public `/metrics` becomes required;
- source topology read-back falsifies the approved repo/branch/root/Dockerfile
  or deployment-policy owner.

Reopen technical design if:

- the selected Kafka candidate must change while preserving the spec;
- worker readiness needs a private HTTP health surface instead of the selected
  non-HTTP evidence surface;
- app/worker resource or replica sizing changes materially;
- restore/reconciliation, broker lag, proxy proof, or rollback sequence needs a
  different design without changing the approved spec.

## Parallel Follow-Up

Technical design review may inspect design artifacts, repo code, sibling proxy
evidence, and Railway template evidence in parallel as read-only checks.

No mutation, planning, or task-ledger work may start until review finishes.

## Stop Rule

Stop at the technical design boundary.

Do not:

- run technical design review in this session;
- create, edit, or approve `tasks.md`;
- implement code or tests;
- mutate Railway resources;
- expose secrets or variable values.

## Next Action

Next action: mandatory technical design review.

Expected output of next phase: a distinct read-only technical design review
record identifying the reviewed packet, classifying findings, and returning a
gate verdict of `PASS`, eligible `CONCERNS`, or `FAIL`. Planning may not start
until that verdict exists and has no unresolved planning blockers.
