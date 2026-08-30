# Goal
status: ready
Completion: T001 and T101 through T129 each have one independently accepted candidate or bounded blocked receipt; every canonical repository is zero-drift against the accepted template or names its exact blocker; duplicate checkouts consume the canonical Git result.
Global constraints: Apply the accepted Specification, Technical Design, Test Design, Repository Boundaries, External Effects, and Template Sync owners. Consumer pushes, pull requests, deployments, required-check changes, credentials, and provider actions remain unauthorized.

## Tasks

- [x] T001: Publish the source template's portable, profile-independent tooling through the existing sync engine.
  - Depends on: none
  - Provides: accepted portable tooling baseline on template `main` at `493e61ae6abc311df89a58d7cba64f407311ff41`.
  - Packet: tasks/T001-source-template.md

- [ ] T101: Migrate the canonical `Dankosik/analytics-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `Dankosik/analytics-service`.
  - Packet: tasks/T101-dankosik-analytics-service.md

- [ ] T102: Migrate the canonical `Dankosik/api-key-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `Dankosik/api-key-service`.
  - Packet: tasks/T102-dankosik-api-key-service.md

- [ ] T103: Migrate the canonical `Dankosik/billing-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `Dankosik/billing-service`.
  - Packet: tasks/T103-dankosik-billing-service.md

- [ ] T104: Migrate the canonical `Dankosik/document-processing-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `Dankosik/document-processing-service`.
  - Packet: tasks/T104-dankosik-document-processing-service.md

- [ ] T105: Migrate the canonical `Dankosik/identity-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `Dankosik/identity-service`.
  - Packet: tasks/T105-dankosik-identity-service.md

- [ ] T106: Migrate the canonical `Dankosik/model-catalog-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `Dankosik/model-catalog-service`.
  - Packet: tasks/T106-dankosik-model-catalog-service.md

- [ ] T107: Migrate the canonical `Dankosik/notification-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `Dankosik/notification-service`.
  - Packet: tasks/T107-dankosik-notification-service.md

- [ ] T108: Migrate the canonical `Dankosik/payments-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `Dankosik/payments-service`.
  - Packet: tasks/T108-dankosik-payments-service.md

- [ ] T109: Migrate the canonical `Dankosik/pricing-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `Dankosik/pricing-service`.
  - Packet: tasks/T109-dankosik-pricing-service.md

- [ ] T110: Migrate the canonical `Dankosik/privacy-sanitization-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `Dankosik/privacy-sanitization-service`.
  - Packet: tasks/T110-dankosik-privacy-sanitization-service.md

- [ ] T111: Migrate the canonical `Dankosik/quota-rate-limit-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `Dankosik/quota-rate-limit-service`.
  - Packet: tasks/T111-dankosik-quota-rate-limit-service.md

- [ ] T112: Migrate the canonical `Dankosik/search-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `Dankosik/search-service`.
  - Packet: tasks/T112-dankosik-search-service.md

- [ ] T113: Migrate the canonical `Dankosik/usage-history-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `Dankosik/usage-history-service`.
  - Packet: tasks/T113-dankosik-usage-history-service.md

- [ ] T114: Migrate the canonical `Dankosik/user-config-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `Dankosik/user-config-service`.
  - Packet: tasks/T114-dankosik-user-config-service.md

- [ ] T115: Migrate the canonical `BitrinaLabs/audit-log-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/audit-log-service`.
  - Packet: tasks/T115-bitrinalabs-audit-log-service.md

- [ ] T116: Migrate the canonical `BitrinaLabs/bff-auth-gateway-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/bff-auth-gateway-service`.
  - Packet: tasks/T116-bitrinalabs-bff-auth-gateway-service.md

- [ ] T117: Migrate the canonical `BitrinaLabs/brand-sidecar-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/brand-sidecar-service`.
  - Packet: tasks/T117-bitrinalabs-brand-sidecar-service.md

- [ ] T118: Migrate the canonical `BitrinaLabs/catalog-search-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/catalog-search-service`.
  - Packet: tasks/T118-bitrinalabs-catalog-search-service.md

- [ ] T119: Migrate the canonical `BitrinaLabs/clickout-attribution-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/clickout-attribution-service`.
  - Packet: tasks/T119-bitrinalabs-clickout-attribution-service.md

- [ ] T120: Migrate the canonical `BitrinaLabs/hosted-page-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/hosted-page-service`.
  - Packet: tasks/T120-bitrinalabs-hosted-page-service.md

- [ ] T121: Migrate the canonical `BitrinaLabs/identity-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/identity-service`.
  - Packet: tasks/T121-bitrinalabs-identity-service.md

- [ ] T122: Migrate the canonical `BitrinaLabs/ledger-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/ledger-service`.
  - Packet: tasks/T122-bitrinalabs-ledger-service.md

- [ ] T123: Migrate the canonical `BitrinaLabs/manual-review-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/manual-review-service`.
  - Packet: tasks/T123-bitrinalabs-manual-review-service.md

- [ ] T124: Migrate the canonical `BitrinaLabs/notification-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/notification-service`.
  - Packet: tasks/T124-bitrinalabs-notification-service.md

- [ ] T125: Migrate the canonical `BitrinaLabs/ops-control-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/ops-control-service`.
  - Packet: tasks/T125-bitrinalabs-ops-control-service.md

- [ ] T126: Migrate the canonical `BitrinaLabs/policy-legal-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/policy-legal-service`.
  - Packet: tasks/T126-bitrinalabs-policy-legal-service.md

- [ ] T127: Migrate the canonical `BitrinaLabs/provider-integration-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/provider-integration-service`.
  - Packet: tasks/T127-bitrinalabs-provider-integration-service.md

- [ ] T128: Migrate the canonical `BitrinaLabs/public-runtime-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/public-runtime-service`.
  - Packet: tasks/T128-bitrinalabs-public-runtime-service.md

- [ ] T129: Migrate the canonical `BitrinaLabs/validation-evidence-service` repository without overwriting repository-owned tooling or ambiguous profile state.
  - Depends on: T001 accepted portable tooling baseline.
  - Provides: one independently accepted candidate or bounded blocked receipt for `BitrinaLabs/validation-evidence-service`.
  - Packet: tasks/T129-bitrinalabs-validation-evidence-service.md

Accepted: T001; evidence: PR #256 and #257 exact-head `required` and `codeql-required` success plus PR #264 classifier repair; candidate: `493e61ae6abc311df89a58d7cba64f407311ff41`

