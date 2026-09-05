# Material Rule

Load for every materially affected Specification rule before declaring its
observable divergence closed.

A material rule closes only the parts that can change meaning: actor/context,
trigger/input/preconditions, normative rule and precedence, states and invalid
or repeated transitions, observable normal/boundary/rejection/failure/recovery/
compatibility outcomes, required or forbidden effects, identifiers/units/bounds,
absence/default behavior, and the nearest feasible falsifier.

Ask whether two reasonable implementations could satisfy the wording yet
produce different user- or operator-visible results. If yes, resolve the fork
from current authority, a Specification decision, or a bounded assumption with
an objective reopen condition.

Use plain prose unless a compact decision table, state model, quality scenario,
decisive example, or literal schema/type fragment closes the ambiguity more
precisely. Keep only the decision-carrying part; exhaustive scenarios belong to
Test Design and mechanism or placement belongs to Technical Design.
