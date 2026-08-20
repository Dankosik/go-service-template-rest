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

Load [External Effects](docs/spec-first-workflow/shared/external-effects.md)
immediately before an authorized external, costly, sensitive, or irreversible
action. Load [Repository Boundaries](docs/spec-first-workflow/shared/repository-boundaries.md)
before an accepted outcome first requires work in another checkout.

### Decision Ownership

The agent owns technical decisions, routing, proof, and rollout within the
accepted outcome. The user owns business meaning, otherwise-unowned policy,
priority and deadline, money, legal commitments, and irreversible external
effects.

Resolve ordinary uncertainty from repository evidence. Ask only when materially
different outcomes remain and no bounded assumption keeps the work honest;
otherwise state the assumption and its reopen condition.

### Proceeding

Within the current authority and macro phase, continue through every named
in-scope step until the workflow's [phase movement
boundary](docs/spec-first-workflow.md#phase-movement) requires handoff. A failed
lookup changes the evidence route, not the outcome: inspect an authoritative
alternative and do not repeat an unchanged failed route.

## Engineering

Reuse the current owner, repository pattern, standard library, and installed
dependencies before adding machinery. Match the surrounding code's naming,
comment density, idiom, and responsibility boundaries. Preserve unrelated work
and generated/manual authority.

## Routing And Loading

A Markdown link names an owner; it does not load it. Read the current owner
immediately before its first governed action or claim, and re-evaluate only when
evidence changes phase, risk, ownership, proof, or harness control.

## Direct Work

Use [Direct Work](docs/spec-first-workflow/direct-work.md) for a clear, local,
reversible, single-owner outcome with bounded proof and no unresolved protected
decision. Otherwise read the [workflow router](docs/spec-first-workflow.md) and
load only the owner it selects.

Load [Repository Architecture](docs/repo-architecture.md) before changing a
repository boundary, generated-source ownership, contract capability,
integration, persistence or durable flow, durable schema, or executable
surface. Load [Validation Routing](docs/validation-routing.md) before selecting
proof. Load [Prompt Maintenance](docs/prompt-maintenance.md) before editing
instructions, tools, roles, or skills; skill edits also load [Skill
Authoring](docs/skill-authoring.md). Load [Agent Harness](docs/agent-harness.md)
only before choosing or operating a durable control, carrier, model, or effort.

## Go Change Surface

For Go changes, apply only the skills whose descriptions match a pressure in
the changed surface; each selected skill owns its method and reopen condition.
