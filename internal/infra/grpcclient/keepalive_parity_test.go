// Two default owners held to one answer: the ping interval this client ships
// against the minimum interval the server half accepts.
//
// They cannot share code — one is a Go constant here, the other a configuration
// default in internal/config — so a value tightened on one side alone breaks
// nothing anyone runs until this file compares them. internal/infra/grpc's
// config_parity_test.go exists for the same reason, and this is the client's
// half of the same obligation.

package grpcclient_test

import (
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
)

// grpcGoClientPingFloor is the interval grpc-go raises anything smaller to. A
// client configured below it does not ping more often; it silently pings at ten
// seconds, which would make a parity margin measured against the configured
// value a fiction.
const grpcGoClientPingFloor = 10 * time.Second

func TestShippedKeepaliveDefaultsAgreeAcrossBothHalves(t *testing.T) {
	client := grpcclient.DefaultConfig("passthrough:///parity")
	server := config.DefaultGRPCServerConfig()

	if client.KeepalivePingInterval < grpcGoClientPingFloor {
		t.Fatalf(
			"client ping interval %s is below grpc-go's %s floor, so it does not ping at the configured value",
			client.KeepalivePingInterval,
			grpcGoClientPingFloor,
		)
	}
	if client.KeepalivePingInterval <= server.MinClientPingInterval {
		t.Fatalf(
			"client pings every %s while the server accepts no faster than %s: the shipped pair disconnects itself",
			client.KeepalivePingInterval,
			server.MinClientPingInterval,
		)
	}
	// The client pings with no active RPC, which is the only way a ping reaches
	// an idle connection. A server that did not permit it would answer those
	// pings with GOAWAY.
	if !server.PermitPingWithoutStream {
		t.Fatal("the server half rejects pings with no active stream, which is exactly when this client sends them")
	}
	if client.KeepalivePingTimeout <= 0 {
		t.Fatalf("client ping timeout = %s, want positive", client.KeepalivePingTimeout)
	}
}
