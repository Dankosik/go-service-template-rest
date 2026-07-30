package grpcx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/problem"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAccessLogPolicyPreservesErrorAndSlowVisibility(t *testing.T) {
	policy := accessLogPolicy{
		successSampleRate: 0,
		slowThreshold:     time.Second,
	}

	for _, testCase := range []struct {
		name             string
		missingRequestID bool
		code             codes.Code
		elapsed          time.Duration
		want             bool
	}{
		{name: "fast success sampled out", code: codes.OK, elapsed: time.Second - 1, want: false},
		{name: "threshold success retained", code: codes.OK, elapsed: time.Second, want: true},
		{name: "error retained", code: codes.Internal, want: true},
		{name: "missing correlation fails open", missingRequestID: true, code: codes.OK, want: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := t.Context()
			if !testCase.missingRequestID {
				ctx = reqctx.ContextWithRequestID(ctx, "stable-request")
			}
			if got := policy.shouldLog(ctx, testCase.code, testCase.elapsed); got != testCase.want {
				t.Fatalf("shouldLog() = %t, want %t", got, testCase.want)
			}
		})
	}
}

func TestAccessLogSuccessSamplingIsDeterministicAndBounded(t *testing.T) {
	if sampleRequestID("request", 0) {
		t.Fatal("sampleRequestID(rate 0) = true, want false")
	}
	if !sampleRequestID("request", 1) {
		t.Fatal("sampleRequestID(rate 1) = false, want true")
	}
	if !sampleRequestID("", 0) {
		t.Fatal("sampleRequestID(empty request ID) = false, want fail-open logging")
	}

	selected := 0
	const candidates = 1000
	for index := range candidates {
		requestID := "request-" + strconv.Itoa(index)
		first := sampleRequestID(requestID, 0.1)
		if got := sampleRequestID(requestID, 0.1); got != first {
			t.Fatalf("sampleRequestID(%q) changed decision", requestID)
		}
		if first {
			selected++
		}
	}
	if selected < 50 || selected > 150 {
		t.Fatalf("10%% sample selected %d/%d request IDs, want a bounded partition", selected, candidates)
	}
}

func TestAccessLogInterceptorsApplyHealthAndSuccessPolicy(t *testing.T) {
	var output bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&output, nil))
	policy := accessLogPolicy{successSampleRate: 0}
	ctx := reqctx.ContextWithRequestID(t.Context(), "sampled-out-success")

	handlerCalls := 0
	if _, err := accessLogUnaryInterceptor(log, policy)(
		ctx,
		nil,
		&grpc.UnaryServerInfo{FullMethod: testUnaryFullMethod},
		func(context.Context, any) (any, error) {
			handlerCalls++
			return struct{}{}, nil
		},
	); err != nil {
		t.Fatalf("successful unary interceptor error = %v", err)
	}
	if _, err := accessLogUnaryInterceptor(log, policy)(
		ctx,
		nil,
		&grpc.UnaryServerInfo{FullMethod: healthMethodPrefix + "Check"},
		func(context.Context, any) (any, error) {
			handlerCalls++
			return nil, status.Error(codes.Internal, "health failure")
		},
	); status.Code(err) != codes.Internal {
		t.Fatalf("health unary status = %s, want %s", status.Code(err), codes.Internal)
	}

	streamErr := status.Error(codes.ResourceExhausted, "busy")
	if err := accessLogStreamInterceptor(log, policy)(
		nil,
		testServerStream{context: func() context.Context { return ctx }},
		&grpc.StreamServerInfo{FullMethod: "/grpcx.test.Service/Stream"},
		func(any, grpc.ServerStream) error {
			handlerCalls++
			return fmt.Errorf("stream handler: %w", streamErr)
		},
	); !errors.Is(err, streamErr) {
		t.Fatalf("stream interceptor error = %v, want %v", err, streamErr)
	}

	if handlerCalls != 3 {
		t.Fatalf("handler calls = %d, want 3", handlerCalls)
	}
	encoded := strings.TrimSpace(output.String())
	if lines := strings.Count(encoded, "\n") + 1; lines != 1 {
		t.Fatalf("access-log records = %d, want only the business error; logs = %s", lines, encoded)
	}
	if !strings.Contains(encoded, `"rpc.status":"ResourceExhausted"`) ||
		!strings.Contains(encoded, `"/grpcx.test.Service/Stream"`) {
		t.Fatalf("access log = %s, want business stream error", encoded)
	}
}

func TestAccessLogStreamInterceptorMatchesUnaryPolicy(t *testing.T) {
	for _, testCase := range []struct {
		name             string
		missingRequestID bool
		method           string
		policy           accessLogPolicy
		handlerDelay     time.Duration
		handlerErr       error
		wantLog          bool
	}{
		{
			name:       "health excluded before errors",
			method:     healthMethodPrefix + "Watch",
			policy:     accessLogPolicy{successSampleRate: 1},
			handlerErr: status.Error(codes.Internal, "health failure"),
			wantLog:    false,
		},
		{
			name:       "business error retained",
			method:     "/grpcx.test.Service/Stream",
			policy:     accessLogPolicy{successSampleRate: 0},
			handlerErr: status.Error(codes.ResourceExhausted, "busy"),
			wantLog:    true,
		},
		{
			name:   "slow success retained before sampling",
			method: "/grpcx.test.Service/Stream",
			policy: accessLogPolicy{
				successSampleRate: 0,
				slowThreshold:     time.Second,
			},
			handlerDelay: time.Second,
			wantLog:      true,
		},
		{
			name:    "fast success sampled out",
			method:  "/grpcx.test.Service/Stream",
			policy:  accessLogPolicy{successSampleRate: 0},
			wantLog: false,
		},
		{
			name:             "missing correlation fails open",
			missingRequestID: true,
			method:           "/grpcx.test.Service/Stream",
			policy:           accessLogPolicy{successSampleRate: 0},
			wantLog:          true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx := t.Context()
				if !testCase.missingRequestID {
					ctx = reqctx.ContextWithRequestID(ctx, "stream-policy-request")
				}
				var output bytes.Buffer
				log := slog.New(slog.NewJSONHandler(&output, nil))
				handlerCalled := false

				err := accessLogStreamInterceptor(log, testCase.policy)(
					nil,
					testServerStream{context: func() context.Context { return ctx }},
					&grpc.StreamServerInfo{FullMethod: testCase.method},
					func(any, grpc.ServerStream) error {
						handlerCalled = true
						time.Sleep(testCase.handlerDelay)
						return testCase.handlerErr
					},
				)
				if !errors.Is(err, testCase.handlerErr) || !handlerCalled {
					t.Fatalf(
						"stream interceptor = (called %t, error %v), want handler and %v",
						handlerCalled,
						err,
						testCase.handlerErr,
					)
				}
				if got := output.Len() > 0; got != testCase.wantLog {
					t.Fatalf(
						"stream access-log emitted = %t, want %t; log = %q",
						got,
						testCase.wantLog,
						output.String(),
					)
				}
			})
		})
	}
}

func TestAccessLogSkipsAllWorkWhenInfoIsDisabled(t *testing.T) {
	var output bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelWarn}))
	unaryCalled := false

	_, err := accessLogUnaryInterceptor(log, accessLogPolicy{successSampleRate: 1})(
		t.Context(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: testUnaryFullMethod},
		func(context.Context, any) (any, error) {
			unaryCalled = true
			return nil, status.Error(codes.Internal, "failure")
		},
	)
	if status.Code(err) != codes.Internal || !unaryCalled {
		t.Fatalf("level-disabled unary interceptor = (called %t, status %s), want handler and Internal", unaryCalled, status.Code(err))
	}

	streamCalled := false
	err = accessLogStreamInterceptor(log, accessLogPolicy{successSampleRate: 1})(
		nil,
		testServerStream{context: t.Context},
		&grpc.StreamServerInfo{FullMethod: "/grpcx.test.Service/Stream"},
		func(any, grpc.ServerStream) error {
			streamCalled = true
			return status.Error(codes.Internal, "failure")
		},
	)
	if status.Code(err) != codes.Internal || !streamCalled {
		t.Fatalf("level-disabled stream interceptor = (called %t, status %s), want handler and Internal", streamCalled, status.Code(err))
	}
	if output.Len() != 0 {
		t.Fatalf("level-disabled access log = %q, want empty", output.String())
	}
}

func TestAdmissionLimitIsSharedAcrossUnaryAndStreamingRPCs(t *testing.T) {
	load := &recordingLoad{}
	limiter := newAdmissionLimiter(1, load)
	unary := admissionUnaryInterceptor(limiter)
	streaming := admissionStreamInterceptor(limiter)

	entered := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		_, err := unary(
			t.Context(),
			nil,
			&grpc.UnaryServerInfo{FullMethod: testUnaryFullMethod},
			func(context.Context, any) (any, error) {
				close(entered)
				<-release
				return struct{}{}, nil
			},
		)
		firstDone <- err
	}()
	<-entered

	streamHandlerCalled := false
	err := streaming(
		nil,
		testServerStream{context: t.Context},
		&grpc.StreamServerInfo{FullMethod: "/grpcx.test.Service/Stream"},
		func(any, grpc.ServerStream) error {
			streamHandlerCalled = true
			return nil
		},
	)
	assertStatusCode(t, err, codes.ResourceExhausted)
	if streamHandlerCalled {
		t.Fatal("shed streaming RPC entered its handler")
	}

	if _, err := unary(
		t.Context(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: healthMethodPrefix + "Check"},
		func(context.Context, any) (any, error) { return struct{}{}, nil },
	); err != nil {
		t.Fatalf("health RPC was shed: %v", err)
	}

	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("admitted unary RPC error = %v", err)
	}
	if active, shed := load.snapshot(); active != 0 || shed != 1 {
		t.Fatalf("load = (active %d, shed %d), want (0, 1)", active, shed)
	}
}

func TestMapErrorUsesOwnedProvenanceAndSanitizesUnknownStatus(t *testing.T) {
	sentinel := errors.New("domain sentinel")
	mapper := func(err error) (problem.Mapped, bool) {
		if !errors.Is(err, sentinel) {
			return problem.Mapped{}, false
		}
		return problem.Mapped{Code: problem.CodeNotFound, Detail: "record not found"}, true
	}

	for _, testCase := range []struct {
		name       string
		err        error
		mappers    []problem.Mapper
		wantCode   codes.Code
		wantDetail string
	}{
		{name: "canceled", err: context.Canceled, wantCode: codes.Canceled, wantDetail: "request canceled"},
		{
			name:       "deadline",
			err:        context.DeadlineExceeded,
			wantCode:   codes.DeadlineExceeded,
			wantDetail: "request deadline exceeded",
		},
		{
			name:       "domain",
			err:        sentinel,
			mappers:    []problem.Mapper{mapper},
			wantCode:   codes.NotFound,
			wantDetail: "record not found",
		},
		{
			name:       "owned",
			err:        ownedStatus(codes.Unavailable, "service draining"),
			wantCode:   codes.Unavailable,
			wantDetail: "service draining",
		},
		{
			name:       "unmarked downstream",
			err:        status.Error(codes.PermissionDenied, "dependency secret"),
			wantCode:   codes.Internal,
			wantDetail: "request failed",
		},
		{
			name:       "raw",
			err:        errors.New("password=secret"),
			wantCode:   codes.Internal,
			wantDetail: "request failed",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got := mapError(testCase.err, testCase.mappers)
			if code := status.Code(got); code != testCase.wantCode {
				t.Fatalf("code = %s, want %s", code, testCase.wantCode)
			}
			if detail := status.Convert(got).Message(); detail != testCase.wantDetail {
				t.Fatalf("detail = %q, want %q", detail, testCase.wantDetail)
			}
		})
	}
}

func TestRecoveryReturnsSanitizedOwnedStatus(t *testing.T) {
	interceptor := recoveryUnaryInterceptor(slog.New(slog.DiscardHandler))
	_, err := interceptor(
		t.Context(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: testUnaryFullMethod},
		func(context.Context, any) (any, error) {
			panic("credential=secret")
		},
	)
	assertStatusCode(t, err, codes.Internal)
	if detail := status.Convert(err).Message(); detail != "request failed" {
		t.Fatalf("panic status detail = %q, want sanitized detail", detail)
	}
	if _, ok := errors.AsType[*ownedStatusError](err); !ok {
		t.Fatalf("panic error type = %T, want ownedStatusError", err)
	}
}

type recordingLoad struct {
	mu     sync.Mutex
	active int
	shed   int
}

func (l *recordingLoad) Admitted(context.Context) func() {
	l.mu.Lock()
	l.active++
	l.mu.Unlock()
	return func() {
		l.mu.Lock()
		l.active--
		l.mu.Unlock()
	}
}

func (l *recordingLoad) Shed(context.Context) {
	l.mu.Lock()
	l.shed++
	l.mu.Unlock()
}

func (l *recordingLoad) snapshot() (int, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.active, l.shed
}

type testServerStream struct {
	context func() context.Context
}

func (s testServerStream) SetHeader(metadata.MD) error  { return nil }
func (s testServerStream) SendHeader(metadata.MD) error { return nil }
func (s testServerStream) SetTrailer(metadata.MD)       {}
func (s testServerStream) Context() context.Context     { return s.context() }
func (s testServerStream) SendMsg(any) error            { return nil }
func (s testServerStream) RecvMsg(any) error            { return nil }
