package bearerauthn

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	grpcx "github.com/example/go-service-template-rest/internal/infra/grpc"
	"github.com/example/go-service-template-rest/internal/infra/grpc/grpctest"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/example/go-service-template-rest/internal/waittest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

const expiringBidiMethod = "/bearerauthn.test.Expiry/Block"

func TestGRPCAuthenticationErrorsStayPrivate(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		err  error
		code codes.Code
	}{
		{err: NewError(KindInvalid), code: codes.Unauthenticated},
		{err: NewError(KindOversize), code: codes.ResourceExhausted},
		{err: NewError(KindUnavailable), code: codes.Unavailable},
		{err: context.Canceled, code: codes.Canceled},
		{err: context.DeadlineExceeded, code: codes.DeadlineExceeded},
	} {
		if got := status.Code(grpcAuthenticationError(testCase.err)); got != testCase.code {
			t.Fatalf("status.Code(%v) = %v, want %v", testCase.err, got, testCase.code)
		}
	}
}

func TestProtectedStreamExpiry(t *testing.T) {
	t.Parallel()

	now := time.Now()
	expiryBeforeCaller := now.Add(time.Minute)
	callerBeforeExpiry := now.Add(2 * time.Hour)

	t.Run("handler deadline is min of caller and exp plus skew", func(t *testing.T) {
		t.Parallel()
		runtime := newTestRuntime(t, &fakeVerifier{
			result: Result{
				Principal: reqctx.Principal{Subject: "subject-1"},
				ExpiresAt: expiryBeforeCaller,
			},
		})
		caller, cancel := context.WithDeadline(t.Context(), callerBeforeExpiry)
		t.Cleanup(cancel)
		stream := &stubServerStream{ctx: metadata.NewIncomingContext(caller, metadata.Pairs("authorization", "Bearer token"))}
		var observed time.Time
		err := runtime.StreamInterceptor()(nil, stream, &grpc.StreamServerInfo{}, func(_ any, wrapped grpc.ServerStream) error {
			deadline, ok := wrapped.Context().Deadline()
			if !ok {
				t.Fatal("handler context has no deadline")
			}
			observed = deadline
			return nil
		})
		if err != nil {
			t.Fatalf("StreamInterceptor() error = %v", err)
		}
		want := expiryBeforeCaller.Add(ClockSkew)
		if !observed.Equal(want) {
			t.Fatalf("handler deadline = %v, want exp+skew %v", observed, want)
		}
	})

	t.Run("caller deadline wins when earlier than expiry", func(t *testing.T) {
		t.Parallel()
		runtime := newTestRuntime(t, &fakeVerifier{
			result: Result{
				Principal: reqctx.Principal{Subject: "subject-1"},
				ExpiresAt: callerBeforeExpiry,
			},
		})
		callerDeadline := now.Add(30 * time.Second)
		caller, cancel := context.WithDeadline(t.Context(), callerDeadline)
		t.Cleanup(cancel)
		stream := &stubServerStream{ctx: metadata.NewIncomingContext(caller, metadata.Pairs("authorization", "Bearer token"))}
		err := runtime.StreamInterceptor()(nil, stream, &grpc.StreamServerInfo{}, func(_ any, wrapped grpc.ServerStream) error {
			deadline, ok := wrapped.Context().Deadline()
			if !ok || !deadline.Equal(callerDeadline) {
				t.Fatalf("handler deadline = (%v, %v), want caller %v", deadline, ok, callerDeadline)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("StreamInterceptor() error = %v", err)
		}
	})

	t.Run("message before expiry delegates once", func(t *testing.T) {
		t.Parallel()
		runtime := newTestRuntime(t, &fakeVerifier{
			result: Result{
				Principal: reqctx.Principal{Subject: "subject-1"},
				ExpiresAt: now.Add(time.Hour),
			},
		})
		inner := &stubServerStream{ctx: metadata.NewIncomingContext(t.Context(), metadata.Pairs("authorization", "Bearer token"))}
		err := runtime.StreamInterceptor()(nil, inner, &grpc.StreamServerInfo{}, func(_ any, wrapped grpc.ServerStream) error {
			if err := wrapped.RecvMsg(&emptypb.Empty{}); err != nil {
				return fmt.Errorf("receive pre-expiry stream message: %w", err)
			}
			return wrapped.SendMsg(&emptypb.Empty{})
		})
		if err != nil {
			t.Fatalf("pre-expiry stream I/O error = %v", err)
		}
		if inner.recvs != 1 || inner.sends != 1 {
			t.Fatalf("inner I/O = recv %d send %d, want one each", inner.recvs, inner.sends)
		}
	})

	t.Run("message after expiry does not delegate", func(t *testing.T) {
		t.Parallel()
		runtime := newTestRuntime(t, &fakeVerifier{
			result: Result{
				Principal: reqctx.Principal{Subject: "subject-1"},
				ExpiresAt: time.Now().Add(-time.Hour),
			},
		})
		inner := &stubServerStream{ctx: metadata.NewIncomingContext(t.Context(), metadata.Pairs("authorization", "Bearer token"))}
		err := runtime.StreamInterceptor()(nil, inner, &grpc.StreamServerInfo{}, func(_ any, wrapped grpc.ServerStream) error {
			recvErr := wrapped.RecvMsg(&emptypb.Empty{})
			sendErr := wrapped.SendMsg(&emptypb.Empty{})
			if recvErr == nil || sendErr == nil {
				t.Fatal("post-expiry I/O succeeded")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("StreamInterceptor() error = %v", err)
		}
		if inner.recvs != 0 || inner.sends != 0 {
			t.Fatalf("inner I/O = recv %d send %d, want none after expiry", inner.recvs, inner.sends)
		}
	})
}

func TestProtectedStreamExpiryUnblocksMessageIO(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name      string
		configure func(*stubServerStream, <-chan struct{}, chan<- struct{})
		invoke    func(grpc.ServerStream) error
		calls     func(*stubServerStream) int
	}{
		{
			name: "receive",
			configure: func(stream *stubServerStream, block <-chan struct{}, done chan<- struct{}) {
				stream.recvBlock = block
				stream.recvDone = done
			},
			invoke: func(stream grpc.ServerStream) error { return stream.RecvMsg(&emptypb.Empty{}) },
			calls:  func(stream *stubServerStream) int { return stream.recvs },
		},
		{
			name: "send",
			configure: func(stream *stubServerStream, block <-chan struct{}, done chan<- struct{}) {
				stream.sendBlock = block
				stream.sendDone = done
			},
			invoke: func(stream grpc.ServerStream) error { return stream.SendMsg(&emptypb.Empty{}) },
			calls:  func(stream *stubServerStream) int { return stream.sends },
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				runtime := newTestRuntime(t, &fakeVerifier{
					result: Result{
						Principal: reqctx.Principal{Subject: "subject-1"},
						ExpiresAt: time.Now().Add(time.Second - ClockSkew),
					},
				})
				block := make(chan struct{})
				done := make(chan struct{})
				inner := &stubServerStream{
					ctx: metadata.NewIncomingContext(t.Context(), metadata.Pairs("authorization", "Bearer token")),
				}
				testCase.configure(inner, block, done)
				err := runtime.StreamInterceptor()(nil, inner, &grpc.StreamServerInfo{}, func(_ any, wrapped grpc.ServerStream) error {
					return testCase.invoke(wrapped)
				})
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("stream I/O error = %v, want deadline", err)
				}
				close(block)
				synctest.Wait()
				select {
				case <-done:
				default:
					t.Fatal("inner stream operation did not exit")
				}
				if got := testCase.calls(inner); got != 1 {
					t.Fatalf("inner calls = %d, want 1", got)
				}
			})
		})
	}
}

func TestProtectedStreamExpiryReleasesRealTransportIO(t *testing.T) {
	verifier := new(fakeVerifier)
	runtime := newTestRuntime(t, verifier)
	blocked := make(chan struct{})
	handlerDone := make(chan struct{})
	server, err := grpcx.NewServer(grpcx.Options{
		StreamPolicy: []grpc.StreamServerInterceptor{runtime.StreamInterceptor()},
		Services: []grpcx.RegisterService{func(registrar grpc.ServiceRegistrar) {
			registrar.RegisterService(&grpc.ServiceDesc{
				ServiceName: "bearerauthn.test.Expiry",
				Streams: []grpc.StreamDesc{{
					StreamName:    "Block",
					ClientStreams: true,
					ServerStreams: true,
					Handler: func(_ any, stream grpc.ServerStream) error {
						defer close(handlerDone)
						var message emptypb.Empty
						if recvErr := stream.RecvMsg(&message); recvErr != nil {
							return fmt.Errorf("receive initial message: %w", recvErr)
						}
						close(blocked)
						return stream.RecvMsg(&message)
					},
				}},
			}, nil)
		}},
	})
	if err != nil {
		t.Fatalf("grpcx.NewServer() error = %v", err)
	}
	connection := grpctest.ServeBufconn(t, server)
	verifier.result = Result{
		Principal: reqctx.Principal{Subject: "subject-1"},
		ExpiresAt: time.Now().Add(500*time.Millisecond - ClockSkew),
	}
	credential := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("authorization", "Bearer token"))
	stream, err := connection.NewStream(
		credential,
		&grpc.StreamDesc{ClientStreams: true, ServerStreams: true},
		expiringBidiMethod,
	)
	if err != nil {
		t.Fatalf("ClientConn.NewStream() error = %v", err)
	}
	if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
		t.Fatalf("ClientStream.SendMsg() error = %v", err)
	}
	waittest.ReceiveSignal(t, blocked, time.Second, "handler to block on the next receive")

	result := make(chan error, 1)
	go func() { result <- stream.RecvMsg(&emptypb.Empty{}) }()
	err = waittest.Receive(t, result, 2*time.Second, "expired stream result")
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("expired stream code = %s, want %s (%v)", status.Code(err), codes.DeadlineExceeded, err)
	}
	waittest.ReceiveSignal(t, handlerDone, time.Second, "expired handler to exit")
}

func TestGRPCAuthnRunsInsideBusinessAdmission(t *testing.T) {
	entered := make(chan struct{}, 256)
	block := make(chan struct{})
	verifier := &fakeVerifier{entered: entered, block: block}
	runtime := newTestRuntime(t, verifier)
	const method = "/bearerauthn.test.Admission/Call"
	server, err := grpcx.NewServer(grpcx.Options{
		UnaryPolicy: []grpc.UnaryServerInterceptor{runtime.UnaryInterceptor()},
		Services: []grpcx.RegisterService{
			func(registrar grpc.ServiceRegistrar) {
				grpctest.Register(registrar, grpctest.Unary(
					method,
					func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
						return &emptypb.Empty{}, nil
					},
				))
			},
		},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	connections := serveBufconnClients(t, server, 3)
	credential := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("authorization", "Bearer token"))

	var started sync.WaitGroup
	started.Add(256)
	for i := range 256 {
		connection := connections[i%len(connections)]
		go func() {
			defer started.Done()
			_ = connection.Invoke(credential, method, &emptypb.Empty{}, &emptypb.Empty{})
		}()
	}
	for range 256 {
		waittest.ReceiveSignal(t, entered, 5*time.Second, "admitted RPC to enter verifier")
	}
	if verifier.calls.Load() != 256 || verifier.peak.Load() != 256 {
		t.Fatalf("verifier calls/peak = %d/%d, want 256/256", verifier.calls.Load(), verifier.peak.Load())
	}

	rejected := make(chan error, 1)
	go func() {
		rejected <- connections[0].Invoke(credential, method, &emptypb.Empty{}, &emptypb.Empty{})
	}()
	err = waittest.Receive(t, rejected, 2*time.Second, "257th RPC to be shed")
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("257th status = %v, want ResourceExhausted", status.Code(err))
	}
	if verifier.calls.Load() != 256 || verifier.peak.Load() != 256 {
		t.Fatalf("after shed verifier calls/peak = %d/%d, want 256/256", verifier.calls.Load(), verifier.peak.Load())
	}
	close(block)
	started.Wait()
}

type stubServerStream struct {
	grpc.ServerStream

	ctx       context.Context //nolint:containedctx // grpc.ServerStream requires Context to return the stub RPC context.
	recvs     int
	sends     int
	recvBlock <-chan struct{}
	recvDone  chan<- struct{}
	sendBlock <-chan struct{}
	sendDone  chan<- struct{}
}

func (s *stubServerStream) Context() context.Context { return s.ctx }

func (s *stubServerStream) RecvMsg(any) error {
	s.recvs++
	if s.recvBlock != nil {
		<-s.recvBlock
	}
	if s.recvDone != nil {
		close(s.recvDone)
	}
	return nil
}

func (s *stubServerStream) SendMsg(any) error {
	s.sends++
	if s.sendBlock != nil {
		<-s.sendBlock
	}
	if s.sendDone != nil {
		close(s.sendDone)
	}
	return nil
}

func (s *stubServerStream) SetHeader(metadata.MD) error  { return nil }
func (s *stubServerStream) SendHeader(metadata.MD) error { return nil }
func (s *stubServerStream) SetTrailer(metadata.MD)       {}

var _ grpc.ServerStream = (*stubServerStream)(nil)

func serveBufconnClients(t *testing.T, server *grpcx.Server, count int) []*grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	connections := make([]*grpc.ClientConn, 0, count)
	for range count {
		connection, err := grpc.NewClient(
			"passthrough:///bufconn",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			t.Fatalf("grpc.NewClient() error = %v", err)
		}
		connections = append(connections, connection)
	}
	t.Cleanup(func() {
		for _, connection := range connections {
			if err := connection.Close(); err != nil {
				t.Errorf("ClientConn.Close() error = %v", err)
			}
		}
		if err := server.Close(); err != nil {
			t.Errorf("Server.Close() error = %v", err)
		}
		if err := <-serveDone; err != nil {
			t.Errorf("Server.Serve() error = %v", err)
		}
		if err := listener.Close(); err != nil {
			t.Errorf("bufconn.Listener.Close() error = %v", err)
		}
	})
	return connections
}
