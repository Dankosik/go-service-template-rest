# AGENTS.md

OpenAPI-first Go service template with safe runtime defaults, optional
persistence profiles, observability, agent workflows, and CI.

## Authority

Inspection and reporting requests are read-only. Change, build, and fix
authorize in-scope local edits and non-destructive validation.

External writes, destructive actions, purchases, sensitive-data handling,
material scope expansion, and irreversible effects require matching authority.
Never place raw secrets in prompts, artifacts, logs, or handoffs. Respect
explicit read-only, docs-only, research-only, and named-phase boundaries.

Skills and durable controls provide methods; they neither create work nor expand
request authority. Task-local artifacts own accepted decisions, while runtime
and generated-source authorities named by those artifacts remain canonical.

### Decision Ownership

The agent owns technical decisions, routing, proof, and rollout within the
accepted outcome. The user owns business meaning, otherwise-unowned policy,
priority and deadline, money, legal commitments, and irreversible external
effects.

Resolve ordinary uncertainty from repository evidence. Ask only when materially
different outcomes remain and no bounded assumption keeps the work honest;
otherwise state the assumption and its reopen condition.

## Engineering

Reuse the current owner, repository pattern, standard library, and installed
dependencies before adding machinery. Match the surrounding code's naming,
comment density, idiom, and responsibility boundaries. Preserve unrelated work
and generated/manual authority.

## Work Selection And Loading

A Markdown link names an owner; it does not load it. Read the current owner
immediately before its first governed action or claim, and re-evaluate only when
evidence changes phase, risk, ownership, proof, or harness control.

Use [Direct Work](docs/spec-first-workflow/direct-work.md) for a clear, local,
reversible, single-owner outcome with bounded proof and no unresolved protected
decision. Otherwise read the [workflow router](docs/spec-first-workflow.md) and
load only the owner it selects.

| Trigger | Owner |
| --- | --- |
| Authorized external, costly, sensitive, destructive, or irreversible action | [External Effects](docs/spec-first-workflow/shared/external-effects.md) |
| Accepted work first enters another checkout | [Repository Boundaries](docs/spec-first-workflow/shared/repository-boundaries.md) |
| Repository boundary or generated-source ownership changes, or a new contract capability, integration adapter or outbound target, persistence or durable flow, durable schema, or executable entry point is added | [Repository Architecture](docs/repo-architecture.md) |
| Proof must be selected | [Validation Routing](docs/validation-routing.md) |
| A prompt for another agent, session, phase, or native entry skill must be written | [Prompt Composition](docs/prompt-composition.md) |
| Instructions, tools, roles, or skills change | [Prompt Maintenance](docs/prompt-maintenance.md); also [Skill Authoring](docs/skill-authoring.md) for skills |
| A durable control, carrier, model, or effort must be chosen or operated | [Agent Harness](docs/agent-harness.md) |

## Go Change Surface

For Go changes, apply only the skills whose descriptions match a pressure in
the changed surface; each selected skill owns its method and reopen condition.
