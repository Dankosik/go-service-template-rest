package referenceservice

import (
	"context"
	"fmt"
	"log/slog"
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
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const articleCreateFullMethod = "/reference.test.Article/Create"

func TestArticleAlreadyExistsMapsThroughGRPCTransport(t *testing.T) {
	server, err := grpcx.NewServer(grpcx.Config{
		MaxConcurrentRPCs:          4,
		MaxConcurrentHealthRPCs:    4,
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
		t.Fatalf("grpcx.NewServer() error = %v", err)
	}
	connection := grpctest.ServeBufconn(t, server)

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
