# Keep only service-owned tooling
status: blocked
Implementation status: T1 and every writable T2 surface are implemented. T2
cannot delete or repair `.agents/skills/*/evals/**` and its stale runtime
references because the current sandbox exposes `.agents` read-only. T3 has
current static, initialization, lint, generated-contract, OpenAPI, gosec, and
host-benchmark evidence; full tests are blocked by sandbox loopback-bind
denial, `govulncheck` by unavailable `vuln.go.dev`, secret scanning by four
historical findings in a deleted `.agents` skill, and Docker-backed proof by
the inaccessible OrbStack socket. None of those gaps is reported green.
Completion: The local template exposes only the accepted initialization,
service-validation, benchmark, runtime, migration, and security command surface;
all retired machinery and active references are gone; native and static
claim-scoped proof passes.
Blocked stop: Docker-backed integration, migration runtime/cleanup, and image
build/scan remain unclaimed; Implementation/Validation reopens them when a
reachable daemon can pass `make check-full` plus the forced-failure
invocation-scoped Compose cleanup oracle. Publication remains unclaimed and no
push is allowed; a repository administrator reopens it by exporting the
effective Ruleset/classic protection, replacing retired required contexts with
the surviving blocking job IDs from `.github/workflows/ci.yml`, and recording
the before/after comparison.
Global constraints: Preserve unrelated benchmark work and all other user changes.
Keep existing benchmark commands/artifacts, removing only their Docker-Go
fallback and moving pinned k6 execution to the benchmark owner. Do not mutate
GitHub settings or publish. Immediately before T1 edits, snapshot the complete
dirty checkout under
`/private/tmp/go-service-template-rest-benchmark-baseline-20260720`: create
`tracked.patch` with `git diff --binary --output=<path>`, create
`untracked.tar` from the exact newline-delimited
`git ls-files --others --exclude-standard` output, and record SHA-256 for both.
This deterministic recoverable baseline is outside the repository and therefore
covers every overlapping tracked and untracked benchmark path without a
hand-maintained file list.
Planned waves:
- W1: T1; it establishes every surviving command owner before deletion.
- W2: T2; it consumes T1's final command surface.
- W3: T3; terminal proof requires the complete candidate.

- [x] T1: Establish the minimal native command, initialization, benchmark, and
  Docker-backed service-gate owners without changing service runtime behavior.
  - Source: `spec.md` Behavior and contract delta; Invariants and edge cases;
    Decisions, constraints, and authorities.
  - Owner/surface/resources: `Makefile`, `scripts/init-module.sh`,
    `scripts/ci/template-init-check.sh`, `scripts/dev/benchmark.sh`,
    `env/docker-compose.yml`; local Go/Node tools and isolated temporary
    initialization fixtures; benchmark backup path
    `/private/tmp/go-service-template-rest-benchmark-baseline-20260720`.
  - Depends on: none
  - Handoff: T2 receives the final Make target names and the four retained
    script owners. T3 receives the exact benchmark backup path and its content
    manifest/hash.
  - Proof: initialization success and pre-mutation rejection; `make
    template-init-check` passes and its forbidden-import fixture is rejected by
    depguard after module rename. The same check proves a template-source origin
    without `CODEOWNER` preserves its existing module, sources, lint config,
    CODEOWNERS, and `.env` byte-identically, and proves successful
    derived-repository initialization never overwrites an existing `.env`.
    Command ownership; `make help` and Make dry runs contain no general Docker
    mirror. Native aggregate; `make ci-local` runs no conditional Docker skip.
    Static Docker wiring; Compose configuration and `make -n check-full` show
    integration, isolated migration plus `/migrate`, and the digest-pinned image
    scan without claiming execution. The external dirty-state baseline exists,
    both recorded hashes revalidate, and its archive member list equals the
    pre-edit untracked list before repository mutation.
  - Reopen if: benchmark commands/artifact semantics cannot survive without the
    general Docker emulator; Specification owns that contract.

- [ ] T2: Remove retired agent-authoring, self-guardrail, Docker-emulation,
  repository-admin, and diagnostic machinery, then align CI, runtime
  instructions, skills, and operator documentation with T1.
  - Source: `spec.md` Scope and non-goals; Behavior and contract delta;
    Decisions, constraints, and authorities.
  - Owner/surface/resources: delete `scripts/ci/hard-skills-check/`,
    `scripts/ci/instruction-evals-check.sh`,
    `scripts/ci/required-guardrails-check.sh`,
    `scripts/ci/workflow-instructions-check.sh`,
    `scripts/dev/codex-worktree-preflight.sh`,
    `scripts/dev/configure-branch-protection.sh`,
    `scripts/dev/docker-tooling.sh`, `scripts/dev/doctor.sh`,
    `scripts/dev/module-origin.sh`, `scripts/dev/setup.sh`,
    `scripts/dev/workflow-behavior-evals.sh`,
    `build/docker/tooling-images.Dockerfile`,
    `docs/spec-first-workflow-evals.md`, all `.agents/skills/*/evals/**`, and
    stale eval-only references; before deleting the eval trees, rehome the three
    runtime examples consumed by
    `agent-prompt-composer/references/example-transformations.md`
    (`http-options-cors.md`, `skill-tooling.md`, `flaky-shutdown.md`) under that
    skill's `references/` subtree and update their links/commands. Update
    `.github/workflows/*.yml`, active repository docs/instructions,
    delivery/security skill references, `.github/CODEOWNERS`, `railway.toml`,
    and `test/README.md`.
  - Depends on: T1
  - Current blocker: every writable owner and active reference outside
    `.agents` is repaired, but `.agents` is read-only in this sandbox. It still
    contains 53 eval files and 24 references to retired commands across 13
    runtime files.
  - Proof: workflows use only surviving Make targets; merge CI and release
    preflight retain lint, coverage, race, integration, migration, generated,
    Go/secret/container security, and publishing gates without standalone
    ordinary test/vet or methodology checks. Repository-wide active-surface
    scan finds no retired identifier or deleted path. `scripts/` contains only
    init, init contract check, benchmark, and generated drift.

- [ ] T3: Review and prove the complete local candidate without disturbing
  unrelated checkout state.
  - Source: `spec.md` Success criteria and proof expectations; Risks,
    assumptions, and reopen conditions.
  - Owner/surface/resources: complete task diff; local Go/Node tools; generated
    coverage/test artifacts only.
  - Depends on: T2
  - Current evidence: shell syntax, Make parsing/dry-runs, workflow and Compose
    YAML, `git diff --check`, template initialization contract, formatting,
    golangci-lint, deadcode, NilAway, module drift, SQLC/OpenAPI drift and
    runtime contract, Redocly/schema validation, gosec, and host
    benchmark/benchstat paths passed. The blocked observables are recorded in
    the implementation status above.
  - Proof: whitespace integrity; `git diff --check` exits zero. Workflow syntax;
    `/usr/bin/ruby -e 'require "yaml"; Dir[".github/workflows/*.{yml,yaml}"].sort.each { |f| YAML.parse_file(f) or abort f }'`
    exits zero. Initialization; `make template-init-check` passes every positive,
    rejection, template-source no-op, existing-`.env` preservation,
    byte-identity, and depguard case. Native quick and broad aggregates; `make
    check` and `make ci-local` exit zero, thereby owning lint, coverage, race,
    OpenAPI/sqlc drift, and Go/secret security without duplicate standalone
    runs. Static Docker wiring; `make -n check-full` and
    `BASE_REF=HEAD make -n pr-check` parse and print integration, migration
    runtime, image scan, and OpenAPI breaking commands without a deleted target.
    Retired reachability; `rg --hidden -n
    'hard-skills-check|workflow-routing-check|workflow-behavior-evals|instruction-evals|guardrails-check|gh-protect|docker-tooling|docker-(check|ci|fmt|lint|test|mod|openapi|sqlc|go|secret|migration|container|pull|init)|doctor-(native|docker)|template-init-(strict|native|docker)|scripts/dev/(setup|doctor|module-origin|codex-worktree-preflight)\.sh|spec-first-workflow-evals'
    --glob '!specs/**' --glob '!.git/**' .` returns no match. Script inventory;
    sorted `rg --files scripts` equals exactly
    `scripts/ci/generated-drift-check.sh`,
    `scripts/ci/template-init-check.sh`, `scripts/dev/benchmark.sh`, and
    `scripts/init-module.sh`. Deletion coverage; a shell loop over every exact
    path listed in T2 observes `! -e`, a fixed-string active-surface scan for
    each deleted basename returns no match outside `specs/` and `.git/`, and
    `rg --files .agents/skills | rg '/evals/'` returns no match. Comparison with
    the external pre-edit patch/copy manifest shows benchmark commands and
    artifact contract remain except the accepted Docker-Go fallback removal and
    k6 relocation. Pre-existing unrelated changes remain present or recoverable
    from that baseline.
    Record Docker-backed integration, isolated migration runtime/cleanup, and
    runtime image scan as unverified; they are excluded from current completion
    because the current OrbStack socket is not reachable from this environment.
  - Reopen if: a native or static protected gate cannot execute through its
    accepted owner, or concurrent user edits overlap an unmerged task surface;
    implementation/validation records the exact gap without claiming it green.
