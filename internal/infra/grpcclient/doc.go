// Package grpcclient builds one bounded, instrumented native gRPC connection
// per dependency.
//
// New uses grpc.NewClient, performs no network I/O, and leaves resolution,
// reconnects, pick_first balancing, and transparent retries to grpc-go. Resolver
// service config is disabled so retry, health checking, or another load-balancing
// policy cannot appear without a dependency-owned design. Each RPC still owns
// its deadline, and callers share and close the returned ClientConn.
package grpcclient
