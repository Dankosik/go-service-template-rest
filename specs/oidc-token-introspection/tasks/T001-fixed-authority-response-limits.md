# T001 — Fixed-authority response-header limits

Outcome:
The accepted `internal/infra/httpclient` owner adds one positive caller-supplied response-limit value and equivalent external/private constructors that enforce the fixed five-second response-header timeout and 32 KiB maximum for HTTP/1.1 and HTTP/2, while existing constructors, fixed-authority behavior, callers, and lifecycle remain unchanged.

Consumes:
- [`../design/system-integration-design.md`](../design/system-integration-design.md) sha256 `0c1c8b627cc79c6cac44bb727473e9a05bf37acba248b5ca9e87b47798759b53`, especially Fixed provider edge and its implementation-start refresh rule.
- [`../design/go-code-ownership-design.md`](../design/go-code-ownership-design.md) sha256 `41e754e723a27356752013abeabc51d0b269b0dfd9137c40401dcae233637523`, fixed-authority response-header ownership.
- [`../test-plan.md`](../test-plan.md) sha256 `856eb3dccf985c8d00764901e96b7c5843a4a54a8e4882eefa2b5e563a86dfea`, rows TD-HDR-01 through TD-HDR-03.
- Independent PASS receipts in [`../design/technical-design-review.md`](../design/technical-design-review.md) and [`../test-design-review.md`](../test-design-review.md).
- Accepted current owner at `HEAD` `8967a4ac06d4fce0515703b15ffa5db35e5378ae`; the dirty generic-client refactor is excluded.

Provides:
- One accepted fixed-authority response-header guard surface for T003.
- Direct transport proof that provider-adapter body limits cannot falsely stand in for header enforcement.

Boundary:
Extend only the accepted `internal/infra/httpclient` transport-construction owner and its package-local proof. Keep provider-selected values, request/body budgets, parsing, authn errors, retry, telemetry, dependency names, connection limits, and every existing caller outside this unit. If an independently accepted concurrent client already supplies equivalent semantics, use that surface with retry and instrumentation disabled instead of adding a parallel API.

Mutable owners:
- `internal/infra/httpclient` fixed-authority transport construction and package-local response-limit tests

Exclusive locks:
- `internal/infra/httpclient` constructor and encapsulated transport contract

Accept when:
- Claim: Positive limits preserve all existing fixed-authority behavior; non-positive limits fail before I/O; the real five-second header timeout and 32 KiB maximum reject the reviewed HTTP/1.1 and negotiated HTTP/2 falsifiers.
- Focused check: Run the exact `command_or_procedure` cells for TD-HDR-01, TD-HDR-02, and TD-HDR-03 in the fixed Test Design, with non-zero matching test counts and each row's independent observable.
- Observable: Old and limited constructors pass the under-limit controls; invalid construction performs no I/O; withheld headers fail at the fixed bound before body parsing; over-limit headers fail before body delivery on proven HTTP/1.1 and `h2`; no caller or lifecycle behavior changes.

Reopen if:
Reopen System / Integration Design only if the refreshed accepted client cannot preserve destination pinning, post-DNS admission, no proxy/redirect/retry, lifecycle, and both header guards through one equivalent surface. Reopen Planning only if this result can no longer land green and be consumed by T003 without unfinished companion work. A constructor-name-only refresh stays inside Implementation.
