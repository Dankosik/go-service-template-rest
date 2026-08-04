# Reporting a System Design

Use the full interface only when several components, guarantees, failure domains, or rollout states interact. Omit empty sections. Prefer one row per decision-changing journey or failure class, and state a contract once rather than repeating it across verdict, tables, and proof. Completeness is coverage, not length.

```markdown
## Verdict
[Recommended design, boundary, readiness state, and decisive forces]

## Requirements and assumptions
| Item | Target or assumption | Evidence/status |
| --- | --- | --- |

## Capacity envelope
| Flow | Peak/burst | Data and fan-out | First ceiling/headroom |
| --- | --- | --- | --- |

## Architecture
[Mermaid diagram plus component responsibility and data ownership]

## Interaction and state contracts
| Edge/flow | Identity and commit | Consistency/ordering | Deadline/backpressure | Recovery |
| --- | --- | --- | --- | --- |

## Failure and recovery
[Failure matrix, degraded modes, remaining capacity, RPO/RTO]

## Decisions and tradeoffs
[Forces, patterns earned, strongest alternative, and falsifiers]

## Evolution and proof
[Compatibility, migration, rollout, rollback or roll-forward, and tests]

## Handoffs and gaps
[For each required deep dive: `skill -> unresolved question or evidence`; then missing evidence, open decisions, and next action]
```

## Completion criterion

A full design or readiness claim is complete when:

- requirements and conflicts are classified;
- design-driving paths have numeric envelopes or named missing inputs;
- every component is earned by a force;
- stateful flows name authority, commit, guarantees, divergence, and recovery;
- critical journeys have bounded failure behavior, remaining capacity, signals, and tests;
- material choices include a practical alternative and falsifier;
- high-risk assumptions, mixed-version rollout, rollback or roll-forward, temporary-state cleanup owners, and gaps have evidence or an executable validation plan.
