# Execution Shape Selection

## Behavior Change Thesis
When loaded for symptom "the recorded route may not match its evidence," this file makes the model validate or falsify the existing `SHAPE-*` result against canonical rule IDs instead of becoming a second classifier.

## When To Load
Load this only when the recorded shape or trigger audit is the active uncertainty. Shape selection itself belongs to `AGENTS.md`; if the hard part is lane design, typed artifact state, file authoring, or the adequacy boundary, load that narrower reference instead.

## Decision Rubric
- Require one matched `SHAPE-*` rule plus evidence for every evaluated `FULL-*`, `DIRECT-*`, and `LEAN-*` row.
- Reject `SHAPE-DIRECT` when any `DIRECT-*` predicate is false or approval-relevant unknown, or when any `FULL-*` floor is true or unknown.
- Reject `SHAPE-LEAN` unless every `LEAN-*` predicate is true and every `FULL-*` floor is false.
- Accept `SHAPE-FULL-FLOOR` only with at least one true/unknown `FULL-*` row. Accept `SHAPE-FALLBACK-FULL` only when direct and lean admission both fail; unsubstantiated full is challengeable.
- Treat `AGENT-CAPABILITY` as capability only. Only `AGENT-SUBSTANTIVE` activates `FULL-AGENT-SUBSTANTIVE`.
- If evidence changes, return the exact falsification and required `TRANS-*` reopen; do not overwrite `execution_shape` in this reference.

## Imitate

Direct-path calibration:

```markdown
Matched rule: SHAPE-DIRECT
Trigger audit: every FULL-* false; every DIRECT-* true; agent_request=capability_only
Actor: current orchestrator before any ledger exists
Artifact consequence: DEPTH-DIRECT
Reopen: TRANS-DIRECT-ELIGIBILITY-LOSS if any predicate changes
```

What to copy: the explanation names why the full workflow contract is not buying safety.

Lean-local calibration:

```markdown
Matched rule: SHAPE-LEAN
Trigger audit: every FULL-* false; every LEAN-* true
Artifact consequence: DEPTH-LEAN
Reopen: TRANS-UPWARD if later evidence activates a FULL-* floor
```

What to copy: the escalation trigger is part of the route, not an afterthought.

Full-orchestrated calibration:

```markdown
Matched rule: SHAPE-FULL-FLOOR
Trigger audit: FULL-PUBLIC-CONTRACT, FULL-DATA, FULL-SECURITY, and FULL-DELIVERY=true
Artifact consequence: DEPTH-FULL
Adequacy: ADEQUACY-FULL-SHAPE and ADEQUACY-FULL-TRIGGER
```

What to copy: the route names the affected seams without doing their research.

## Reject

```markdown
Matched rule: SHAPE-FULL-FLOOR
Evidence: the repository prefers orchestrators and workflow artifacts.
```

Failure: no canonical `FULL-*` evidence supports the claimed rule.

```markdown
Matched rule: SHAPE-LEAN
Evidence: probably only code.
```

Failure: it does not prove every `LEAN-*` row or falsify every `FULL-*` row.

## Agent Traps
- Recomputing shape locally instead of checking the recorded `SHAPE-*` evidence.
- Treating capability authorization as `FULL-AGENT-SUBSTANTIVE`.
- Accepting full without `SHAPE-FULL-FLOOR` or `SHAPE-FALLBACK-FULL` evidence.
- Starting research while validating the route.
