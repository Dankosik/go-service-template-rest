# Adoption hardening proof

TD-1 | derived identity/profile safety | initialize full repository with defaults and explicit full profiles | assert unique identity, no template links, minimal profile compiles, and full profile retains PostgreSQL/agents | `make template-init-check`

TD-2 | metrics trust boundary | run client and diagnostics handlers separately | `/metrics` is 404 on the application handler and 200 only on diagnostics | focused HTTP/bootstrap tests

TD-3 | public ingress | non-local wildcard bind with explicit true | policy accepts; missing declaration still fails | focused network-policy tests

TD-4 | strict config | unknown YAML, overlay, and `APP__...` key | load returns the stable unknown-key error before runtime wiring | focused config tests

TD-5 | lifecycle | readiness failure, early server stop, signal, diagnostics bind/serve failure, shutdown timeout | no readiness leak, no blocked goroutine, bounded exit, stage-preserving error | focused bootstrap tests, repeated test, race, goleak

TD-6 | optional PostgreSQL | minimal derived profile and full template service graph | minimal build has no PostgreSQL surfaces; full `go list -deps ./cmd/service` excludes `golang-migrate` | template-init check and dependency assertion

TD-7 | delivery admission | default repository variables and explicit enablement | publish jobs are skipped by default without weakening signing/attestation after admission | workflow structural check and CI syntax/check suite

TD-8 | feature adoption guide | every mapped source and command exists; reference feature tests run | guide cannot silently point at stale owners | documentation check plus existing reference-service tests

TD-9 | integrated candidate | all changed sources, generated boundaries, docs, and commands | no drift, lint/security/test regressions, or unowned changes | `git diff --check`, focused checks, `make check`, triggered CI-local/check-full gates
