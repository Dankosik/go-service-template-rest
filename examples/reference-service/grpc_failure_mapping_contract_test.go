package referenceservice

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/internal/failure"
	grpcx "github.com/example/go-service-template-rest/internal/infra/grpc"
	"github.com/example/go-service-template-rest/internal/infra/grpc/grpctest"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

const articleCreateFullMethod = "/reference.test.Article/Create"

func TestArticleAlreadyExistsMapsThroughGRPCTransport(t *testing.T) {
	listener := bufconn.Listen(1 << 20)
	server, err := grpcx.NewServer(grpcx.Config{
		MaxConcurrentRPCs:          4,
		MaxConcurrentStreams:       4,
		MaxHeaderListBytes:         16 << 10,
		MaxReceiveMessageBytes:     4 << 20,
		MaxSendMessageBytes:        4 << 20,
		AccessLogSuccessSampleRate: 1,
		MaxConnectionIdle:          time.Minute,
		ServerPingInterval:         time.Minute,
		ServerPingTimeout:          20 * time.Second,
		MinClientPingInterval:      10 * time.Second,
		PermitPingWithoutStream:    true,
	}, grpcx.Options{
		Logger:       slog.New(slog.DiscardHandler),
		DomainErrors: []failure.Mapper{article.ClassifyError},
		ErrorDomain:  "reference-service.test",
		Services: []grpcx.RegisterService{func(registrar grpc.ServiceRegistrar) {
			grpctest.Register(registrar, grpctest.Unary(articleCreateFullMethod,
				func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
					return nil, fmt.Errorf("create article in storage adapter: %w", article.ErrAlreadyExists)
				},
			))
		}},
	})
	if err != nil {
		_ = listener.Close()
		t.Fatalf("grpcx.NewServer() error = %v", err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()

	connection, err := grpc.NewClient("passthrough:///reference", grpc.WithContextDialer(
		func(context.Context, string) (net.Conn, error) { return listener.Dial() },
	), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = server.Close()
		_ = listener.Close()
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("ClientConn.Close() error = %v", err)
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

	err = connection.Invoke(t.Context(), articleCreateFullMethod, &emptypb.Empty{}, &emptypb.Empty{})
	converted := status.Convert(err)
	if converted.Code() != codes.AlreadyExists || converted.Message() != "an article with this slug already exists" ||
		strings.Contains(converted.Message(), "storage adapter") {
		t.Fatalf("status = %s/%q, want safe AlreadyExists", converted.Code(), converted.Message())
	}
	details := converted.Details()
	if len(details) != 1 {
		t.Fatalf("details = %v, want one ErrorInfo and no RetryInfo", details)
	}
	info, ok := details[0].(*errdetails.ErrorInfo)
	if !ok || info.GetReason() != "ALREADY_EXISTS" || info.GetDomain() != "reference-service.test" {
		t.Fatalf("details = %v, want ALREADY_EXISTS ErrorInfo", details)
	}
}
