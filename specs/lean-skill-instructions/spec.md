# Lean, behavior-preserving skill instructions
status: ready

## Scope and non-goals

Rewrite the repository's triggerable skills so each is a minimal behavioral
adapter: precise trigger, owned outcome, decisive method, stop or escalation
boundary, checkable return, and conditional reference loading. Apply the rule
to every canonical `.agents/skills/*/SKILL.md`, not only the five examples in
the supplied analysis.

Add one short canonical skill-authoring standard, keep common specialist
mechanics in one shared contract, and reduce Codex's implicit routing surface
without removing explicit specialist entry points.

Do not rename or delete existing skills, change workflow phase authority,
weaken repository-specific invariants, rewrite domain references that still
change decisions, or add a new general engineering handbook. Do not claim
cross-runtime implicit-routing parity without a documented runtime mechanism.

## Behavior and contract delta

1. Every skill description front-loads the job and trigger, then states the
   decisive exclusion in one concise line.
2. Every skill body contains only instruction that changes routing, critical
   order, a material non-obvious failure mode, a stop/escalation boundary, or a
   checkable output/proof obligation.
3. Session/router skills remain thin adapters to their canonical workflow
   owners. Specialist specification and review skills reuse the shared
   specialist contract instead of repeating review, evidence, escalation, and
   handoff mechanics.
4. Long symptom/reference tables move out of `SKILL.md`. A skill names the
   pressure and loads at most one matching local selector/reference by default;
   additional references require independent pressures.
5. Method-heavy skills such as systematic debugging and verification retain
   their non-obvious causal or falsification method, but lose general
   professional teaching and duplicate repository policy.
6. The maintainability trio remains explicitly addressable but has mutually
   exclusive routing: Go/stdlib contract, local behavioral simplification, and
   explicit harsh whole-diff structural review.
7. Test skills remain explicitly distinct by result: design proof obligations,
   implement executable tests, review test quality, and verify a completion
   claim.
8. Codex gets one new implicit `go-specialist-router`. The router reconstructs
   the affected surface, selects one primary specialist for each materially
   affected independent axis, and uses shared arbitration to avoid duplicate or
   overlapping coverage. It does not rely on nested skill invocation: it reads
   each selected specialist's canonical `SKILL.md` as an instruction reference
   and applies it locally. The 29 current specification and review skills that
   load `.agents/skills/specialist-contract.md` switch to explicit-only through
   documented `agents/openai.yaml` policy only after a representative live
   baseline/candidate eval proves this automatic router path. Until that gate
   passes they retain implicit eligibility. All other existing
   workflow/session/direct-method skills retain current implicit eligibility
   unless a later reviewed change proves a narrower boundary.

## Invariants and edge cases

- `AGENTS.md`, the workflow router, phase documents, task artifacts, generated
  sources, and accepted domain contracts retain their existing authority.
- Shortening must not remove Worker-only implementation ownership, read-only
  review boundaries, fresh-proof requirements, verdict semantics, generated
  source ownership, or named escalation owners.
- A word budget is an authoring heuristic, not a hard correctness metric.
  Router/session skills should normally fit 50-150 words, ordinary specialists
  100-250 words, and genuinely non-obvious methods 250-500 words. Exceeding a
  budget is allowed only for a concrete behavior-preserving reason.
- References are loaded progressively. Rare detail stays available without
  occupying the default skill body.
- Existing explicit skill names remain compatible. Codex-specific implicit
  policy must not make a bundle invalid for other supported runtimes.
- Automatic specialist routing is not considered preserved until a
  representative behavior eval proves the router selects and applies
  explicit-only specialists without user `$skill` invocation. The gate must
  exercise the actual candidate metadata, at least one coupled multi-axis case
  that requires multiple specialists, and one single-axis negative case that
  rejects unrelated specialists.
- Missing runner, judge, or explicit cost authorization blocks only the 29
  explicit-only policy flips. Lean rewrites, the router, and closed eval
  definitions may land, but closeout must report routing reduction as pending
  rather than activating unproved metadata.
- The completed task bundle is removed after durable rules land in canonical
  docs and skills.

## Decisions, constraints, and authorities

- The user-supplied analysis is the accepted refactoring direction.
- OpenAI's current `Build skills` documentation owns Codex invocation metadata;
  the task research note records the evidence and limit.
- `docs/skill-authoring.md` will own the authoring contract. `AGENTS.md` will
  link to it only if a repository-wide authoring pointer is required; it will
  not duplicate the contract.
- `.agents/skills/specialist-contract.md` remains the non-triggerable owner of
  shared specialist selection, specification, review, evidence, return, and
  arbitration mechanics.
- Do not add a second execution/proof contract. `go-coder`, test
  implementation, systematic debugging, verification, and closeout remain
  thin adapters to `AGENTS.md`, the implementation/validation phase, and the
  narrow test or verification owner already named by those authorities.
- Individual `SKILL.md` files own only their unique trigger, domain invariant,
  decisive method, stop condition, return, and reference selector handoff.

## Success criteria and proof expectations

- All canonical skills satisfy the new authoring contract and retain valid
  frontmatter, unique names, valid relative links, and their explicit trigger
  compatibility.
- The full skill catalog is materially smaller than baseline, with per-family
  and total word-count evidence; no numeric reduction alone counts as success.
- Existing hard-skill, workflow-instruction, routing, agent, documentation,
  drift, and diff checks triggered by the changed surfaces pass freshly.
- Representative eval coverage exists for routing boundaries, specialist
  selection, stop/escalation, proof honesty, and method-heavy debugging and
  verification behavior.
- Live baseline/candidate model evals are run only when an authorized runner,
  judge, and cost authorization are available. Otherwise the closeout reports
  that proof gap explicitly, does not treat structural checks as behavioral
  equivalence, and leaves the 29 specialists implicitly eligible.

## Risks, assumptions, and reopen conditions

- Reopen specification if preserving an existing skill name conflicts with a
  user-visible routing requirement.
- Reopen design if a current platform rejects per-skill
  `agents/openai.yaml` or cannot preserve explicit invocation when implicit
  invocation is disabled.
- Reopen proof design if current eval infrastructure cannot isolate baseline
  and candidate skill bundles without changing their inputs.
- Assumption: reference content remains useful unless direct inspection shows
  it only repeats general professional knowledge or a higher canonical owner.
