# Outbound machine authentication

`OUTBOUND_AUTH=oauth2-client-credentials` retains one fixed OAuth 2.0
client-credentials dependency. Select it only with `OUTBOUND_HTTP=bounded`,
`GRPC=enabled`, or both.

Supply one immutable `outbound_auth` configuration tuple, including a fixed
HTTPS token endpoint and resource authority. `client_secret` is supplied only
through `APP__OUTBOUND_AUTH__CLIENT_SECRET`; every file example stays empty.
Restart to apply any credential, endpoint, scope, resource, or audience change.

The credential owner obtains and reuses opaque bearer tokens only for its fixed
dependency. HTTP and gRPC adapters attach them without exposing token or
provider APIs to feature code. Acquisition failures are sanitized and the
dependency remains optional for startup, readiness, and liveness.

Provider registration, credentials, scopes, network policy, rotation, and live
compatibility are deployment-owner work. The template proves only its portable
client contract; it does not certify a provider or deployment path.
