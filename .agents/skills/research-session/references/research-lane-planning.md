# Research Lane Planning Examples

## When To Load
Load this file when `workflow-plans/research.md` needs clearer lane ownership, role choice, evidence targets, parallelism, or fan-in mechanics.

Do not load it to invent lanes for tiny local research. A lane plan should reduce ambiguity, not add ceremony.

## Lane Planning Shape
A useful research lane names:
- lane ID
- owned question
- execution type: local or read-only subagent
- role
- one skill or `no-skill`
- evidence target
- order or parallelism
- status

## Good Research Artifact
```markdown
## Research Lanes

Research mode: fan-out
Parallelism: L1, L2, and L3 can run in parallel. L4 waits for fan-in only if API/data conflict remains.

| Lane | Execution | Role | Skill | Owned question | Evidence target | Status |
| --- | --- | --- | --- | --- | --- | --- |
| L1 | subagent | api-agent | no-skill | What existing endpoint patterns apply to async job creation, lookup, and download handoff? | route files, OpenAPI inputs, handler tests | planned |
| L2 | subagent | data-agent | no-skill | What persisted state shape and tenant isolation patterns already exist for long-running work? | migrations, repository methods, transaction tests | planned |
| L3 | subagent | security-agent | no-skill | What trust-boundary risks exist for signed download URLs? | auth middleware, download handlers, token or signing helpers | planned |
| L4 | local | orchestrator | no-skill | Are lane outputs compatible enough for specification handoff? | L1-L3 summaries and preserved research notes | pending fan-in |
```

Why this is good:
- each lane owns one question
- the security lane uses one review skill and stays read-only
- local fan-in belongs to the orchestrator
- nothing asks a lane to edit `spec.md`, create `design/`, or write a plan

## Bad Research Artifact
```markdown
## Research Lanes

- API/data/security/reliability: research export jobs.
- qa-agent: make the test plan.
- worker: try the implementation and report back.
- challenger-agent: approve the final spec.
```

Why this is bad:
- the first lane has no owned question or evidence target
- QA is asked to make a future planning artifact
- worker execution is write-capable
- a challenger is asked to approve final spec instead of advising

## Evidence Vs Decision Separation
Good lane output request:

```markdown
Return:
- repository evidence with file references
- external source links, if used
- conflicts or weak evidence
- a handoff note describing what future specification must decide

Do not:
- edit files
- approve decisions
- create design or plan artifacts
```

Bad lane output request:

```markdown
Return the final endpoint shape, schema, task list, and implementation order.
```

The good request keeps the lane advisory and evidence-oriented. The bad request collapses specification, design, and planning into research.

## Exa Source Links
These external method links were found through Exa before drafting the examples. Use them as support for structured evidence gathering only.

- [University of Minnesota, Frameworks for Systematic Review and Evidence Synthesis](https://libguides.umn.edu/c.php?g=1264119&p=9278660)
- [University of California Santa Cruz, Creating the Search](https://guides.library.ucsc.edu/c.php?g=1440266&p=10697344)
- [Cochrane Handbook, Chapter 4: Searching for and selecting studies](http://training.cochrane.org/handbook/current/chapter-04)
- [University of Tasmania, Documenting Search Strategies](https://utas.libguides.com/SystematicReviews/Documenting)
