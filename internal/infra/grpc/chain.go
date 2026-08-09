package grpcx

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

// aroundRPC is one server policy, written once for both RPC kinds. It wraps the
// work below it: it may observe or replace that work's result, decline to run it
// at all, and choose the context it runs under. [asUnaryInterceptor] and
// [asStreamInterceptor] adapt it to grpc-go's two separate interceptor types;
// the package doc owns why a policy is written this way rather than against
// those types directly.
//
// A policy that only observes passes down the context it received. Deriving one
// is what lets a policy bound the work below it, which is the whole of what
// [deadlineAround] needs and the reason this shape takes the context rather than
// closing over it.
type aroundRPC func(ctx context.Context, fullMethod string, call func(context.Context) error) error

// asUnaryInterceptor adapts one policy to the unary interceptor type.
//
// The response is returned even alongside a non-nil error, as the hand-written
// interceptors this replaced did: grpc-go discards it, and an outer interceptor
// that inspects it sees what it saw before. When the handler panics, the
// assignment below never runs, so recovery still answers with a nil response.
func asUnaryInterceptor(around aroundRPC) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		var response any
		err := around(ctx, info.FullMethod, func(callCtx context.Context) error {
			var callErr error
			response, callErr = handler(callCtx, request)
			return callErr
		})
		return response, err
	}
}

// asStreamInterceptor adapts one policy to the streaming interceptor type.
//
// A streaming handler reads its context from the stream, so a policy that
// derived one is only visible through a replacement stream. The identity check
// keeps that allocation on the policies that actually derive: an observing
// policy hands back the context it received, and the handler sees the original
// stream.
func asStreamInterceptor(around aroundRPC) grpc.StreamServerInterceptor {
	return func(
		server any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := stream.Context()
		return around(ctx, info.FullMethod, func(callCtx context.Context) error {
			if callCtx == ctx {
				return handler(server, stream)
			}
			return handler(server, serverStreamWithContext{ServerStream: stream, ctx: callCtx})
		})
	}
}

// Names for the built-in policies, used by [builtinPolicies] and by the
// benchmark variants in performance_test.go that measure a deliberate subset of
// the chain.
const (
	builtinAccessLog           = "access_log"
	builtinDeadline            = "deadline"
	builtinRecovery            = "recovery"
	builtinAdmission           = "admission"
	builtinPolicyErrorBoundary = "policy_error_boundary"
)

// builtinPolicy is one entry in this package's own policy chain, carrying the
// name the benchmark refers to it by.
type builtinPolicy struct {
	name   string
	around aroundRPC
}

// builtinPolicies returns this package's own policies, outermost first. Every
// position is load-bearing, and the package doc owns the order, what each one
// buys, and why this is a function rather than a literal — read it before
// reordering this list.
//
// Correlation and the handler error boundary are not in the list because they do
// not sit next to each other: correlation is outermost and is the one policy
// still written per RPC kind, while the handler boundary must land inside the
// supplied policy.
//
// timeout is the one value that differs between the two chains, which is why
// NewServer calls this once per chain rather than sharing one list. Producing
// both lists from this function is what still makes it impossible for a policy
// to reach unary RPCs and miss streaming ones.
func builtinPolicies(
	log *slog.Logger,
	accessLogs accessLogPolicy,
	admission admissionPolicy,
	timeout time.Duration,
) []builtinPolicy {
	return []builtinPolicy{
		{name: builtinAccessLog, around: accessLogAround(log, accessLogs)},
		{name: builtinDeadline, around: deadlineAround(timeout)},
		{name: builtinRecovery, around: recoveryAround(log)},
		{name: builtinAdmission, around: admission.around},
		{name: builtinPolicyErrorBoundary, around: policyErrorBoundary(log)},
	}
}

// unaryChain and streamChain assemble the one interceptor order this transport
// serves, outermost first: correlation, the builtins, the supplied policy, then
// the handler error boundary. Both RPC kinds and the benchmark variants in
// performance_test.go build their chains here, so a position cannot drift
// between them; the package doc owns what each position buys.
//
// log reaches correlation only. Every policy below it takes the logger it needs
// through builtins, and correlation is not one of them because it is the one
// policy still written per RPC kind.
func unaryChain(
	log *slog.Logger,
	builtins []builtinPolicy,
	supplied []grpc.UnaryServerInterceptor,
	handlerErrors aroundRPC,
) []grpc.UnaryServerInterceptor {
	chain := []grpc.UnaryServerInterceptor{correlationUnaryInterceptor(log)}
	for _, policy := range builtins {
		chain = append(chain, asUnaryInterceptor(policy.around))
	}
	chain = append(chain, supplied...)
	return append(chain, asUnaryInterceptor(handlerErrors))
}

func streamChain(
	log *slog.Logger,
	builtins []builtinPolicy,
	supplied []grpc.StreamServerInterceptor,
	handlerErrors aroundRPC,
) []grpc.StreamServerInterceptor {
	chain := []grpc.StreamServerInterceptor{correlationStreamInterceptor(log)}
	for _, policy := range builtins {
		chain = append(chain, asStreamInterceptor(policy.around))
	}
	chain = append(chain, supplied...)
	return append(chain, asStreamInterceptor(handlerErrors))
}
