# Outbound machine authentication

`OUTBOUND_AUTH=oauth2-client-credentials` retains a client factory for one
OAuth 2.0 client-credentials dependency. Select it with
`OUTBOUND_HTTP=bounded`, `GRPC=enabled`, or both.

## Configuration

Supply `token_url`, `client_id`, and `client_secret`; `scopes` is optional.
Some providers additionally require exactly one of `audience` or `resource`.
The secret is supplied only through `APP__OUTBOUND_AUTH__CLIENT_SECRET` and
remains empty in file-backed configuration.

The default token path is fixed external HTTPS with Basic client
authentication, no proxy, redirect, retry, or general outbound
instrumentation, a five-second acquisition bound, and finite response limits.
A private HTTPS token endpoint is a dependency-specific code decision.

## Composition

The concrete dependency adapter constructs and closes the credential owner. It
passes only authenticated clients to generated provider clients; feature code
never receives the owner, a token source, or a token.

```go
owner, err := oauth2clientcredentials.New(oauth2clientcredentials.Config{
	TokenURL:     cfg.OutboundAuth.TokenURL,
	ClientID:     cfg.OutboundAuth.ClientID,
	ClientSecret: cfg.OutboundAuth.ClientSecret,
	Scopes:       strings.Fields(cfg.OutboundAuth.Scopes),
	Audience:     cfg.OutboundAuth.Audience,
	Resource:     cfg.OutboundAuth.Resource,
})

authenticatedHTTP, err := owner.HTTP(resourceHTTP)
providerHTTP, err := generated.NewClient(resourceHTTP.BaseURL(),
	generated.WithHTTPClient(authenticatedHTTP))

authenticatedGRPC, err := owner.GRPC(resourceGRPCConfig, resourceGRPCOptions)
providerGRPC := providerv1.NewProviderClient(authenticatedGRPC)
```

Token acquisition uses `golang.org/x/oauth2/clientcredentials`. The local owner
keeps only one usable expiring Bearer token, shares one provider request, lets
each caller cancel its own wait, and joins provider work during `Close`.
Provider response bodies, refresh tokens, extra fields, and raw retrieval
errors never cross the package boundary. Resource `401`/`403` and gRPC auth
statuses pass through without token invalidation or replay.

Provider registration, credentials, scopes/audience/resource, resource
endpoint and TLS, network policy, rotation, criticality, capacity, and live
compatibility remain deployment- or dependency-owner work.
