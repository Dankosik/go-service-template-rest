# Transport Security

Load for plaintext, TLS, mTLS, trust, or certificate rotation.

The retained capability is still disabled at runtime. To serve locally over
explicit plaintext:

```dotenv
APP__GRPC__SERVER__ENABLED=true
APP__GRPC__SERVER__ADDR=:9091
APP__GRPC__SERVER__TRANSPORT_SECURITY=plaintext
APP__GRPC__SERVER__ALLOW_PLAINTEXT=true
```

For application TLS:

```dotenv
APP__GRPC__SERVER__ENABLED=true
APP__GRPC__SERVER__ADDR=:9091
APP__GRPC__SERVER__TRANSPORT_SECURITY=tls
APP__GRPC__SERVER__TLS__CERT_FILE=/run/secrets/service.crt
APP__GRPC__SERVER__TLS__KEY_FILE=/run/secrets/service.key
```

There is no implicit plaintext mode. TLS files are loaded and the
certificate/key pair is checked before any listener is opened or readiness is
published.

The listener floors at **TLS 1.3**. It is not configurable: every gRPC runtime
this service can serve has supported 1.3 for years, and a knob would exist only
to lower it. A peer that genuinely cannot reach 1.3 is a contract decision, not
a setting.

The certificate pair is **re-read per handshake**, so a renewal that lands while
the process is running is served to the next caller without a restart. The pair
is compared by size and modification time before being read, and the last usable
pair keeps serving whenever the files on disk are not currently a valid pair —
which is what a rotation looks like between writing the certificate and writing
the key. A pair that never becomes valid is visible as one
`grpc_tls_certificate_rejected` record, not one per connection.

Mutual TLS is off unless a trust root is named:

```dotenv
APP__GRPC__SERVER__TLS__CLIENT_CA_FILE=/run/secrets/clients.pem
```

Naming it requires every caller to present a certificate chaining to that file;
there is no request-and-ignore mode. Unlike the pair above it is read once at
startup, because a trust root changes on a deployment's schedule rather than an
issuer's. A file holding no certificate stops the process instead of becoming an
empty pool that rejects every caller at handshake time.

