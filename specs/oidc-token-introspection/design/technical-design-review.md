# Independent Technical Design Review

candidate:

- `system-integration-design.md` sha256 `0c1c8b627cc79c6cac44bb727473e9a05bf37acba248b5ca9e87b47798759b53`
- `go-code-ownership-design.md` sha256 `41e754e723a27356752013abeabc51d0b269b0dfd9137c40401dcae233637523`
- complementary ownership receipt `go-ownership-review.md` sha256 `b35b7c512c2c4e5b23e7fb0c045ebca21412e1c3f6c05df2b231ddf9c9348a2e`

verdict: PASS

findings: none

evidence_boundary: A fresh read-only reviewer reconstructed the fixed
Specification/drivers -> selected shared bearer shell and concrete
introspection engine -> fixed provider flow -> local/provider authorities -> Go
owners -> configuration/bootstrap/initializer inputs -> proving owners. The
review checked current baseline code, dirty-tree contradictions, provider
failure/load/lifecycle behavior, alternatives, physical profile containment,
proof feasibility, and phase/reopen boundaries. The refreshed three-lens Go
Ownership panel independently passed on the same candidate.

The review's only proposed finding claimed that Go 1.27 HTTP/2 ignores
`net/http.Transport.MaxResponseHeaderBytes`. A fresh read-only adjudicator
verified the exact installed Go 1.27 source and invalidated that finding:
HTTP/2 derives its framer header-list bound from `MaxResponseHeaderBytes`, and a
truncated final response is rejected. The skipped HTTP/2 branch in the older
HTTP/1-oriented standard-library test is not evidence that enforcement is
absent. No finding survives adjudication.

The adjusted HTTP/2 header-list accounting remains a Test Design proof
obligation: the later phase should require a real TLS/ALPN HTTP/2 over-limit
final-response falsifier beside `internal/infra/httpclient`. That is a proving
surface already assigned by the design, not an open mechanism or ownership
decision.

A later independent Test Design Review found one missing mandatory input: the
design required `ResponseHeaderTimeout` without fixing its duration. System /
Integration Design reopened only for that value and selected five seconds,
matching the already fixed provider sub-timeout. A fresh read-only bounded
Technical Design Review passed the repaired candidate above. It verified that
shorter caller contexts retain precedence, an otherwise-live provider/header
timeout remains unavailable trust, the direct transport falsifier is now
discriminating, and no behavior, interface, owner, lifecycle, external input,
or risk surface changed. The dirty `httpclient` refactor remained unaccepted
state throughout.

reopen_owner: none

## Review history within Technical Design

The first candidate failed because it selected header bounds without assigning
an accepted current-tree mechanism. System / Integration Design reopened only
for that gap and added the minimal `httpclient.ResponseLimits`/`WithLimits`
surface. The ownership panel was rerun on the expanded fixed candidate before
the final independent Technical Design Review above.

During Test Design, System / Integration Design reopened once more only to fix
the previously omitted response-header timeout duration. The unchanged Go
Ownership receipt remains scoped to the unchanged placement and interface; the
fresh bounded Technical Design Review above independently accepted the value
repair against the complete candidate.

This receipt authorizes movement only to Test Design. It authorizes no Test
Design artifact, Planning, implementation, provider registration, credential,
network, deployment, or other external action.
