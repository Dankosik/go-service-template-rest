# Technical Design transition

status: ready
owner: Technical Design
result: `specs/external-integration-initializer/design/overview.md` SHA-256 `ad02cc02cd79dae097850eb241cb8d0f04ce8ee399fc5b2882a68b0255d3c2ac`
review: `specs/external-integration-initializer/design/review.md` SHA-256 `d80db1713d1118e2345b5cb3297b842f6bf9c21e5ca2d7d04a960a5d4dbd2639` — fresh independent Technical Design Review `PASS`
movement_evidence: The reopened System / Integration Design now classifies the first singleton-retiring OAuth initial mode before staging, admits only an exact root `.env` presence bit before staging and caller patch application, and leaves the ignored path outside all snapshots, patches, rollback, and cleanup. The affected Go Ownership Design retains no `.env` reader or custody owner and assigns stale legacy-key rejection to existing `internal/config`. All three ownership lenses passed, and fresh independent review found no surviving material trace or ownership divergence.
reopen_owner: Technical Design
next_owner: Test Design

This revision preserves every unaffected decision from prior design candidate
`a239cc57100fcf35576b5bc687718596d77e678b16797a48de462b8c765a91d7`.
Specification reopens only if its fixed fail-closed behavior, developer or
operator custody boundary, byte-restoration rules, or initial/refresh
consequences cannot be realized without changing accepted behavior or
authority. A mechanism or placement contradiction reopens only the affected
Technical Design owner.

Test Design is the next owner because the eight accepted proof expectations
require a deterministic scenario matrix. It remains unopened in this receipt;
Planning, Implementation, provider/network/deployment actions, and all other
external effects also remain unopened.
