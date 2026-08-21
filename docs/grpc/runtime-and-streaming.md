# Runtime And Streaming

Embed the generated unimplemented server and implement generated methods
directly:

```go
type Server struct {
    widgetsv1.UnimplementedWidgetServiceServer
    widgets *widget.Service
}

func (s *Server) GetWidget(
    ctx context.Context,
    request *widgetsv1.GetWidgetRequest,
) (*widgetsv1.GetWidgetResponse, error) {
    value, err := s.widgets.Get(ctx, request.GetId())
    if err != nil {
        return nil, err
    }
    id := value.ID
    return widgetsv1.GetWidgetResponse_builder{Id: &id}.Build(), nil
}
```

Return feature/domain errors. The shared mapper renders their gRPC status and
structured error details consistently with HTTP. Unknown handler errors and raw
handler statuses are sanitized; generated unimplemented methods remain a safe
`UNIMPLEMENTED` response.

Register once in `serviceGRPCBindings`:

```go
bindings.Services = append(bindings.Services, func(r grpc.ServiceRegistrar) {
    widgetsv1.RegisterWidgetServiceServer(r, widgetGRPC)
})
```

The server applies, with unary/stream parity: panic recovery, process/health
admission, supplied authentication/authorization, Protovalidate, error privacy,
official OpenTelemetry protocol instrumentation, and a fixed unary safety
deadline. A feature adds none of these.

Streams validate every received message. The feature still owns aggregate
memory, idle/duration semantics, cancellation compliance, and concurrent I/O:
one goroutine may read while another writes, but concurrent reads or concurrent
writes on the same stream are unsupported.

Build one client connection per dependency:

```go
conn, err := grpcclient.New(
    grpcclient.DefaultConfig("dns:///widgets.internal:9091"),
    grpcclient.Options{TransportCredentials: transportCredentials},
)
client := widgetsv1.NewWidgetServiceClient(conn)
```

The connection performs no startup I/O and is safe to share. grpc-go owns DNS,
reconnects, `pick_first`, and transparent retry. Resolver service config is
disabled, so application retry, client health, or another balancing policy
requires a separate dependency decision. Each call propagates or sets its
business deadline.
