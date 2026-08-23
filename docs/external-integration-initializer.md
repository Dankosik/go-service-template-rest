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
already record the retained transport and, for OAuth, `outbound_auth`.

The first `AUTH=oauth2-client-credentials` invocation that would retire the
singleton `outbound_auth` configuration fails if repository-root `.env` exists.
The command observes only path presence. Move or preserve that file yourself,
run the initializer, then map values to the named `integrations.<name>.oauth.*`
inputs.

A same-identity repeat with an unchanged contract is a no-op. A committed
contract change refreshes generated-only bytes. Manual adapter, config,
bootstrap, record, and documentation files are not rewritten.

The scaffold constructs transport and generated bindings. A later manual
operation in the adapter must own mapping, errors, budget, and retry
eligibility. The initializer does not claim provider compatibility.
