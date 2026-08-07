// The interceptor policies as units: access-log decisions, admission accounting,
// panic recovery, and the two error boundaries' trust rules.
//
// These drive the policies directly rather than through a server, so a failure
// names the rule that broke. server_test.go and telemetry_test.go cover the same
// rules as a caller and an operator see them.

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

// One accessLogPolicy reaches both RPC kinds through their own adapter, so this
// drives all three of that policy's answers — sampled-out success, health
// exclusion, business error — once per adapter and counts the records they
// produced together. What it owns is the sharing; the decision matrix itself
// belongs to TestAccessLogPolicyAppliesToStreamingRPCs below, which is where a
// new rule goes.
func TestAccessLogPolicyIsSharedByUnaryAndStreamAdapters(t *testing.T) {
	var output bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&output, nil))
	policy := accessLogPolicy{successSampleRate: 0}
	ctx := reqctx.ContextWithRequestID(t.Context(), "sampled-out-success")

	handlerCalls := 0
	if _, err := asUnaryInterceptor(accessLogAround(log, policy))(
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
	if _, err := asUnaryInterceptor(accessLogAround(log, policy))(
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
	if err := asStreamInterceptor(accessLogAround(log, policy))(
		nil,
		testServerStream{ctx: ctx},
		&grpc.StreamServerInfo{FullMethod: testStreamFullMethod},
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
		!strings.Contains(encoded, strconv.Quote(testStreamFullMethod)) {
		t.Fatalf("access log = %s, want business stream error", encoded)
	}
}

// The access-log policy is one function shared by both chains, so this owns its
// whole decision matrix as a streaming RPC reaches them: health exclusion, both
// sides of the slow threshold, sampling, and failing open without a request ID.
//
// It drives accessLogAround rather than shouldLog, which is one shim lower than
// a server and still names the rule that broke — and unlike a predicate-level
// twin it also proves the decision is consulted. A new rule belongs in this
// table, not in a second one beside shouldLog.
func TestAccessLogPolicyAppliesToStreamingRPCs(t *testing.T) {
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
			method:     testStreamFullMethod,
			policy:     accessLogPolicy{successSampleRate: 0},
			handlerErr: status.Error(codes.ResourceExhausted, "busy"),
			wantLog:    true,
		},
		{
			name:   "slow success retained before sampling",
			method: testStreamFullMethod,
			policy: accessLogPolicy{
				successSampleRate: 0,
				slowThreshold:     time.Second,
			},
			handlerDelay: time.Second,
			wantLog:      true,
		},
		{
			name:    "fast success sampled out",
			method:  testStreamFullMethod,
			policy:  accessLogPolicy{successSampleRate: 0},
			wantLog: false,
		},
		{
			// The boundary of elapsed >= slowThreshold, from below: an RPC that
			// finishes one tick short of the threshold is left to sampling, which
			// is what makes the case above it a threshold decision rather than a
			// coincidence.
			name:   "success below the threshold is still sampled out",
			method: testStreamFullMethod,
			policy: accessLogPolicy{
				successSampleRate: 0,
				slowThreshold:     time.Second,
			},
			handlerDelay: time.Second - time.Nanosecond,
			wantLog:      false,
		},
		{
			name:             "missing correlation fails open",
			missingRequestID: true,
			method:           testStreamFullMethod,
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

				err := asStreamInterceptor(accessLogAround(log, testCase.policy))(
					nil,
					testServerStream{ctx: ctx},
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

	_, err := asUnaryInterceptor(accessLogAround(log, accessLogPolicy{successSampleRate: 1}))(
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
	err = asStreamInterceptor(accessLogAround(log, accessLogPolicy{successSampleRate: 1}))(
		nil,
		testServerStream{ctx: t.Context()},
		&grpc.StreamServerInfo{FullMethod: testStreamFullMethod},
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
	unary := asUnaryInterceptor(limiter.around)
	streaming := asStreamInterceptor(limiter.around)

	release, firstDone := occupyAdmissionSlot(t, unary)

	streamHandlerCalled := false
	err := streaming(
		nil,
		testServerStream{ctx: t.Context()},
		&grpc.StreamServerInfo{FullMethod: testStreamFullMethod},
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

// TestHealthPrefixExemptsMethodsGRPCGoAddsLater pins the over-matching half of a
// deliberate pair. isHealthMethod matches the whole standard health service by
// prefix, so a method grpc-go adds to it later is exempt from admission with no
// edit here; failing to exempt one costs the service its admission budget under
// probe load.
//
// oidcjwt's TestGRPCAuthnBoundaryExactHealthAllowlist pins the other half — that
// same future method must still be authenticated, because over-matching a trust
// boundary publishes an RPC nobody meant to publish. Both drive the method name
// below, so the two halves stay about one hypothetical method.
func TestHealthPrefixExemptsMethodsGRPCGoAddsLater(t *testing.T) {
	const futureHealthMethod = healthMethodPrefix + "Future"

	load := &recordingLoad{}
	limiter := newAdmissionLimiter(1, load)
	unary := asUnaryInterceptor(limiter.around)

	release, occupied := occupyAdmissionSlot(t, unary)

	handlerCalled := false
	if _, err := unary(
		t.Context(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: futureHealthMethod},
		func(context.Context, any) (any, error) {
			handlerCalled = true
			return struct{}{}, nil
		},
	); err != nil {
		t.Fatalf("future health method %q was shed against a full budget: %v", futureHealthMethod, err)
	}
	if !handlerCalled {
		t.Fatalf("future health method %q did not reach its handler", futureHealthMethod)
	}

	close(release)
	if err := <-occupied; err != nil {
		t.Fatalf("admitted unary RPC error = %v", err)
	}
	if _, shed := load.snapshot(); shed != 0 {
		t.Fatalf("shed = %d, want 0; a health-service method must not consume the budget", shed)
	}
}

// occupyAdmissionSlot parks a unary RPC inside its handler, so the limiter the
// interceptor was built from is full when the caller drives its own RPC. Close
// the returned channel to let that RPC finish, then read its result.
func occupyAdmissionSlot(t *testing.T, unary grpc.UnaryServerInterceptor) (chan struct{}, <-chan error) {
	t.Helper()

	entered := make(chan struct{})
	release := make(chan struct{})
	occupied := make(chan error, 1)
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
		occupied <- err
	}()
	<-entered

	return release, occupied
}

func TestMapErrorAppliesEachBoundarysTrustRule(t *testing.T) {
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
		trusted    trustedStatus
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
		{
			// The policy boundary must keep trusting a status it did not build:
			// a policy in another package cannot construct this package's marker
			// type, so collapsing anyServiceStatus into ownedStatusOnly would
			// turn every policy rejection into INTERNAL.
			name:       "policy status is service-owned output",
			err:        status.Error(codes.Unauthenticated, "credential is missing or invalid"),
			trusted:    anyServiceStatus,
			wantCode:   codes.Unauthenticated,
			wantDetail: "credential is missing or invalid",
		},
		{
			// Only a status the policy returned directly is its own output; a
			// status it merely wrapped came from somewhere it does not own.
			name:       "policy wrapped status is not",
			err:        fmt.Errorf("call dependency: %w", status.Error(codes.PermissionDenied, "dependency secret")),
			trusted:    anyServiceStatus,
			wantCode:   codes.Internal,
			wantDetail: "request failed",
		},
		{
			name:       "policy raw error",
			err:        errors.New("authentication dependency credential=secret"),
			trusted:    anyServiceStatus,
			wantCode:   codes.Internal,
			wantDetail: "request failed",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			trusted := testCase.trusted
			if trusted == nil {
				trusted = ownedStatusOnly
			}
			got := mapError(testCase.err, trusted, testCase.mappers)
			if code := status.Code(got); code != testCase.wantCode {
				t.Fatalf("code = %s, want %s", code, testCase.wantCode)
			}
			if detail := status.Convert(got).Message(); detail != testCase.wantDetail {
				t.Fatalf("detail = %q, want %q", detail, testCase.wantDetail)
			}
		})
	}
}

// Recovery is one policy, but a panic has to unwind through a different shim for
// each RPC kind, so both are driven here. A shim that let the panic escape would
// take down the process instead of failing this test politely.
func TestRecoveryReturnsSanitizedOwnedStatus(t *testing.T) {
	const panicValue = "credential=secret"
	log := slog.New(slog.DiscardHandler)

	t.Run("unary", func(t *testing.T) {
		response, err := asUnaryInterceptor(recoveryAround(log))(
			t.Context(),
			nil,
			&grpc.UnaryServerInfo{FullMethod: testUnaryFullMethod},
			func(context.Context, any) (any, error) {
				panic(panicValue)
			},
		)
		if response != nil {
			t.Fatalf("panic response = %v, want nil", response)
		}
		assertRecoveredPanicStatus(t, err)
	})

	t.Run("streaming", func(t *testing.T) {
		err := asStreamInterceptor(recoveryAround(log))(
			nil,
			testServerStream{ctx: t.Context()},
			&grpc.StreamServerInfo{FullMethod: testStreamFullMethod},
			func(any, grpc.ServerStream) error {
				panic(panicValue)
			},
		)
		assertRecoveredPanicStatus(t, err)
	})
}

func assertRecoveredPanicStatus(t *testing.T, err error) {
	t.Helper()

	assertStatusCode(t, err, codes.Internal)
	if detail := status.Convert(err).Message(); detail != "request failed" {
		t.Fatalf("panic status detail = %q, want sanitized detail", detail)
	}
	if _, ok := errors.AsType[*ownedStatusError](err); !ok {
		t.Fatalf("panic error type = %T, want ownedStatusError", err)
	}
}

// recordingLoad records admission decisions for the unit tests above, which read
// an exact snapshot after the RPCs they drive have finished. performance_test.go
// declares a second LoadRecorder rather than reusing this one; it says why.
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
	ctx context.Context //nolint:containedctx // grpc.ServerStream requires Context to return the RPC context.
}

func (s testServerStream) SetHeader(metadata.MD) error  { return nil }
func (s testServerStream) SendHeader(metadata.MD) error { return nil }
func (s testServerStream) SetTrailer(metadata.MD)       {}
func (s testServerStream) Context() context.Context     { return s.ctx }
func (s testServerStream) SendMsg(any) error            { return nil }
func (s testServerStream) RecvMsg(any) error            { return nil }
