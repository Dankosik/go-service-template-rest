# Subagent Brief Template

Use only the fields that can change the lane's answer or stop decision.

```text
Question: <one decision or falsification target>

Context:
- <accepted facts and minimal artifact paths>

Evidence boundary:
- inspect: <sources/files>
- enough evidence: <threshold>

Constraints:
- <read-only/write boundary, non-goals, external-action limits>

Output:
- <required finding/evidence/recommendation shape>

Stop:
- <completion condition or missing-input blocker>
```

Do not copy repository-wide workflow rules, model catalogs, unrelated context, or generic strictness language into every brief.
