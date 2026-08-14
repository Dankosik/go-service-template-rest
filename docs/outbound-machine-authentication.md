# Outbound machine authentication

`OUTBOUND_AUTH=oauth2-client-credentials` retains one fixed OAuth 2.0
client-credentials dependency. Select it only with `OUTBOUND_HTTP=bounded`,
`GRPC=enabled`, or both.

Supply one immutable `outbound_auth` configuration tuple, including a fixed
HTTPS token endpoint and resource authority. `client_secret` is supplied only
through `APP__OUTBOUND_AUTH__CLIENT_SECRET`; every file example stays empty.
Restart to apply any credential, endpoint, scope, resource, or audience change.

The credential owner obtains and reuses opaque bearer tokens only for its fixed
dependency. HTTP and gRPC clients attach them without exposing token or
provider APIs to feature code. Token and resource authorities are independently
fixed HTTPS origins with normal certificate and hostname verification, bounded
DNS/address policy, no proxy, and no redirects. `client_secret_basic` is built
from the form-escaped client ID and secret and is sent only to the admitted
token endpoint.

The portable response contract is intentionally strict: exact JSON `200`, an
opaque bounded Bearer token, integer `expires_in` of at most one hour, a
ten-second early-expiry margin, no non-empty refresh token, and exact
case-sensitive returned-scope set equality when `scope` is present. `resource`
is the standard RFC 8707 parameter; `audience` is provider-contract-specific.
Provider ownership must confirm which one is accepted, whether an omitted scope
means the registration's exact least-privilege default, and whether the exact
scope response rule is compatible.

Acquisition is one attempt with no automatic retry. Concurrent callers share
that attempt and its result. After a failed attempt completes, the process fails
fast for at least the acquisition budget or one second; a valid `Retry-After`
on `429` or `503` can extend that delay. Positive jitter prevents synchronized
process-local recovery, and the delay is capped at one hour. This is not a
fleet-wide quota or circuit breaker: replica count, provider limits, and
degraded-operation policy remain deployment obligations.

For HTTP, pass the result of `NewHTTPClient` to the concrete or generated
client. It attaches credentials inside each already-authorized resource attempt
without adding retries or replaying an unsafe operation. For gRPC, construct the
connection only through `NewGRPCClient`:

```go
connection, err := oauth2clientcredentials.NewGRPCClient(
    authOwner,
    grpcclient.DefaultConfig("dns:///orders.internal:443"),
    grpcclient.Options{TransportCredentials: tlsCredentials},
)
if err != nil {
    return err
}
// The dependency owner closes connection during process shutdown.
client := ordersv1.NewOrdersServiceClient(connection)
```

That constructor installs the same credential on application RPCs and grpc-go
control RPCs such as `Health/Watch`, rejects competing credential/observer
wiring, and records terminal `Unauthenticated`/`PermissionDenied` results once.
Those results are not translated into token refresh or replay. A long-lived
stream authenticates only when it is created; generic code neither mutates its
metadata after expiry nor claims reconnect, resume, or replay continuity.

Acquisition failures are sanitized and the dependency remains optional for
startup, readiness, and liveness. Secrets, bearer values, token responses,
provider details, scopes, resource/audience values, and endpoint identities are
excluded from logs, traces, metrics, and returned errors. Rotation is a new
process: immutable Go strings and transport buffers are not claimed to be
zeroized in place.

This profile is the interoperable confidential-client floor, not the strongest
available deployment posture. Prefer workload identity, asymmetric client
authentication, and sender-constrained tokens when the authorization server,
resource server, and deployment support them; those choices reopen this
profile's credential and transport boundary rather than becoming dormant flags.

Provider registration, credentials, scopes, network policy, rotation, and live
compatibility are deployment-owner work. So are discovery/metadata inspection,
grant and authentication-method support, TLS trust roots, quotas and
`Retry-After` behavior, token lifetime/revocation, resource authorization,
long-stream expiry, and capacity. The template proves only its portable client
contract; it does not certify a provider or deployment path.
