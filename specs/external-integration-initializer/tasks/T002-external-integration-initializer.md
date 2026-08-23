# T002 — Complete External Integration Initializer

Outcome:
The source template implements exactly the reviewed fail-closed `make integration-init` initial, no-op, and generated-only refresh behavior for named HTTP and gRPC integrations, reuses the retained transport and OAuth owners, preserves manual and unrelated bytes, takes no custody of `.env`, exposes no provider operation, and satisfies all 25 fixed Test Design rows.

Consumes:
- T001 accepted `outbound_http` choice, lock/readback, and bounded HTTP retention surface.
- [`../test-design-transition.md`](../test-design-transition.md) SHA-256 `8c4afa98856011370f897cdb5bcee903d985104d2d39cb2148e8b44ffd4dffc3`.
- [`../spec.md`](../spec.md) SHA-256 `9a54ee75953d242cd37cd27b56e791e2e7f92e1fbdb7e5e528f9917bb50fbbf1` and independent [`../review.md`](../review.md) SHA-256 `0ce18e168e5f90ddcf631a164567f283e2a732b88e95aef235e6dc1791a71395`.
- [`../design/overview.md`](../design/overview.md) SHA-256 `ad02cc02cd79dae097850eb241cb8d0f04ce8ee399fc5b2882a68b0255d3c2ac` and independent [`../design/review.md`](../design/review.md) SHA-256 `d80db1713d1118e2345b5cb3297b842f6bf9c21e5ca2d7d04a960a5d4dbd2639`.
- [`../test-plan.md`](../test-plan.md) SHA-256 `4e5d409a6cc3f153d29740163817b2a01ebf64e67aca8a8dc4ac90259ca499d3` and independent [`../test-design-review.md`](../test-design-review.md) SHA-256 `ee8fb27a050fd9cb537c95f33811b9c6225f4671ff6adb2dd7fdb719299d742b`.
- At dispatch, local pinned OpenAPI/Buf/Go tools and a readable `origin/main` base; for acceptance, a Linux runner with local `strace` and Docker for the mandatory custody and delivery gates. Unavailability is an evidence gap, never a substitute or pass.

Provides:
- Complete locally accepted initializer command, transaction, integration identity, HTTP/gRPC canonical generation and drift, concrete named config/adapter/bootstrap lifecycle, generated/manual separation, tracked examples/documentation, and aggregate routing.
- Exact initial failure restoration, same-identity no-op, generated-only refresh, unrelated/manual byte preservation, and non-disclosure evidence suitable for the required integrated review.
- No provider operation, provider/deployment compatibility claim, credential, network action, or `.env` custody.

Boundary:
Implement the reviewed Make/script transaction, integration record, generator registration, generated/manual scaffold, named config, concrete adapter, bootstrap construction/close, singleton-to-first-named OAuth migration, documentation/examples, canonical validation routing, and the single disposable command harness with its generated package-local tests. These are execution lanes under one postcondition, not separate acceptance units. Reuse repository Bash/Git, existing dependencies, retained HTTP/gRPC/OAuth owners, and pinned generators. Do not add a framework, registry, generic interface, provider SDK, dependency, callable provider operation, retry/readiness policy, live value, or external action. Never inspect or mutate the checkout's actual `.env`; only the fixed harness may create synthetic entries in disposable repositories and observe them through the accepted presence-only oracle.

Mutable owners:
- Integration initializer command, clean-start transaction, identity record, generated/manual renderers, and disposable proof harness
- Canonical external OpenAPI and Protobuf source/output registration, drift, structure, and change-scope routing
- Named integration config, concrete adapters, bootstrap construction/close, and first-OAuth singleton-to-named source migration
- Template-init integration retention proof, tracked config examples, dependency boundaries, and initializer/integration architecture documentation

Exclusive locks:
- Integration initializer transaction, identity, render, and proof-harness contract
- Canonical OpenAPI/Protobuf generation and drift routing
- Template profile/removal and `template.lock` contract
- Integration config schema, first-OAuth singleton migration, and bootstrap dependency lifecycle

Accept when:
- Claim: Every scenario and non-test falsifier E1-HTTP-01 through E8-NAMED-01 reaches its fixed independent oracle for one exact candidate, including fail-before-mutation admission, Linux syscall custody, byte restoration, no-op/refresh containment, retained-owner use, non-disclosure, local fake exchange, canonical drift, aggregate routing, and lifecycle order.
- Focused check: Run the nine commands in the fixed Test Design Validation ladder, in order, against the exact T001+T002 candidate; record all 25 matrix IDs with the exact case count. A zero-match, cached, skipped, unavailable mandatory input/tool, untriggered wrapper, unreadable secret-scan base, or missing Docker/required Linux trace is not passing evidence.
- Observable: Every command is terminal-success; every fixed row reports its named oracle; only reviewed initial-mode paths change, refresh changes generated-only bytes, manual/unrelated bytes remain identical, actual `.env` stays uninspected and untouched, generated packages compile and remain contained, and no provider/deployment claim or external effect occurs.

Reopen if:
Use the smallest row-specific reopen owner in the fixed Test Plan. Reopen Technical Design or Go Ownership only when a reviewed mechanism, placement, or proving surface cannot realize the row. Reopen Test Design only for a false-pass, nondeterministic control, infeasible proving layer, missing mandatory input definition, or command that does not execute its row. Reopen Specification only if the fixed fail-closed behavior, developer/operator custody boundary, byte-restoration rules, or initial/refresh consequences cannot be preserved without changing accepted behavior or authority.
