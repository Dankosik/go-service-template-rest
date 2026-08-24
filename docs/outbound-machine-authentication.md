# Outbound machine authentication

`OUTBOUND_AUTH=oauth2-client-credentials` retains the credential package but
adds no root runtime configuration. Each `make integration-init` invocation
with OAuth owns one disjoint `integrations.<name>.oauth.*` tuple; HTTP is
retained by default and `GRPC=enabled` additionally retains the gRPC adapter.

## Configuration

Each named integration supplies `token_url`, `client_id`, and
`client_secret`; `scopes` is optional. The secret uses only
`APP__INTEGRATIONS__<NAME>__OAUTH__CLIENT_SECRET` and stays empty in
file-backed configuration.

The token path is fixed external HTTPS with Basic client authentication, no
proxy, redirect, retry, or general outbound instrumentation, a five-second
acquisition bound, and finite response limits. A private HTTPS token endpoint
is a dependency-specific code decision.

## Composition

The concrete dependency adapter constructs and closes the credential owner. It
passes only authenticated clients to generated provider clients; feature code
never receives the owner, a token source, or a token.

```go
owner, err := oauth2clientcredentials.New(oauth2clientcredentials.Config{
	TokenURL:     integration.OAuth.TokenURL,
	ClientID:     integration.OAuth.ClientID,
	ClientSecret: integration.OAuth.ClientSecret,
	Scopes:       strings.Fields(integration.OAuth.Scopes),
})

authenticatedHTTP, err := owner.HTTP(resourceHTTP)
providerHTTP, err := generated.NewClient(resourceHTTP.BaseURL(),
	generated.WithHTTPClient(authenticatedHTTP))

authenticatedGRPC, err := owner.GRPC(resourceGRPCConfig, resourceGRPCOptions)
providerGRPC := providerv1.NewProviderClient(authenticatedGRPC)
```

Token acquisition uses `golang.org/x/oauth2/clientcredentials`.
`oauth2.ReuseTokenSourceWithExpiry` owns process-local cache, ten-second
early expiry, and concurrent refresh serialization. The local owner adds only
the bounded provider context, safe Bearer projection, retirement, and transport
cleanup. Provider response bodies, refresh tokens, extra fields, and raw
retrieval errors never cross the package boundary. Resource `401`/`403` and
gRPC auth statuses pass through without token invalidation or replay.

Provider registration, credentials, scopes, resource endpoint and TLS, network
policy, rotation, criticality, capacity, and live compatibility remain
deployment- or dependency-owner work.
