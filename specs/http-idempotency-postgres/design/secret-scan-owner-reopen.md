# Secret-scan runtime semantics reopen

status: review-cleared

## Fixed boundary

This Technical Design and Test Design addendum replaces the failed
exact-false-positive classification. It reopens only the runtime semantics of
the S3 `generic-api-key` exception used by PR-GOSEC-01. The preserved
implementation candidate remains
`309b9a912535243da159a4b7faf8d0559ebee94cbb2f81348839f2c5642dc12e`.
The supplied ledger revision remains
`0aa92e678aec39ace7639f9022585da7aa246d2f9ce2b103d5183805b975fd61`.

No receipt, S3 design value, webhook wording candidate, scanner baseline,
ledger, candidate, CI workflow, or Make target is writable in this reopen.
The blocked Lead and its Goal remain preserved. This addendum does not accept
PR-GOSEC-01 or authorize an aggregate beyond the scanner route named below.

## Runtime falsifier

Installed Gitleaks is `github.com/zricethezav/gitleaks/v8 v8.30.0`.
`make secret-scan-check` passes, but the actual copied-worktree route of
`make secret-scan` reports exactly these five `generic-api-key` findings:

| Path | Line | Classification |
| --- | ---: | --- |
| `scripts/ci/secret-scan.sh` | 168 | Self-test source contains the S3 script receipt literal. |
| `scripts/ci/secret-scan.sh` | 169 | Self-test source contains the S3 design receipt literal. |
| `scripts/ci/secret-scan.sh` | 170 | Self-test source contains the synthetic generic-key fixture literal. |
| `scripts/ci/s3-source-receipt.sh` | 89 | Immutable public AWS module checksum receipt. |
| `specs/s3-compatible-object-storage/design/overview.md` | 415 | Same immutable public checksum receipt. |

The existing one-entry allowlist is not runtime-correct. In installed v8.30.0,
the `line` target for a non-first line begins at the preceding newline, so its
anchored `^...$` alternatives cannot match either receipt line. A `^\n?` probe
clears the two receipts, which confirms that cause, but is less exact than the
selected mechanism. Two `regexTarget = "match"` exceptions, each anchored to
the actual `generic-api-key` match and its one exact path, suppress only the
two S3 findings; the three self-test-source findings remain. The result is
observed scanner behavior, not an inference from documentation.

## Selected correction

The smallest runtime-correct correction owns exactly:

1. `.gitleaks.toml`
2. `scripts/ci/secret-scan.sh`

Replace the single S3 allowlist with two `generic-api-key` allowlists. Each
uses `condition = "AND"` and `regexTarget = "match"`; one pairs the exact
Gitleaks match for the shell receipt with only
`^scripts/ci/s3-source-receipt\.sh$`, and the other pairs the exact match for
the design receipt with only
`^specs/s3-compatible-object-storage/design/overview\.md$`. Both patterns are
anchored to the complete reported match, not a source line, and no path list is
shared between them.

Change the existing disposable self-test only to assemble the public receipt
and synthetic generic-key fixture values from non-matching fragments at runtime
inside its temporary repository. The fixture still writes the same exact
bytes before the scanner runs; this removes the three self-test literals from
the repository scan rather than allowlisting scanner-test source. Keep the
existing synthetic GitHub PAT negative checks and strengthen their oracle to
require the expected rule ID where the scanner reports one.

Rejected mechanisms: a baseline, a whole-file or path-only exemption, a
generic-rule change, a receipt edit, a third source-code allowlist, and any
CI/Make route change. They either hide more than the two public receipts or
alter a preserved authority.

## Required proof

The correction Lead runs one serialized `make secret-scan-check`; its
disposable fixture must use the same `change` mode as production: indexed
snapshot plus modified/deleted/untracked overlay, configured scanner, baseline,
merge-base range, and missing-base full-history fallback. The self-test rows
are:

| Falsifier | Oracle |
| --- | --- |
| Each S3 receipt alone before the exception is present | `change` exits 1 and verbose output contains `RuleID: generic-api-key`. |
| Each exact receipt at its one approved path with the exception restored | `change` exits 0. |
| Either receipt at the other approved S3 path | `change` exits 1 with `generic-api-key`. |
| A synthetic generic key appended to either approved receipt line or placed on a separate approved-path line | `change` exits 1 with `generic-api-key`. |
| Either receipt in an unapproved path | `change` exits 1 with `generic-api-key`. |
| Untracked GitHub PAT, deleted-in-range GitHub PAT, full-history GitHub PAT, and missing-base fallback | Each fails under its existing route; the focused negative oracle retains the expected detection rule. |

After that focused proof, the correction Lead runs exactly one real copied-tree
route: `make secret-scan SECRET_SCAN_BASE_REF=HEAD`. It must close all five
listed findings and reaches commit scanning only after the copied-worktree
scan is clean. The later PR-GOSEC-01 Lead, not this reopen, reruns
`make secret-scan SECRET_SCAN_BASE_REF=<exact-pr-base-sha>` on its frozen
candidate, then its complete recorded proof and fresh independent review.
Main/release retain `make secret-scan-history` after the exact PR gate; no
history claim is supplied by the correction fixture.

Safe serialization: perform this security correction before the independent
webhook wording correction; do not run another broad Go or Docker gate while
the scanner route runs. The correction changes no S3 receipt, so its existing
receipt authority remains valid once the exact bytes are rechecked.

## Reopen conditions

Reopen Go Security if a non-S3 path or a non-exact match is exempted, any
synthetic key passes, a real credential is found, or a baseline/rule weakening
is proposed. Reopen Test Design if the fixture cannot prove both pre-exception
detection and post-exception rejection rows through `change` mode. Reopen
Delivery if snapshot/commit ordering, CI routing, or full-history policy
changes. Reopen S3 ownership if either immutable receipt byte changes. Any
such condition blocks PR-GOSEC-01 resumption.

## Review disposition

Fresh independent Technical Design review returned **PASS** on the fixed
pre-disposition addendum SHA-256
`4a4b094f270b12e78ddc4c37b01b00ecfa9ce18f1a773b551b5a0269d634a2d3`.
It independently confirmed the v8.30.0 preceding-newline behavior and the
one-match/one-path mechanism, including swapped-path, appended/separate-key,
and unapproved-path falsifiers. This status and disposition recording changes
no selected mechanism or proof obligation.
