# Transport Security

The server is disabled until a listener and explicit mode are supplied.

```dotenv
APP__GRPC__SERVER__ENABLED=true
APP__GRPC__SERVER__ADDR=:9091
APP__GRPC__SERVER__TRANSPORT_SECURITY=tls
APP__GRPC__SERVER__TLS__CERT_FILE=/run/secrets/service.crt
APP__GRPC__SERVER__TLS__KEY_FILE=/run/secrets/service.key
```

TLS uses the standard library and a fixed TLS 1.3 floor. Certificate and key
files are loaded and verified before the listener starts. Rotation is a
deployment/provider responsibility; the template does not run a file watcher.

Set `APP__GRPC__SERVER__TLS__CLIENT_CA_FILE` to require and verify a client
certificate against that CA. An empty or invalid CA fails startup.

Plaintext is explicit:

```dotenv
APP__GRPC__SERVER__TRANSPORT_SECURITY=plaintext
```

Use it only where the deployment has accepted that trust boundary. The OIDC
profile requires TLS. Clients must always supply either TLS credentials or
`insecure.NewCredentials()`; there is no implicit client plaintext mode.
