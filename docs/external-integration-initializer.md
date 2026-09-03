# External Integration Initializer

`make integration-init` adds one named outbound HTTP or gRPC integration to an
already initialized service. It is build-time only. It does not contact a
provider, read `.env`, or invent a business operation.

```bash
make integration-init \
  NAME=billing \
  TRANSPORT=http \
  CONTRACT=api/external/billing/openapi.yaml \
  TARGET=external-https \
  AUTH=none
```

```bash
make integration-init \
  NAME=identity \
  TRANSPORT=grpc \
  CONTRACT=api/proto/external/identity/identity.proto \
  AUTH=oauth2-client-credentials
```

`TARGET` is required for HTTP and rejected for gRPC. Unknown command-line
variables are rejected. The worktree must be clean and `template.lock` must
have `state = "complete"` and already record the retained transport and, for
OAuth, `outbound_auth`.
`NAME` is a lower-case Go package identifier; `__` is rejected because that
sequence is reserved by the `APP__...` environment-key delimiter.
Retaining that profile keeps only the reusable credential package; each
initializer invocation creates its named `integrations.<name>.oauth.*` tuple
directly. Repository-root `.env` is never an initializer input or precondition.

A same-identity repeat with an unchanged contract is a no-op. A committed
contract change refreshes generated-only bytes. Manual adapter, config,
bootstrap, record, and documentation files are not rewritten.
The integration record must remain byte-exact; schema, generator source, extra
keys, duplicate keys, and symlinks fail closed. `make integration-record-check`
verifies record/source/output parity independently of a refresh invocation.
HTTP contracts may use document-local `#...` references. External file and URI
references are rejected before validation or generation so the initializer
cannot fetch contract material.

The scaffold constructs transport and generated bindings. Its `Config.Limits`
must supply positive response-header timeout, response-header bytes,
request-concurrency, and decoded-body ceilings; zero values fail construction
before provider I/O. A later manual operation in the adapter must own mapping,
errors, any smaller deadline/body policy, and retry eligibility. Streaming is a
separate provider decision. The initializer does not claim provider
compatibility.
