// The interceptor policies as units: access-log decisions, the RPC deadline,
// admission accounting, panic recovery, and the two error boundaries' trust
// rules.
//
// These drive the policies directly rather than through a server, so a failure
// names the rule that broke. server_test.go and telemetry_test.go cover the same
// rules as a caller and an operator see them, and deadline_test.go and
// admission_test.go cover what only a composed server can show.

package grpcx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/failure"
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

func TestOneAdmissionPolicyServesBothInterceptorTypes(t *testing.T) {
	load := &recordingLoad{}
	policy := newAdmissionPolicy(1, 1, load)
	unary := asUnaryInterceptor(policy.around)
	streaming := asStreamInterceptor(policy.around)

	release, firstDone := occupyAdmissionSlot(t, unary, testUnaryFullMethod)

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

// TestHealthServiceHoldsItsOwnAdmissionBudget drives the separation in both
// directions, because either one failing alone reproduces a different outage.
//
// Business saturation must not shed a health watch: a caller whose watch is
// refused stops selecting the backend entirely, so a partial overload would
// become a total one. Health saturation must not shed business work: a hostile
// peer opening watches would otherwise decide what the service may serve.
//
// Check answers under both, because the platform probe is what tells an operator
// the instance is alive while either budget is full.
func TestHealthServiceHoldsItsOwnAdmissionBudget(t *testing.T) {
	const (
		checkMethod        = healthMethodPrefix + "Check"
		watchMethod        = healthMethodPrefix + "Watch"
		futureHealthMethod = healthMethodPrefix + "Future"
	)

	load := &recordingLoad{}
	unary := asUnaryInterceptor(newAdmissionPolicy(1, 1, load).around)

	releaseBusiness, business := occupyAdmissionSlot(t, unary, testUnaryFullMethod)
	assertAdmitted(t, unary, checkMethod)
	assertAdmitted(t, unary, watchMethod)
	assertShed(t, unary, testUnaryFullMethod)

	releaseHealth, health := occupyAdmissionSlot(t, unary, watchMethod)
	assertAdmitted(t, unary, checkMethod)
	assertShed(t, unary, futureHealthMethod)

	close(releaseBusiness)
	if err := <-business; err != nil {
		t.Fatalf("admitted business RPC error = %v", err)
	}
	assertAdmitted(t, unary, testUnaryFullMethod)

	close(releaseHealth)
	if err := <-health; err != nil {
		t.Fatalf("admitted health RPC error = %v", err)
	}

	// Only the business shed is recorded: active and shed are the capacity signal
	// MaxConcurrentRPCs is sized from, and a health-service decision is not one.
	if _, shed := load.snapshot(); shed != 1 {
		t.Fatalf("shed = %d, want only the business RPC refused by its own budget", shed)
	}
}

func assertAdmitted(t *testing.T, unary grpc.UnaryServerInterceptor, fullMethod string) {
	t.Helper()

	handlerCalled := false
	if _, err := unary(
		t.Context(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: fullMethod},
		func(context.Context, any) (any, error) {
			handlerCalled = true
			return struct{}{}, nil
		},
	); err != nil {
		t.Fatalf("%s was refused: %v", fullMethod, err)
	}
	if !handlerCalled {
		t.Fatalf("%s did not reach its handler", fullMethod)
	}
}

func assertShed(t *testing.T, unary grpc.UnaryServerInterceptor, fullMethod string) {
	t.Helper()

	handlerCalled := false
	_, err := unary(
		t.Context(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: fullMethod},
		func(context.Context, any) (any, error) {
			handlerCalled = true
			return struct{}{}, nil
		},
	)
	assertStatusCode(t, err, codes.ResourceExhausted)
	if handlerCalled {
		t.Fatalf("%s bypassed a full admission budget", fullMethod)
	}
}

// occupyAdmissionSlot parks a unary RPC on fullMethod inside its handler, so the
// budget owning that method is full when the caller drives its own RPC. Close
// the returned channel to let that RPC finish, then read its result.
func occupyAdmissionSlot(
	t *testing.T,
	unary grpc.UnaryServerInterceptor,
	fullMethod string,
) (chan struct{}, <-chan error) {
	t.Helper()

	entered := make(chan struct{})
	release := make(chan struct{})
	occupied := make(chan error, 1)
	go func() {
		_, err := unary(
			t.Context(),
			nil,
			&grpc.UnaryServerInfo{FullMethod: fullMethod},
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
	mapper := func(err error) (failure.Classification, bool) {
		if !errors.Is(err, sentinel) {
			return failure.Classification{}, false
		}
		return failure.Classification{Code: failure.CodeNotFound, Detail: "record not found"}, true
	}
	broadMapper := func(error) (failure.Classification, bool) {
		return failure.Classification{Code: failure.CodeAlreadyExists, Detail: "classified"}, true
	}

	// wantRecorded is the second result: the error reached the caller as a
	// generic INTERNAL carrying nothing about what failed, so the boundary owes
	// it a record. It is stated per case rather than derived from the status,
	// because a mapper that classifies its error as failure.CodeInternalError
	// produces the same code and detail while being a deliberate answer.
	for _, testCase := range []struct {
		name         string
		err          error
		trusted      trustedStatus
		mappers      []failure.Mapper
		wantCode     codes.Code
		wantDetail   string
		wantRecorded bool
	}{
		{name: "canceled", err: context.Canceled, mappers: []failure.Mapper{broadMapper}, wantCode: codes.Canceled, wantDetail: "request canceled"},
		{
			name:       "deadline",
			err:        context.DeadlineExceeded,
			mappers:    []failure.Mapper{broadMapper},
			wantCode:   codes.DeadlineExceeded,
			wantDetail: "request deadline exceeded",
		},
		{
			name:       "domain",
			err:        sentinel,
			mappers:    []failure.Mapper{mapper},
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
			name:         "unmarked downstream",
			err:          status.Error(codes.PermissionDenied, "dependency secret"),
			wantCode:     codes.Internal,
			wantDetail:   "request failed",
			wantRecorded: true,
		},
		{
			name:         "raw",
			err:          errors.New("password=secret"),
			wantCode:     codes.Internal,
			wantDetail:   "request failed",
			wantRecorded: true,
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
			name:         "policy wrapped status is not",
			err:          fmt.Errorf("call dependency: %w", status.Error(codes.PermissionDenied, "dependency secret")),
			trusted:      anyServiceStatus,
			wantCode:     codes.Internal,
			wantDetail:   "request failed",
			wantRecorded: true,
		},
		{
			name:         "policy raw error",
			err:          errors.New("authentication dependency credential=secret"),
			trusted:      anyServiceStatus,
			wantCode:     codes.Internal,
			wantDetail:   "request failed",
			wantRecorded: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			trusted := testCase.trusted
			if trusted == nil {
				trusted = ownedStatusOnly
			}
			got, recorded := mapError(testCase.err, trusted, errorRendering{mappers: testCase.mappers})
			if recorded != testCase.wantRecorded {
				t.Fatalf("sanitized = %t, want %t", recorded, testCase.wantRecorded)
			}
			if code := status.Code(got); code != testCase.wantCode {
				t.Fatalf("code = %s, want %s", code, testCase.wantCode)
			}
			if detail := status.Convert(got).Message(); detail != testCase.wantDetail {
				t.Fatalf("detail = %q, want %q", detail, testCase.wantDetail)
			}
		})
	}
}

// TestHandlerErrorBoundaryRecordsTheUnclassifiedFailure is the counterpart to
// the canary tests in telemetry_test.go. Those prove the record leaks nothing;
// this proves there is a record at all, which is what an INTERNAL that carries
// no detail depends on to be diagnosable.
func TestHandlerErrorBoundaryRecordsTheUnclassifiedFailure(t *testing.T) {
	const secretCanary = "password=handler-boundary-secret"

	var logged bytes.Buffer
	boundary := handlerErrorBoundary(slogJSONLogger(&logged), errorRendering{})

	err := boundary(t.Context(), testPayloadFullMethod, func(context.Context) error {
		return fmt.Errorf("store order: %w", &net.OpError{Err: errors.New(secretCanary)})
	})

	if code := status.Code(err); code != codes.Internal {
		t.Fatalf("code = %s, want %s", code, codes.Internal)
	}
	if strings.Contains(logged.String(), secretCanary) {
		t.Fatalf("record discloses the error's own text: %s", logged.String())
	}

	// Decoded as a map for the reason telemetry_test.go does: the attribute keys
	// are the OpenTelemetry dotted names an operator queries on, and a struct tag
	// carrying one is not the snake_case the repository's tagliatelle rule wants.
	var record map[string]any
	if err := json.Unmarshal(logged.Bytes(), &record); err != nil {
		t.Fatalf("decode record %q: %v", logged.String(), err)
	}
	if got := record["level"]; got != slog.LevelError.String() {
		t.Errorf("level = %v, want %s", got, slog.LevelError)
	}
	if got := record["rpc.method"]; got != testPayloadFullMethod {
		t.Errorf("rpc.method = %v, want %q", got, testPayloadFullMethod)
	}
	if chain := asString(record["error_chain"]); !strings.Contains(chain, "*net.OpError") {
		t.Errorf("error_chain = %q, want the wrapped dependency type", chain)
	}
}

// TestHandlerErrorBoundaryDoesNotRecordADeliberateStatus keeps the error stream
// usable: a classified domain failure and a status the service chose are answers,
// not faults, and repeating them at ERROR is how an error rate stops meaning
// anything.
func TestHandlerErrorBoundaryDoesNotRecordADeliberateStatus(t *testing.T) {
	sentinel := errors.New("record not found")
	classify := func(err error) (failure.Classification, bool) {
		if !errors.Is(err, sentinel) {
			return failure.Classification{}, false
		}
		return failure.Classification{Code: failure.CodeNotFound, Detail: "record not found"}, true
	}

	for _, testCase := range []struct {
		name string
		err  error
	}{
		{name: "classified domain failure", err: fmt.Errorf("load order: %w", sentinel)},
		{name: "owned status", err: ownedStatus(codes.Unavailable, "service draining")},
		{name: "caller cancellation", err: context.Canceled},
		{name: "spent deadline", err: context.DeadlineExceeded},
		{name: "success", err: nil},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var logged bytes.Buffer
			boundary := handlerErrorBoundary(
				slogJSONLogger(&logged),
				errorRendering{mappers: []failure.Mapper{classify}},
			)

			_ = boundary(t.Context(), testPayloadFullMethod, func(context.Context) error { return testCase.err })

			if logged.Len() != 0 {
				t.Fatalf("wrote %q, want no record", logged.String())
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
	// setHeaderErr drives the one branch a real stream reaches only once it is
	// already dead, which is why it is a field rather than a second fake.
	setHeaderErr error
}

func (s testServerStream) SetHeader(metadata.MD) error  { return s.setHeaderErr }
func (s testServerStream) SendHeader(metadata.MD) error { return nil }
func (s testServerStream) SetTrailer(metadata.MD)       {}
func (s testServerStream) Context() context.Context     { return s.ctx }
func (s testServerStream) SendMsg(any) error            { return nil }
func (s testServerStream) RecvMsg(any) error            { return nil }

// TestCorrelationFailureIsAnsweredAndRecorded covers the one RPC this transport
// answers without an access-log record.
//
// Correlation is the outermost interceptor, so an RPC it refuses never reaches
// the access log below it and would otherwise leave the server with no trace of
// a status it returned. The unary half drives the real branch: grpc.SetHeader
// refuses a context carrying no server transport stream.
func TestCorrelationFailureIsAnsweredAndRecorded(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		method string
		drive  func(*slog.Logger, *bool) error
	}{
		{
			name:   "unary",
			method: testUnaryFullMethod,
			drive: func(log *slog.Logger, handlerCalled *bool) error {
				_, err := correlationUnaryInterceptor(log)(
					t.Context(),
					nil,
					&grpc.UnaryServerInfo{FullMethod: testUnaryFullMethod},
					func(context.Context, any) (any, error) {
						*handlerCalled = true
						return struct{}{}, nil
					},
				)
				return err
			},
		},
		{
			name:   "stream",
			method: testStreamFullMethod,
			drive: func(log *slog.Logger, handlerCalled *bool) error {
				return correlationStreamInterceptor(log)(
					nil,
					testServerStream{ctx: t.Context(), setHeaderErr: errors.New("stream is done")},
					&grpc.StreamServerInfo{FullMethod: testStreamFullMethod},
					func(any, grpc.ServerStream) error {
						*handlerCalled = true
						return nil
					},
				)
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var output bytes.Buffer
			handlerCalled := false

			err := testCase.drive(slog.New(slog.NewJSONHandler(&output, nil)), &handlerCalled)

			assertStatusCode(t, err, codes.Internal)
			if handlerCalled {
				t.Fatal("handler ran for an RPC whose response metadata was refused")
			}
			recorded := output.String()
			if !strings.Contains(recorded, `"msg":"grpc_correlation_failed"`) ||
				!strings.Contains(recorded, strconv.Quote(testCase.method)) {
				t.Fatalf("correlation failure log = %q, want the refusal and its method", recorded)
			}
		})
	}
}

func TestDeadlineAroundCapsWithoutExtending(t *testing.T) {
	t.Run("derives a deadline when none exists", func(t *testing.T) {
		var observed time.Duration
		err := deadlineAround(time.Minute)(
			t.Context(),
			testUnaryFullMethod,
			func(ctx context.Context) error {
				deadline, ok := ctx.Deadline()
				if !ok {
					t.Fatal("policy passed down a context with no deadline")
				}
				observed = time.Until(deadline)
				return nil
			},
		)
		if err != nil {
			t.Fatalf("deadlineAround() error = %v", err)
		}
		if observed <= 0 || observed > time.Minute {
			t.Fatalf("derived deadline %s away, want within the minute bound", observed)
		}
	})

	t.Run("never extends an earlier caller deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
		defer cancel()

		var observed time.Duration
		if err := deadlineAround(time.Hour)(ctx, testUnaryFullMethod, func(ctx context.Context) error {
			deadline, _ := ctx.Deadline()
			observed = time.Until(deadline)
			return nil
		}); err != nil {
			t.Fatalf("deadlineAround() error = %v", err)
		}
		if observed > time.Second {
			t.Fatalf("derived deadline %s away, want the caller's millisecond", observed)
		}
	})

	t.Run("non-positive disables the cap", func(t *testing.T) {
		if err := deadlineAround(0)(t.Context(), testUnaryFullMethod, func(ctx context.Context) error {
			if _, ok := ctx.Deadline(); ok {
				t.Fatal("disabled policy still derived a deadline")
			}
			return nil
		}); err != nil {
			t.Fatalf("deadlineAround() error = %v", err)
		}
	})

	t.Run("health RPCs are exempt", func(t *testing.T) {
		health := healthMethodPrefix + "Check"
		if err := deadlineAround(time.Minute)(t.Context(), health, func(ctx context.Context) error {
			if _, ok := ctx.Deadline(); ok {
				t.Fatal("a probe was given a business budget")
			}
			return nil
		}); err != nil {
			t.Fatalf("deadlineAround() error = %v", err)
		}
	})
}
