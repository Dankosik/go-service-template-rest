// Package grpcx owns the small repository policy around a native grpc-go
// server.
//
// grpc-go owns HTTP/2, generated service dispatch, transport bounds, health,
// and transport-stop primitives. otelgrpc owns protocol traces and metrics, and
// Protovalidate owns protobuf constraints. This package retains only policy a
// library cannot choose: process admission, the unary safety deadline, panic
// recovery, authentication/authorization slots, public validation rendering,
// transport-neutral domain-error mapping, bounded method telemetry, readiness
// publication, and shutdown-budget adaptation.
//
// Add a generated service through Options.Services. Feature handlers return
// domain errors rather than transport statuses; the shared failure mappers keep
// HTTP and gRPC answers aligned. Raw handler errors are sanitized, while a
// generated unimplemented method remains a sanitized UNIMPLEMENTED response.
package grpcx
