// Package natsjstest starts a JetStream-enabled NATS broker for one test and
// hands back a client already connected to it.
//
// Usage from any package that exercises messaging:
//
//	server := natsjstest.Start(t, natsjstest.WithStreams(jetstream.StreamConfig{
//		Name: "EVENTS", Subjects: []string{"events.>"}, Storage: jetstream.FileStorage,
//	}))
//	client, err := natsjs.Connect(t.Context(), cfg(server.URL), natsjs.RoleWorker, natsjs.Observability{})
//
// A fixture the broker options below cannot express — an authenticated server,
// for instance — builds its own [testcontainers.ContainerRequest] from [Request]
// and still reaches the broker through [Terminate], [Connect], and
// [CreateStreams], so the pinned image and the ready, cleanup, and dial budgets
// stay in one place regardless of how the server itself is configured.
package natsjstest

import (
	"context"
	"net"
	"net/netip"
	"strconv"
	"testing"
	"time"

	containerapi "github.com/moby/moby/api/types/container"
	networkapi "github.com/moby/moby/api/types/network"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// DefaultImage is the NATS image tests run against. It is pinned by digest so a
// suite cannot start passing or failing because a tag moved.
const DefaultImage = "nats:2.14.3-alpine@sha256:c11af972c99ae542de8925e6a7d9c533aa1eb039660420d2074beed6089b3bf0"

// ClientPort is the broker's client port, exposed so a caller building its own
// request can name the same port in a wait strategy or a port binding.
const ClientPort = "4222/tcp"

const (
	startTimeout     = 2 * time.Minute
	readyDeadline    = time.Minute
	terminateTimeout = 30 * time.Second
	connectTimeout   = 5 * time.Second
)

// Server is a running broker and this fixture's own connection to it. Container
// is exposed because a reconnect test stops and restarts it; everything else is
// released by the cleanup [Start] registers.
type Server struct {
	Container testcontainers.Container
	URL       string
	Conn      *nats.Conn
	JS        jetstream.JetStream
}

// Client is a connection to a broker and the JetStream context on it.
type Client struct {
	Conn *nats.Conn
	JS   jetstream.JetStream
}

// Option adjusts what [Start] brings up.
type Option func(*settings)

type settings struct {
	streams       []jetstream.StreamConfig
	fixedHostPort bool
}

// WithStreams creates each stream before Start returns, so a test can publish
// immediately. Streams are created in order and a failure fails the test.
func WithStreams(streams ...jetstream.StreamConfig) Option {
	return func(s *settings) { s.streams = append(s.streams, streams...) }
}

// WithFixedHostPort binds the broker to one reserved host port for the life of
// the test.
//
// Only a test that stops and restarts the container needs it: clients retain the
// URL they were admitted with, and Docker would otherwise publish the restarted
// container on a different host port, so the reconnect under test could never
// succeed. Every other caller lets Docker choose, which is what lets fixtures run
// in parallel.
func WithFixedHostPort() Option {
	return func(s *settings) { s.fixedHostPort = true }
}

// Start brings up a JetStream broker, connects to it, and registers the teardown
// of both.
func Start(t *testing.T, options ...Option) *Server {
	t.Helper()

	var applied settings
	for _, option := range options {
		option(&applied)
	}

	ctx, cancel := context.WithTimeout(t.Context(), startTimeout)
	defer cancel()

	request := Request()
	if applied.fixedHostPort {
		bindHostPort(t, &request)
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start NATS container: %v", err)
	}
	Terminate(t, container)

	url := URL(t, container)
	client := Connect(t, url)
	CreateStreams(t, client.JS, applied.streams...)
	return &Server{Container: container, URL: url, Conn: client.Conn, JS: client.JS}
}

// Request is the broker request Start uses: JetStream on, file storage under
// /data, and ready only once the port listens and the server says so. A caller
// that needs another server configuration starts from this and replaces the
// fields it owns.
func Request() testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
		Image:        DefaultImage,
		ExposedPorts: []string{ClientPort},
		Cmd:          []string{"-js", "-sd", "/data"},
		// Both conditions, because the port listens before JetStream is usable and
		// the log line alone does not prove the port is published yet.
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(ClientPort),
			wait.ForLog("Server is ready"),
		).WithDeadline(readyDeadline),
	}
}

// Terminate registers the container's removal on a context detached from the
// test's, which is already done by the time cleanup runs.
func Terminate(t *testing.T, container testcontainers.Container) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), terminateTimeout)
		defer cancel()
		if err := testcontainers.TerminateContainer(container, testcontainers.StopContext(ctx)); err != nil {
			t.Errorf("terminate NATS container: %v", err)
		}
	})
}

// Endpoint returns the container's published host:port for the client port.
func Endpoint(t *testing.T, container testcontainers.Container) string {
	t.Helper()

	endpoint, err := container.Endpoint(t.Context(), "")
	if err != nil {
		t.Fatalf("resolve NATS endpoint: %v", err)
	}
	return endpoint
}

// URL returns the nats:// URL a client dials the container with.
func URL(t *testing.T, container testcontainers.Container) string {
	t.Helper()

	return "nats://" + Endpoint(t, container)
}

// Connect dials url and opens a JetStream context on it, registering the close.
// Extra options are appended after the dial timeout, so a caller may supply
// credentials or override it.
func Connect(t *testing.T, url string, options ...nats.Option) Client {
	t.Helper()

	conn, err := nats.Connect(url, append([]nats.Option{nats.Timeout(connectTimeout)}, options...)...)
	if err != nil {
		t.Fatalf("connect NATS fixture client: %v", err)
	}
	t.Cleanup(conn.Close)

	js, err := jetstream.New(conn)
	if err != nil {
		t.Fatalf("create fixture JetStream client: %v", err)
	}
	return Client{Conn: conn, JS: js}
}

// CreateStreams creates each stream on the broker.
func CreateStreams(t *testing.T, js jetstream.JetStream, streams ...jetstream.StreamConfig) {
	t.Helper()

	for _, stream := range streams {
		if _, err := js.CreateStream(t.Context(), stream); err != nil {
			t.Fatalf("create stream %s: %v", stream.Name, err)
		}
	}
}

// bindHostPort reserves a free host port and publishes the broker on exactly
// that port. The reservation is released before the container starts, which is
// the only way to learn a free port from the host; the window between is why
// this is opt-in rather than the default.
func bindHostPort(t *testing.T, request *testcontainers.ContainerRequest) {
	t.Helper()

	var config net.ListenConfig
	listener, err := config.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve NATS test port: %v", err)
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		t.Fatalf("reserved NATS test port is not a TCP address: %T", listener.Addr())
	}
	port := address.Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release NATS test port reservation: %v", err)
	}

	request.HostConfigModifier = func(hostConfig *containerapi.HostConfig) {
		hostConfig.PortBindings = networkapi.PortMap{
			networkapi.MustParsePort(ClientPort): {
				{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: strconv.Itoa(port)},
			},
		}
	}
}
