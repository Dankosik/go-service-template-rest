package grpcclient_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

const transparentRetryMethod = "/grpcclient.test.Transparent/Call"

func TestTransparentRetryReappliesClosedPropagationPerAttempt(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		policy        grpcclient.PropagationPolicy
		wantTrace     bool
		wantRequestID bool
	}{
		{name: "none", policy: grpcclient.PropagationNone},
		{name: "trace context", policy: grpcclient.PropagationTraceContext, wantTrace: true},
		{
			name:          "trusted service",
			policy:        grpcclient.PropagationTrustedService,
			wantTrace:     true,
			wantRequestID: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			peer := startTransparentRetryPeer(t)
			recorder := tracetest.NewSpanRecorder()
			tracerProvider := sdktrace.NewTracerProvider(
				sdktrace.WithSampler(sdktrace.AlwaysSample()),
				sdktrace.WithSpanProcessor(recorder),
			)
			t.Cleanup(func() {
				if err := tracerProvider.Shutdown(context.Background()); err != nil {
					t.Errorf("TracerProvider.Shutdown() error = %v", err)
				}
			})

			connection, err := grpcclient.New(
				grpcclient.DefaultConfig("passthrough:///"+peer.address),
				grpcclient.Options{
					TransportCredentials: insecure.NewCredentials(),
					TracerProvider:       tracerProvider,
					Propagation:          testCase.policy,
				},
			)
			if err != nil {
				t.Fatalf("grpcclient.New() error = %v", err)
			}
			t.Cleanup(func() {
				if err := connection.Close(); err != nil {
					t.Errorf("ClientConn.Close() error = %v", err)
				}
			})

			const requestID = "transparent_retry_request_123"
			callerMetadata := metadata.MD{
				"TraceParent":   {"00-11111111111111111111111111111111-2222222222222222-01"},
				"TRACESTATE":    {"vendor=stale"},
				"Baggage":       {"private=value"},
				"X-Request-ID":  {"stale-request"},
				"authorization": {"Bearer retained"},
			}
			ctx := metadata.NewOutgoingContext(t.Context(), callerMetadata)
			ctx = reqctx.ContextWithRequestID(ctx, requestID)
			ctx, parent := tracerProvider.Tracer("transparent-retry-test").Start(ctx, "parent")
			parentTraceID := parent.SpanContext().TraceID()

			callCtx, cancelCall := context.WithTimeout(ctx, 3*time.Second)
			err = connection.Invoke(
				callCtx,
				transparentRetryMethod,
				&emptypb.Empty{},
				&emptypb.Empty{},
			)
			cancelCall()
			if err != nil {
				parent.End()
				t.Fatalf("transparent retry Invoke() error = %v", err)
			}

			attempts := peer.awaitAttempts(t)
			spans := clientAttemptSpans(recorder.Ended(), parentTraceID)
			if len(spans) != 2 {
				parent.End()
				t.Fatalf("client attempt spans = %d, want 2", len(spans))
			}
			assertTransparentRetryCorrelation(
				t,
				attempts,
				spans,
				parentTraceID,
				requestID,
				testCase.wantTrace,
				testCase.wantRequestID,
			)
			parent.End()

			if got := callerMetadata["TraceParent"]; len(got) != 1 ||
				got[0] != "00-11111111111111111111111111111111-2222222222222222-01" {
				t.Fatalf("caller traceparent = %v, want unchanged", got)
			}
			if got := callerMetadata["Baggage"]; len(got) != 1 || got[0] != "private=value" {
				t.Fatalf("caller baggage = %v, want unchanged", got)
			}
		})
	}
}

func assertTransparentRetryCorrelation(
	t *testing.T,
	attempts []metadata.MD,
	spans []sdktrace.ReadOnlySpan,
	parentTraceID trace.TraceID,
	requestID string,
	wantTrace bool,
	wantRequestID bool,
) {
	t.Helper()

	if len(attempts) != 2 {
		t.Fatalf("wire attempts = %d, want 2", len(attempts))
	}
	spanIDs := make(map[trace.SpanID]struct{}, len(spans))
	for _, span := range spans {
		if span.SpanContext().TraceID() != parentTraceID {
			t.Fatalf(
				"attempt span trace ID = %s, want parent trace ID %s",
				span.SpanContext().TraceID(),
				parentTraceID,
			)
		}
		spanIDs[span.SpanContext().SpanID()] = struct{}{}
	}
	if len(spanIDs) != 2 {
		t.Fatalf("distinct attempt span IDs = %d, want 2", len(spanIDs))
	}

	wireSpanIDs := make(map[trace.SpanID]struct{}, len(attempts))
	for index, attempt := range attempts {
		traceparents := attempt.Get("traceparent")
		if (len(traceparents) == 1) != wantTrace {
			t.Fatalf(
				"attempt %d traceparent = %v, want present %v",
				index+1,
				traceparents,
				wantTrace,
			)
		}
		if wantTrace {
			carrier := propagation.MapCarrier{"traceparent": traceparents[0]}
			remote := trace.SpanContextFromContext(
				propagation.TraceContext{}.Extract(context.Background(), carrier),
			)
			if remote.TraceID() != parentTraceID {
				t.Fatalf(
					"attempt %d propagated trace ID = %s, want %s",
					index+1,
					remote.TraceID(),
					parentTraceID,
				)
			}
			if _, ok := spanIDs[remote.SpanID()]; !ok {
				t.Fatalf(
					"attempt %d propagated parent span ID = %s, want a recorded attempt span",
					index+1,
					remote.SpanID(),
				)
			}
			wireSpanIDs[remote.SpanID()] = struct{}{}
		}

		requestIDs := attempt.Get("x-request-id")
		if (len(requestIDs) == 1) != wantRequestID {
			t.Fatalf(
				"attempt %d x-request-id = %v, want present %v",
				index+1,
				requestIDs,
				wantRequestID,
			)
		}
		if wantRequestID && requestIDs[0] != requestID {
			t.Fatalf("attempt %d x-request-id = %v, want %q", index+1, requestIDs, requestID)
		}
		for _, forbidden := range []string{"tracestate", "baggage"} {
			if got := attempt.Get(forbidden); len(got) != 0 {
				t.Fatalf("attempt %d %s = %v, want absent", index+1, forbidden, got)
			}
		}
		if got := attempt.Get("authorization"); len(got) != 1 || got[0] != "Bearer retained" {
			t.Fatalf("attempt %d authorization = %v, want retained", index+1, got)
		}
	}
	if wantTrace && len(wireSpanIDs) != 2 {
		t.Fatalf("distinct propagated attempt parents = %d, want 2", len(wireSpanIDs))
	}
}

func clientAttemptSpans(
	spans []sdktrace.ReadOnlySpan,
	traceID trace.TraceID,
) []sdktrace.ReadOnlySpan {
	attempts := make([]sdktrace.ReadOnlySpan, 0, 2)
	for _, span := range spans {
		if span.SpanKind() == trace.SpanKindClient && span.SpanContext().TraceID() == traceID {
			attempts = append(attempts, span)
		}
	}
	return attempts
}

type transparentRetryPeer struct {
	address  string
	attempts <-chan []metadata.MD
}

func startTransparentRetryPeer(t *testing.T) *transparentRetryPeer {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("transparent retry peer Listen() error = %v", err)
	}
	attempts := make(chan []metadata.MD, 1)
	done := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		serveErr := serveTransparentRetryConnection(connection, attempts)
		closeErr := connection.Close()
		if serveErr != nil {
			done <- serveErr
			return
		}
		if closeErr != nil {
			done <- fmt.Errorf("close transparent retry connection: %w", closeErr)
			return
		}
		done <- nil
	}()
	t.Cleanup(func() {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("transparent retry listener Close() error = %v", err)
		}
		select {
		case err := <-done:
			if err != nil &&
				!errors.Is(err, net.ErrClosed) &&
				!errors.Is(err, io.EOF) &&
				!errors.Is(err, syscall.ECONNRESET) {
				t.Errorf("transparent retry peer error = %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("transparent retry peer did not stop")
		}
	})
	return &transparentRetryPeer{
		address:  listener.Addr().String(),
		attempts: attempts,
	}
}

func (p *transparentRetryPeer) awaitAttempts(t *testing.T) []metadata.MD {
	t.Helper()

	select {
	case attempts := <-p.attempts:
		return attempts
	case <-time.After(2 * time.Second):
		t.Fatal("transparent retry peer did not report two attempts")
		return nil
	}
}

func serveTransparentRetryConnection(
	connection net.Conn,
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
			headers := metadata.MD{}
			for _, field := range frame.Fields {
				headers.Append(strings.ToLower(field.Name), field.Value)
			}
			captured = append(captured, headers)
			if len(captured) == 1 {
				if err := framer.WriteRSTStream(
					frame.StreamID,
					http2.ErrCodeRefusedStream,
				); err != nil {
					return fmt.Errorf("refuse first stream: %w", err)
				}
				continue
			}
			if len(captured) == 2 {
				if err := writeTransparentRetrySuccess(framer, frame.StreamID); err != nil {
					return err
				}
				result <- captured
			}
		}
	}
}

func writeTransparentRetrySuccess(framer *http2.Framer, streamID uint32) error {
	if err := writeHPACKHeaderBlock(
		framer,
		streamID,
		false,
		hpack.HeaderField{Name: ":status", Value: "200"},
		hpack.HeaderField{Name: "content-type", Value: "application/grpc"},
	); err != nil {
		return fmt.Errorf("write success headers: %w", err)
	}
	if err := framer.WriteData(streamID, false, []byte{0, 0, 0, 0, 0}); err != nil {
		return fmt.Errorf("write empty gRPC response: %w", err)
	}
	if err := writeHPACKHeaderBlock(
		framer,
		streamID,
		true,
		hpack.HeaderField{Name: "grpc-status", Value: "0"},
	); err != nil {
		return fmt.Errorf("write success trailers: %w", err)
	}
	return nil
}

func writeHPACKHeaderBlock(
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
