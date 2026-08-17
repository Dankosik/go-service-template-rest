package oauth2clientcredentials

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	testGRPCHost      = "outbound-auth-grpc.test.internal"
	testUnaryMethod   = "/outbound.auth.test.Service/Unary"
	testServerStream  = "/outbound.auth.test.Service/ServerStream"
	testClientStream  = "/outbound.auth.test.Service/ClientStream"
	testBidiStream    = "/outbound.auth.test.Service/BidiStream"
	grpcBearerToken   = "grpc-operation-token"
	secondBearerToken = "grpc-reconnected-token"
)

func TestGRPCResourceAuthorityIsFixed(t *testing.T) {
	t.Parallel()
	clock := newMovableClock(fixedProviderTime)
	provider := &scriptedAcquirer{steps: []acquisitionStep{{token: grpcTestToken(clock)}}}
	client := requireTestClient(t, validTestConfig(), testClientOptions{now: clock.Now, acquire: provider.acquire})
	auth, err := NewGRPC(client)
	if err != nil {
		t.Fatalf("NewGRPC() error = %v", err)
	}

	metadataValues, err := auth.GetRequestMetadata(t.Context(), client.config.ResourceAuthority+testUnaryMethod)
	if err != nil {
		t.Fatalf("GetRequestMetadata() error = %v", err)
	}
	if got := metadataValues["authorization"]; got != "Bearer "+grpcBearerToken {
		t.Fatalf("authorization = %q, want one bearer", got)
	}

	for _, requestURI := range []string{
		"https://other.example.com" + testUnaryMethod,
		"http://payments.example.com" + testUnaryMethod,
		client.config.ResourceAuthority + ".attacker.test" + testUnaryMethod,
	} {
		values, mismatchErr := auth.GetRequestMetadata(t.Context(), requestURI)
		assertFailureClass(t, mismatchErr, FailureInvalidConfiguration)
		if values != nil {
			t.Fatalf("GetRequestMetadata(%q) = %v, want nil", requestURI, values)
		}
	}
	if got := provider.Calls(); got != 1 {
		t.Fatalf("provider calls = %d, want 1", got)
	}

	t.Run("explicit default port matches grpc-go authority", func(t *testing.T) {
		t.Parallel()
		fixture := newGRPCFixtureWithResourceAuthority(t, nil, nil, "https://"+testGRPCHost+":443")
		if err := fixture.application.Invoke(t.Context(), testUnaryMethod, &emptypb.Empty{}, &emptypb.Empty{}); err != nil {
			t.Fatalf("Invoke() error = %v", err)
		}
		watchCtx, cancelWatch := context.WithCancel(t.Context())
		watch, err := healthgrpc.NewHealthClient(fixture.connection).Watch(watchCtx, &healthgrpc.HealthCheckRequest{})
		if err != nil {
			t.Fatalf("Health.Watch() error = %v", err)
		}
		if _, err := watch.Recv(); err != nil {
			t.Fatalf("Health.Watch().Recv() error = %v", err)
		}
		cancelWatch()
	})
}

func TestGRPCApplicationCallsAttachOneToken(t *testing.T) {
	t.Parallel()
	fixture := newGRPCFixture(t, nil, nil)

	if err := fixture.application.Invoke(t.Context(), testUnaryMethod, &emptypb.Empty{}, &emptypb.Empty{}); err != nil {
		t.Fatalf("Invoke() error = %v; credential URI = %q", err, fixture.credential.last.Load())
	}
	exerciseGRPCStream(t, fixture.application, &grpc.StreamDesc{ServerStreams: true}, testServerStream)
	exerciseGRPCStream(t, fixture.application, &grpc.StreamDesc{ClientStreams: true}, testClientStream)
	exerciseGRPCStream(t, fixture.application, &grpc.StreamDesc{ClientStreams: true, ServerStreams: true}, testBidiStream)

	for range 4 {
		assertOneBearer(t, <-fixture.service.metadata, grpcBearerToken)
	}
	if got := fixture.provider.Calls(); got != 1 {
		t.Fatalf("provider calls = %d, want 1", got)
	}
	if got := fixture.credential.calls.Load(); got != 4 {
		t.Fatalf("request metadata calls = %d, want 4", got)
	}
}

func TestGRPCRejectsCompetingAuthorization(t *testing.T) {
	t.Parallel()
	fixture := newGRPCFixture(t, nil, nil)
	ctx := metadata.NewOutgoingContext(
		t.Context(),
		metadata.MD{"AuThOrIzAtIoN": []string{"Bearer caller"}},
	)
	err := fixture.application.Invoke(ctx, testUnaryMethod, &emptypb.Empty{}, &emptypb.Empty{})
	assertFailureClass(t, err, FailureInvalidConfiguration)
	err = fixture.application.Invoke(
		t.Context(),
		testUnaryMethod,
		&emptypb.Empty{},
		&emptypb.Empty{},
		grpc.PerRPCCredentials(staticTestCredential{}),
	)
	assertFailureClass(t, err, FailureInvalidConfiguration)
	if got := fixture.provider.Calls(); got != 0 {
		t.Fatalf("provider calls = %d, want 0", got)
	}
	if got := fixture.service.calls.Load(); got != 0 {
		t.Fatalf("handler calls = %d, want 0", got)
	}
}

func TestGRPCCallerCancellationStopsApplicationWait(t *testing.T) {
	t.Parallel()
	entered := make(chan struct{})
	release := make(chan struct{})
	clock := newMovableClock(fixedProviderTime)
	provider := &scriptedAcquirer{steps: []acquisitionStep{{
		token:              grpcTestToken(clock),
		entered:            entered,
		release:            release,
		ignoreCancellation: true,
	}}}
	fixture := newGRPCFixture(t, clock, provider)

	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() {
		_, err := fixture.application.NewStream(ctx, &grpc.StreamDesc{ServerStreams: true}, testServerStream)
		result <- err
	}()
	<-entered
	cancel()
	assertFailureClass(t, <-result, FailureCallerCanceled)
	if got := fixture.service.calls.Load(); got != 0 {
		t.Fatalf("handler calls before provider release = %d, want 0", got)
	}

	liveResult := make(chan error, 1)
	go func() {
		liveResult <- fixture.application.Invoke(t.Context(), testUnaryMethod, &emptypb.Empty{}, &emptypb.Empty{})
	}()
	close(release)
	if err := <-liveResult; err != nil {
		t.Fatalf("live Invoke() error = %v", err)
	}
	assertOneBearer(t, <-fixture.service.metadata, grpcBearerToken)
	if got := provider.Calls(); got != 1 {
		t.Fatalf("provider calls = %d, want 1", got)
	}

	t.Run("provider failure reaches no handler", func(t *testing.T) {
		t.Parallel()
		failed := newGRPCFixture(t, nil, &scriptedAcquirer{steps: []acquisitionStep{{err: failure(FailureProviderUnavailable)}}})
		err := failed.application.Invoke(t.Context(), testUnaryMethod, &emptypb.Empty{}, &emptypb.Empty{})
		assertFailureClass(t, err, FailureProviderUnavailable)
		if got := failed.service.calls.Load(); got != 0 {
			t.Fatalf("handler calls = %d, want 0", got)
		}
	})
}

func TestGRPCRequiresTransportSecurity(t *testing.T) {
	t.Parallel()
	client := requireTestClient(t, validTestConfig(), testClientOptions{acquire: func(context.Context) (accessToken, error) {
		return accessToken{}, failure(FailureProviderUnavailable)
	}})
	auth, err := NewGRPC(client)
	if err != nil {
		t.Fatalf("NewGRPC() error = %v", err)
	}
	if !auth.RequireTransportSecurity() {
		t.Fatal("RequireTransportSecurity() = false, want true")
	}
	_, err = grpcclient.New(grpcclient.DefaultConfig("passthrough:///127.0.0.1:1"), grpcclient.Options{
		TransportCredentials: insecure.NewCredentials(),
		PerRPCCredentials:    auth,
	})
	if err == nil {
		t.Fatal("grpcclient.New() with insecure transport error = nil")
	}
}

func TestGRPCPreservesDownstreamAuthStatus(t *testing.T) {
	t.Parallel()
	assertGRPCDownstreamAuthStatus(t, nil)
}

func assertGRPCDownstreamAuthStatus(t *testing.T, meterProvider metric.MeterProvider) {
	t.Helper()
	fixture := newGRPCFixtureWithMeter(t, nil, nil, meterProvider)
	for _, code := range []codes.Code{codes.Unauthenticated, codes.PermissionDenied} {
		fixture.service.unaryCode.Store(int32(code))
		err := fixture.application.Invoke(t.Context(), testUnaryMethod, &emptypb.Empty{}, &emptypb.Empty{})
		if got := status.Code(err); got != code {
			t.Fatalf("Invoke() code = %s, want %s: %v", got, code, err)
		}
	}
	if got := fixture.service.calls.Load(); got != 2 {
		t.Fatalf("handler calls = %d, want 2", got)
	}
	if got := fixture.provider.Calls(); got != 1 {
		t.Fatalf("provider calls = %d, want 1", got)
	}
}

func TestGRPCAttemptsUseOneOperationToken(t *testing.T) {
	t.Parallel()
	t.Run("transparent retry reuses one token", func(t *testing.T) {
		t.Parallel()
		clock := newMovableClock(fixedProviderTime)
		provider := &scriptedAcquirer{steps: []acquisitionStep{{token: grpcTestToken(clock)}}}
		peer := startRawGRPCPeer(t, nil)
		application, credential := newRawGRPCApplication(t, peer, clock, provider)

		if err := application.Invoke(t.Context(), testUnaryMethod, &emptypb.Empty{}, &emptypb.Empty{}); err != nil {
			t.Fatalf("Invoke() error = %v", err)
		}
		attempts := <-peer.attempts
		if len(attempts) != 2 {
			t.Fatalf("wire attempts = %d, want 2", len(attempts))
		}
		for _, attempt := range attempts {
			assertOneBearer(t, attempt, grpcBearerToken)
		}
		if got := provider.Calls(); got != 1 {
			t.Fatalf("provider calls = %d, want 1", got)
		}
		if got := credential.calls.Load(); got != 2 {
			t.Fatalf("request metadata calls = %d, want 2", got)
		}
	})

	t.Run("margin stops before second headers", func(t *testing.T) {
		t.Parallel()
		clock := newMovableClock(fixedProviderTime)
		provider := &scriptedAcquirer{steps: []acquisitionStep{{token: grpcTestToken(clock)}}}
		releaseFirst := make(chan struct{})
		peer := startRawGRPCPeer(t, releaseFirst)
		application, credential := newRawGRPCApplication(t, peer, clock, provider)

		result := make(chan error, 1)
		go func() {
			result <- application.Invoke(t.Context(), testUnaryMethod, &emptypb.Empty{}, &emptypb.Empty{})
		}()
		<-peer.firstAttempt
		clock.Advance(25 * time.Second)
		close(releaseFirst)
		err := <-result
		assertFailureClass(t, err, FailureTokenUnusable)
		if got := peer.headers.Load(); got != 1 {
			t.Fatalf("wire HEADERS = %d, want 1", got)
		}
		if got := provider.Calls(); got != 1 {
			t.Fatalf("provider calls = %d, want 1", got)
		}
		if got := credential.calls.Load(); got != 2 {
			t.Fatalf("request metadata calls = %d, want 2", got)
		}
	})

	t.Run("streaming margin returns local failure", func(t *testing.T) {
		t.Parallel()
		clock := newMovableClock(fixedProviderTime)
		provider := &scriptedAcquirer{steps: []acquisitionStep{{token: grpcTestToken(clock)}}}
		releaseFirst := make(chan struct{})
		peer := startRawGRPCPeer(t, releaseFirst)
		application, credential := newRawGRPCApplication(t, peer, clock, provider)
		stream, err := application.NewStream(
			t.Context(),
			&grpc.StreamDesc{ServerStreams: true},
			testServerStream,
		)
		if err != nil {
			t.Fatalf("NewStream() error = %v", err)
		}
		result := make(chan error, 1)
		go func() {
			result <- stream.RecvMsg(&emptypb.Empty{})
		}()
		<-peer.firstAttempt
		clock.Advance(25 * time.Second)
		close(releaseFirst)
		assertFailureClass(t, <-result, FailureTokenUnusable)
		if got := peer.headers.Load(); got != 1 {
			t.Fatalf("wire HEADERS = %d, want 1", got)
		}
		if got := provider.Calls(); got != 1 {
			t.Fatalf("provider calls = %d, want 1", got)
		}
		if got := credential.calls.Load(); got != 2 {
			t.Fatalf("request metadata calls = %d, want 2", got)
		}
	})
}

func TestGRPCControlStreamCancellationStopsOnlyItsWait(t *testing.T) {
	t.Parallel()
	entered := make(chan struct{})
	release := make(chan struct{})
	clock := newMovableClock(fixedProviderTime)
	provider := &scriptedAcquirer{steps: []acquisitionStep{{
		token:              grpcTestToken(clock),
		entered:            entered,
		release:            release,
		ignoreCancellation: true,
	}}}
	fixture := newGRPCFixture(t, clock, provider)
	health := healthgrpc.NewHealthClient(fixture.connection)
	watchCtx, cancelWatch := context.WithCancel(t.Context())
	watchResult := make(chan error, 1)
	go func() {
		watch, watchErr := health.Watch(watchCtx, &healthgrpc.HealthCheckRequest{})
		if watchErr == nil {
			_, watchErr = watch.Recv()
		}
		watchResult <- watchErr
	}()
	<-entered
	liveResult := make(chan error, 1)
	go func() {
		liveResult <- fixture.application.Invoke(t.Context(), testUnaryMethod, &emptypb.Empty{}, &emptypb.Empty{})
	}()
	cancelWatch()
	watchErr := <-watchResult
	close(release)
	if !errors.Is(watchErr, context.Canceled) && status.Code(watchErr) != codes.Canceled {
		t.Fatalf("Health.Watch() error = %v, want caller cancellation", watchErr)
	}
	if got := fixture.service.healthCalls.Load(); got != 0 {
		t.Fatalf("health handler calls = %d, want 0", got)
	}

	if err := <-liveResult; err != nil {
		t.Fatalf("live Invoke() error = %v", err)
	}
	if got := provider.Calls(); got != 1 {
		t.Fatalf("provider calls = %d, want 1", got)
	}

	t.Run("cancellation sends no headers", func(t *testing.T) {
		t.Parallel()
		entered := make(chan struct{})
		release := make(chan struct{})
		clock := newMovableClock(fixedProviderTime)
		provider := &scriptedAcquirer{steps: []acquisitionStep{{
			token:              grpcTestToken(clock),
			entered:            entered,
			release:            release,
			ignoreCancellation: true,
		}}}
		peer := startRawGRPCPeer(t, nil)
		connection, _ := newRawGRPCConnection(t, peer, clock, provider)
		ctx, cancel := context.WithCancel(t.Context())
		result := make(chan error, 1)
		go func() {
			watch, err := healthgrpc.NewHealthClient(connection).Watch(ctx, &healthgrpc.HealthCheckRequest{})
			if err == nil {
				_, err = watch.Recv()
			}
			result <- err
		}()
		<-entered
		cancel()
		if err := <-result; status.Code(err) != codes.Canceled {
			t.Fatalf("Health.Watch() code = %s, want Canceled: %v", status.Code(err), err)
		}
		if got := peer.headers.Load(); got != 0 {
			t.Fatalf("wire HEADERS = %d, want 0", got)
		}
		close(release)
	})
}

func TestGRPCHealthWatchUsesConnectionCredentialOnReconnect(t *testing.T) {
	t.Parallel()
	clock := newMovableClock(fixedProviderTime)
	provider := &scriptedAcquirer{steps: []acquisitionStep{
		{token: grpcTestToken(clock)},
		{token: accessToken{value: secondBearerToken, expiresAt: fixedProviderTime.Add(2 * time.Minute)}},
	}}
	fixture := newGRPCFixture(t, clock, provider)
	health := healthgrpc.NewHealthClient(fixture.connection)

	firstCtx, cancelFirst := context.WithCancel(t.Context())
	first, err := health.Watch(firstCtx, &healthgrpc.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("first Health.Watch() error = %v", err)
	}
	if _, err := first.Recv(); err != nil {
		t.Fatalf("first Health.Watch().Recv() error = %v", err)
	}
	assertOneBearer(t, <-fixture.service.healthMetadata, grpcBearerToken)
	cancelFirst()

	clock.Advance(time.Minute)
	secondCtx, cancelSecond := context.WithCancel(t.Context())
	defer cancelSecond()
	second, err := health.Watch(secondCtx, &healthgrpc.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("second Health.Watch() error = %v", err)
	}
	if _, err := second.Recv(); err != nil {
		t.Fatalf("second Health.Watch().Recv() error = %v", err)
	}
	assertOneBearer(t, <-fixture.service.healthMetadata, secondBearerToken)
	if got := provider.Calls(); got != 2 {
		t.Fatalf("provider calls = %d, want 2", got)
	}
}

func TestLongLivedStreamDoesNotReauthenticateInPlace(t *testing.T) {
	t.Parallel()
	clock := newMovableClock(fixedProviderTime)
	provider := &scriptedAcquirer{steps: []acquisitionStep{{token: grpcTestToken(clock)}}}
	fixture := newGRPCFixture(t, clock, provider)
	fixture.service.longStream.Store(true)

	stream, err := fixture.application.NewStream(
		t.Context(),
		&grpc.StreamDesc{ClientStreams: true, ServerStreams: true},
		testBidiStream,
	)
	if err != nil {
		t.Fatalf("NewStream() error = %v", err)
	}
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		t.Fatalf("first SendMsg() error = %v", err)
	}
	if err := stream.RecvMsg(&emptypb.Empty{}); err != nil {
		t.Fatalf("first RecvMsg() error = %v", err)
	}
	assertOneBearer(t, <-fixture.service.metadata, grpcBearerToken)

	clock.Advance(time.Minute)
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		t.Fatalf("second SendMsg() error = %v", err)
	}
	if err := stream.RecvMsg(&emptypb.Empty{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("second RecvMsg() error = %v, want Unauthenticated", err)
	}
	if got := provider.Calls(); got != 1 {
		t.Fatalf("provider calls = %d, want 1", got)
	}
	if got := fixture.credential.calls.Load(); got != 1 {
		t.Fatalf("request metadata calls = %d, want 1", got)
	}
}

type rawGRPCPeer struct {
	target       string
	firstAttempt <-chan struct{}
	attempts     <-chan []metadata.MD
	headers      atomic.Int64
}

func startRawGRPCPeer(t *testing.T, releaseFirst <-chan struct{}) *rawGRPCPeer {
	t.Helper()
	address := privateTestAddress(t)
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", net.JoinHostPort(address.String(), "0"))
	if err != nil {
		t.Fatalf("listen for raw gRPC peer: %v", err)
	}
	certificate, err := grpcTestPKI().certificateFor(testGRPCHost)
	if err != nil {
		t.Fatal(err)
	}
	tlsListener := tls.NewListener(listener, serverTLSConfig(certificate))
	firstAttempt := make(chan struct{})
	attempts := make(chan []metadata.MD, 1)
	peer := &rawGRPCPeer{firstAttempt: firstAttempt, attempts: attempts}
	done := make(chan error, 1)
	go func() {
		connection, acceptErr := tlsListener.Accept()
		if acceptErr != nil {
			done <- acceptErr
			return
		}
		serveErr := serveRawGRPCConnection(connection, peer, firstAttempt, releaseFirst, attempts)
		closeErr := connection.Close()
		if serveErr != nil {
			done <- serveErr
			return
		}
		done <- closeErr
	}()
	t.Cleanup(func() {
		if err := tlsListener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("raw gRPC listener Close() error = %v", err)
		}
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) && !errors.Is(err, syscall.ECONNRESET) {
				t.Errorf("raw gRPC peer error = %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("raw gRPC peer did not stop")
		}
	})
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split raw gRPC listener address: %v", err)
	}
	installPrivateTestResolver(t, map[string]netip.Addr{testGRPCHost: address})
	peer.target = "dns:///" + net.JoinHostPort(testGRPCHost, port)
	return peer
}

func newRawGRPCApplication( //nolint:ireturn // The test drives the generated-client interface boundary.
	t *testing.T,
	peer *rawGRPCPeer,
	clock *movableClock,
	provider *scriptedAcquirer,
) (grpc.ClientConnInterface, *countingCredential) {
	t.Helper()
	connection, counting := newRawGRPCConnection(t, peer, clock, provider)
	auth, ok := counting.base.(*GRPC)
	if !ok {
		t.Fatal("raw gRPC credential is not *GRPC")
	}
	application, err := auth.Wrap(connection)
	if err != nil {
		t.Fatalf("GRPC.Wrap() error = %v", err)
	}
	return application, counting
}

func newRawGRPCConnection(
	t *testing.T,
	peer *rawGRPCPeer,
	clock *movableClock,
	provider *scriptedAcquirer,
) (*grpc.ClientConn, *countingCredential) {
	t.Helper()
	cfg := validTestConfig()
	cfg.ResourceAuthority = "https://" + testGRPCHost
	client := requireTestClient(t, cfg, testClientOptions{now: clock.Now, acquire: provider.acquire})
	auth, err := NewGRPC(client)
	if err != nil {
		t.Fatalf("NewGRPC() error = %v", err)
	}
	counting := &countingCredential{base: auth}
	clientConfig := grpcclient.DefaultConfig(peer.target)
	clientConfig.HealthCheck = false
	connection, err := grpcclient.New(clientConfig, grpcclient.Options{
		TransportCredentials: credentials.NewTLS(tlsConfigForClient(testGRPCHost)),
		PerRPCCredentials:    counting,
	})
	if err != nil {
		t.Fatalf("grpcclient.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("ClientConn.Close() error = %v", err)
		}
	})
	return connection, counting
}

func serveRawGRPCConnection(
	connection net.Conn,
	peer *rawGRPCPeer,
	firstAttempt chan<- struct{},
	releaseFirst <-chan struct{},
	result chan<- []metadata.MD,
) error {
	if _, err := io.ReadFull(connection, make([]byte, len(http2.ClientPreface))); err != nil {
		return fmt.Errorf("read HTTP/2 client preface: %w", err)
	}
	framer := http2.NewFramer(connection, connection)
	framer.ReadMetaHeaders = hpack.NewDecoder(4096, nil)
	if err := framer.WriteSettings(); err != nil {
		return fmt.Errorf("write server settings: %w", err)
	}
	captured := make([]metadata.MD, 0, 2)
	for {
		frame, err := framer.ReadFrame()
		if err != nil {
			return fmt.Errorf("read client frame: %w", err)
		}
		switch frame := frame.(type) {
		case *http2.SettingsFrame:
			if !frame.IsAck() {
				if err := framer.WriteSettingsAck(); err != nil {
					return fmt.Errorf("write settings ack: %w", err)
				}
			}
		case *http2.PingFrame:
			if !frame.Flags.Has(http2.FlagPingAck) {
				if err := framer.WritePing(true, frame.Data); err != nil {
					return fmt.Errorf("write ping ack: %w", err)
				}
			}
		case *http2.MetaHeadersFrame:
			peer.headers.Add(1)
			headers := metadata.MD{}
			for _, field := range frame.Fields {
				headers.Append(strings.ToLower(field.Name), field.Value)
			}
			captured = append(captured, headers)
			if len(captured) == 1 {
				close(firstAttempt)
				if releaseFirst != nil {
					<-releaseFirst
				}
				if err := framer.WriteRSTStream(frame.StreamID, http2.ErrCodeRefusedStream); err != nil {
					return fmt.Errorf("refuse first stream: %w", err)
				}
				continue
			}
			if len(captured) == 2 {
				if err := writeRawGRPCSuccess(framer, frame.StreamID); err != nil {
					return err
				}
				result <- captured
			}
		}
	}
}

func writeRawGRPCSuccess(framer *http2.Framer, streamID uint32) error {
	if err := writeRawGRPCHeaders(
		framer,
		streamID,
		false,
		hpack.HeaderField{Name: ":status", Value: "200"},
		hpack.HeaderField{Name: "content-type", Value: "application/grpc"},
	); err != nil {
		return err
	}
	if err := framer.WriteData(streamID, false, []byte{0, 0, 0, 0, 0}); err != nil {
		return fmt.Errorf("write empty gRPC response: %w", err)
	}
	return writeRawGRPCHeaders(
		framer,
		streamID,
		true,
		hpack.HeaderField{Name: "grpc-status", Value: "0"},
	)
}

func writeRawGRPCHeaders(
	framer *http2.Framer,
	streamID uint32,
	endStream bool,
	fields ...hpack.HeaderField,
) error {
	var encoded bytes.Buffer
	encoder := hpack.NewEncoder(&encoded)
	for _, field := range fields {
		if err := encoder.WriteField(field); err != nil {
			return fmt.Errorf("encode gRPC response header: %w", err)
		}
	}
	if err := framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      streamID,
		BlockFragment: encoded.Bytes(),
		EndStream:     endStream,
		EndHeaders:    true,
	}); err != nil {
		return fmt.Errorf("write gRPC response headers: %w", err)
	}
	return nil
}

type grpcFixture struct {
	application grpc.ClientConnInterface
	connection  *grpc.ClientConn
	credential  *countingCredential
	provider    *scriptedAcquirer
	service     *grpcTestService
}

func newGRPCFixture(t *testing.T, clock *movableClock, provider *scriptedAcquirer) *grpcFixture {
	t.Helper()
	return newGRPCFixtureWithMeter(t, clock, provider, nil)
}

func newGRPCFixtureWithMeter(
	t *testing.T,
	clock *movableClock,
	provider *scriptedAcquirer,
	meterProvider metric.MeterProvider,
) *grpcFixture {
	t.Helper()
	return newGRPCFixtureWithResourceAuthorityAndMeter(t, clock, provider, "https://"+testGRPCHost, meterProvider)
}

func newGRPCFixtureWithResourceAuthority(
	t *testing.T,
	clock *movableClock,
	provider *scriptedAcquirer,
	resourceAuthority string,
) *grpcFixture {
	t.Helper()
	return newGRPCFixtureWithResourceAuthorityAndMeter(t, clock, provider, resourceAuthority, nil)
}

func newGRPCFixtureWithResourceAuthorityAndMeter(
	t *testing.T,
	clock *movableClock,
	provider *scriptedAcquirer,
	resourceAuthority string,
	meterProvider metric.MeterProvider,
) *grpcFixture {
	t.Helper()
	if clock == nil {
		clock = newMovableClock(fixedProviderTime)
	}
	if provider == nil {
		provider = &scriptedAcquirer{steps: []acquisitionStep{{token: grpcTestToken(clock)}}}
	}
	address := privateTestAddress(t)
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", net.JoinHostPort(address.String(), "0"))
	if err != nil {
		t.Fatalf("listen for gRPC test server: %v", err)
	}
	certificate, err := grpcTestPKI().certificateFor(testGRPCHost)
	if err != nil {
		t.Fatal(err)
	}
	service := newGRPCTestService()
	server := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(serverTLSConfig(certificate))),
		grpc.StreamInterceptor(service.healthInterceptor),
	)
	server.RegisterService(&grpcTestServiceDescription, service)
	healthgrpc.RegisterHealthServer(server, service)
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		if serveErr := <-done; serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			t.Errorf("gRPC server Serve() error = %v", serveErr)
		}
	})

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split gRPC listener address: %v", err)
	}
	installPrivateTestResolver(t, map[string]netip.Addr{testGRPCHost: address})
	targetAuthority := net.JoinHostPort(testGRPCHost, port)
	cfg := validTestConfig()
	cfg.ResourceAuthority = resourceAuthority
	client := requireTestClient(t, cfg, testClientOptions{now: clock.Now, acquire: provider.acquire, meterProvider: meterProvider})
	auth, err := NewGRPC(client)
	if err != nil {
		t.Fatalf("NewGRPC() error = %v", err)
	}
	counting := &countingCredential{base: auth}
	clientConfig := grpcclient.DefaultConfig("dns:///" + targetAuthority)
	clientConfig.HealthCheck = false
	connection, err := grpcclient.New(clientConfig, grpcclient.Options{
		TransportCredentials: credentials.NewTLS(tlsConfigForClient(testGRPCHost)),
		PerRPCCredentials:    counting,
	})
	if err != nil {
		t.Fatalf("grpcclient.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("ClientConn.Close() error = %v", err)
		}
	})
	application, err := auth.Wrap(connection)
	if err != nil {
		t.Fatalf("GRPC.Wrap() error = %v", err)
	}
	return &grpcFixture{
		application: application,
		connection:  connection,
		credential:  counting,
		provider:    provider,
		service:     service,
	}
}

func grpcTestToken(clock *movableClock) accessToken {
	return accessToken{value: grpcBearerToken, expiresAt: clock.Now().Add(30 * time.Second)}
}

func exerciseGRPCStream(t *testing.T, connection grpc.ClientConnInterface, description *grpc.StreamDesc, method string) {
	t.Helper()
	stream, err := connection.NewStream(t.Context(), description, method)
	if err != nil {
		t.Fatalf("NewStream(%s) error = %v", method, err)
	}
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		t.Fatalf("SendMsg(%s) error = %v", method, err)
	}
	if description.ClientStreams {
		if err := stream.CloseSend(); err != nil {
			t.Fatalf("CloseSend(%s) error = %v", method, err)
		}
	}
	if err := stream.RecvMsg(&emptypb.Empty{}); err != nil {
		t.Fatalf("RecvMsg(%s) error = %v", method, err)
	}
}

func assertOneBearer(t *testing.T, values metadata.MD, token string) {
	t.Helper()
	got := values.Get("authorization")
	if len(got) != 1 || got[0] != "Bearer "+token {
		t.Fatalf("authorization metadata = %v, want one bearer", got)
	}
}

type countingCredential struct {
	base  credentials.PerRPCCredentials
	calls atomic.Int64
	last  atomic.Value
}

func (c *countingCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	c.calls.Add(1)
	c.last.Store(strings.Join(uri, " | "))
	//nolint:wrapcheck // The test wrapper must not change the credential result.
	return c.base.GetRequestMetadata(ctx, uri...)
}

func (c *countingCredential) RequireTransportSecurity() bool {
	return c.base.RequireTransportSecurity()
}

type staticTestCredential struct{}

func (staticTestCredential) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer caller"}, nil
}

func (staticTestCredential) RequireTransportSecurity() bool { return true }

type grpcTestServiceServer interface {
	Unary(ctx context.Context, request *emptypb.Empty) (*emptypb.Empty, error)
}

type grpcTestService struct {
	healthgrpc.UnimplementedHealthServer

	metadata       chan metadata.MD
	healthMetadata chan metadata.MD
	calls          atomic.Int64
	healthCalls    atomic.Int64
	unaryCode      atomic.Int32
	longStream     atomic.Bool
}

func newGRPCTestService() *grpcTestService {
	return &grpcTestService{metadata: make(chan metadata.MD, 16), healthMetadata: make(chan metadata.MD, 16)}
}

func (s *grpcTestService) Unary(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	s.record(ctx)
	if code := codes.Code(s.unaryCode.Load()); code != codes.OK {
		return nil, status.Error(code, "fixed downstream status")
	}
	return &emptypb.Empty{}, nil
}

func (s *grpcTestService) Watch(
	_ *healthgrpc.HealthCheckRequest,
	stream grpc.ServerStreamingServer[healthgrpc.HealthCheckResponse],
) error {
	s.healthCalls.Add(1)
	if err := stream.Send(&healthgrpc.HealthCheckResponse{Status: healthgrpc.HealthCheckResponse_SERVING}); err != nil {
		return fmt.Errorf("send health status: %w", err)
	}
	<-stream.Context().Done()
	return fmt.Errorf("health watch: %w", stream.Context().Err())
}

func (s *grpcTestService) healthInterceptor(
	_ any,
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	if info.FullMethod == healthgrpc.Health_Watch_FullMethodName {
		values, _ := metadata.FromIncomingContext(stream.Context())
		s.healthMetadata <- values
	}
	return handler(s, stream)
}

func (s *grpcTestService) record(ctx context.Context) {
	s.calls.Add(1)
	values, _ := metadata.FromIncomingContext(ctx)
	s.metadata <- values
}

func (s *grpcTestService) serverStream(stream grpc.ServerStream) error {
	if err := stream.RecvMsg(&emptypb.Empty{}); err != nil {
		return fmt.Errorf("receive server-stream request: %w", err)
	}
	s.record(stream.Context())
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		return fmt.Errorf("send server-stream response: %w", err)
	}
	return nil
}

func (s *grpcTestService) clientStream(stream grpc.ServerStream) error {
	s.record(stream.Context())
	for {
		if err := stream.RecvMsg(&emptypb.Empty{}); errors.Is(err, io.EOF) {
			if sendErr := stream.SendMsg(&emptypb.Empty{}); sendErr != nil {
				return fmt.Errorf("send client-stream response: %w", sendErr)
			}
			return nil
		} else if err != nil {
			return fmt.Errorf("receive client-stream request: %w", err)
		}
	}
}

func (s *grpcTestService) bidiStream(stream grpc.ServerStream) error {
	s.record(stream.Context())
	if err := stream.RecvMsg(&emptypb.Empty{}); err != nil {
		return fmt.Errorf("receive first bidi request: %w", err)
	}
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		return fmt.Errorf("send first bidi response: %w", err)
	}
	if !s.longStream.Load() {
		return nil
	}
	if err := stream.RecvMsg(&emptypb.Empty{}); err != nil {
		return fmt.Errorf("receive second bidi request: %w", err)
	}
	return status.Error(codes.Unauthenticated, "fixed downstream status")
}

var grpcTestServiceDescription = grpc.ServiceDesc{
	ServiceName: "outbound.auth.test.Service",
	HandlerType: (*grpcTestServiceServer)(nil),
	Methods: []grpc.MethodDesc{{
		MethodName: "Unary",
		Handler: func(service any, ctx context.Context, decode func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
			server, ok := service.(*grpcTestService)
			if !ok {
				return nil, status.Error(codes.Internal, "invalid test service")
			}
			request := &emptypb.Empty{}
			if err := decode(request); err != nil {
				return nil, err
			}
			return server.Unary(ctx, request)
		},
	}},
	Streams: []grpc.StreamDesc{
		{StreamName: "ServerStream", Handler: func(service any, stream grpc.ServerStream) error {
			server, ok := service.(*grpcTestService)
			if !ok {
				return status.Error(codes.Internal, "invalid test service")
			}
			return server.serverStream(stream)
		}, ServerStreams: true},
		{StreamName: "ClientStream", Handler: func(service any, stream grpc.ServerStream) error {
			server, ok := service.(*grpcTestService)
			if !ok {
				return status.Error(codes.Internal, "invalid test service")
			}
			return server.clientStream(stream)
		}, ClientStreams: true},
		{StreamName: "BidiStream", Handler: func(service any, stream grpc.ServerStream) error {
			server, ok := service.(*grpcTestService)
			if !ok {
				return status.Error(codes.Internal, "invalid test service")
			}
			return server.bidiStream(stream)
		}, ClientStreams: true, ServerStreams: true},
	},
}

func serverTLSConfig(certificate tls.Certificate) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"h2"},
	}
}

func tlsConfigForClient(serverName string) *tls.Config {
	return &tls.Config{RootCAs: grpcTestPKI().pool, ServerName: serverName, MinVersion: tls.VersionTLS12}
}
