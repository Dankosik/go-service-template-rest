# Contract And Generation

Put owned schemas below `api/proto`, use Edition 2023, and give each package a
stable version and `go_package`. `buf.gen.yaml` selects the Go Opaque API; the
schema does not repeat generator policy.

```proto
edition = "2023";

package acme.widgets.v1;

option go_package =
  "github.com/acme/widgets/internal/gen/proto/acme/widgets/v1;widgetsv1";

service WidgetService {
  rpc GetWidget(GetWidgetRequest) returns (GetWidgetResponse);
}

message GetWidgetRequest {
  string id = 1;
}

message GetWidgetResponse {
  string id = 1;
}
```

Generated Go belongs under `internal/gen/proto`, is committed with its schema,
and is never edited manually. Reserve removed field numbers and names, and use
an `UNSPECIFIED` zero enum value.

For semantic request constraints, import `buf/validate/validate.proto`, declare
`buf.build/bufbuild/protovalidate` in `buf.yaml`, commit the generated
`buf.lock`, and annotate the message. The server already validates every unary
request and every streaming receive; validation failures are sanitized
`INVALID_ARGUMENT` statuses with structured violations.

```bash
make proto-format
make proto-generate
make proto-check
BASE_REF=origin/main make proto-breaking
```

Buf v2 owns formatting, `STANDARD`/`COMMENTS` lint, generation, and `FILE`
compatibility. The pinned local Go plugins avoid a system `protoc` dependency.
