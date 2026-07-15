# Codex skill routing metadata
status: ready

Checked: 2026-07-16

## Question

Can this repository reduce the model-visible specialist surface without
deleting explicit specialist skills or relying on undocumented frontmatter?

## Current evidence

- OpenAI's current `Build skills` documentation says Codex initially exposes a
  skill's name, description, and path, truncates descriptions first when the
  catalog is large, and may omit skills once the catalog exceeds its prompt
  budget.
- The same documentation defines `agents/openai.yaml` and
  `policy.allow_implicit_invocation: false`. That policy prevents implicit
  selection while preserving explicit `$skill` invocation.
- The documentation recommends concise, front-loaded descriptions with clear
  positive and negative routing boundaries.

Sources:

- https://learn.chatgpt.com/docs/build-skills
- https://developers.openai.com/cookbook/examples/skills_in_api#operational-best-practices

## Decision implication

Codex has a documented mechanism for keeping rare specialists explicitly
available without making them implicit entry points. The target design may use
one implicit specialist router plus per-specialist `agents/openai.yaml` policy.
Other runtimes may continue to expose the same skills explicitly; do not add
undocumented cross-platform frontmatter in this change.

## Limit

The official sources describe Codex behavior only. Portability beyond Codex
must preserve valid `SKILL.md` bundles and explicit invocation, but equivalent
implicit-routing metadata is a runtime-specific follow-up unless supported by
that runtime's current contract.

The sources do not document nested skill invocation by another skill. The
target must not depend on that behavior. An implicit router may instead use its
ordinary repository-reading procedure to select and read one exact canonical
specialist `SKILL.md` as an instruction reference. This is a bounded design
inference, not a documented invocation-policy guarantee, so representative
behavior proof is required before claiming automatic specialist routing works.
