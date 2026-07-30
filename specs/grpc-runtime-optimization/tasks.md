# Goal

status: done

Completion: the native gRPC template exposes a failure-preserving,
workload-tunable access-log policy, excludes routine server health telemetry by
default, and can build the service with an explicitly validated representative
PGO profile while preserving an immediate non-PGO rollback.

Global constraints: preserve RPC contracts, statuses, health serving,
admission, shutdown, duration telemetry for instrumented methods, generated
source, and current transport defaults. Add no runtime dependency and make no
numeric performance or PGO gain claim without equivalent evidence.

- [x] T1: The production gRPC observability path avoids disabled/sampled work while errors, slow calls, business telemetry, and explicit health diagnosis remain visible.
  - Source: [`spec.md` OPT-1, OPT-2, OPT-4](spec.md); [`design/overview.md` D1, D2](design/overview.md); [`test-plan.md` TD-1 through TD-4](test-plan.md).
  - Owner/surface/resources: `internal/config` owns keys/defaults/validation; `internal/infra/grpc` owns interceptors, OTel filter, and focused benchmarks; `cmd/service/internal/bootstrap/startup_grpc.go` owns mapping; `env/.env.example` and `docs/grpc.md` own operator behavior. The Go benchmark run is the only exclusive proof resource.
  - Depends on: none.
  - Proof: focused config/bootstrap/runtime tests prove validation and mapping; real RPC telemetry tests prove default exclusion and opt-in; repeated 64B production-composed benchmarks report full/sampled/disabled paths with exact response correctness; gRPC profile initialization remains byte-stable.
  - Reopen if: an audit owner requires every successful call, or a service SLO supplies a mandatory default slow threshold (Specification).
  - Acceptance: PASS — reviewer: `/root/grpc_runtime_opt_t1_final_acceptance` (fresh independent Task Acceptance, T1 only, read-only); evidence: focused config/bootstrap GRPC tests and access-log/telemetry/streaming/benchmark-composition tests passed; corrected TD-4 executable-alternation selector selected exactly `full_log_disabled`, `full_success_unsampled`, and `full_json` at 64B with payload oracles green; candidate: current T1 acceptance unit at source fingerprint `ebfd70991a797068a0f86d7ba590a037cd802cd1815b229d1fc58b844f219fdb` (HEAD `d59e72e2bd64e6eff900f3f237e10a8edf0e084b`).

- [x] T2: Local and production service builds accept only an explicit validated PGO profile and retain `off` as the default rollback.
  - Source: [`spec.md` OPT-3, OPT-4](spec.md); [`design/overview.md` D3](design/overview.md); [`test-plan.md` TD-5, TD-6](test-plan.md).
  - Owner/surface/resources: `Makefile` owns local build entrypoints; `build/docker/Dockerfile` owns image compilation; `docs/build-test-and-development-commands.md` and `docs/grpc.md` own profile lifecycle guidance. A representative CPU profile is a private operator-owned build input; Docker/build gates are exclusive broad resources.
  - Depends on: T1 — proof gate — needed to prove final closeout against the fixed optimized hot path.
  - Proof: ordinary `make build` succeeds with PGO off; empty/missing/malformed PGO inputs fail before success; a current repository CPU profile passes `make build-pgo`; Dockerfile check and focused production image build preserve the ordinary path; `git diff --check` is clean.
  - Reopen if: the delivery platform cannot supply a private repository-relative profile input, or another production binary needs a distinct profile (Delivery Design).
  - Acceptance: PASS — reviewer: `/root/grpc_runtime_opt_t2_pr_acceptance` (fresh independent Task Acceptance, T2 only, read-only); evidence: local default/valid/invalid PGO matrix, profile parsing, and clean-diff proof passed; GitHub PR #100 workflow `30541081335` passed `delivery-quality` job `90865996246` and `container-security` job `90865996310`, covering Dockerfile tooling, production image build, Trivy, runtime, and migration validation; candidate: `f5afe0855ef14056395b0a10c4edf540e3c7b61f`.
