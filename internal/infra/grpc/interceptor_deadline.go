package grpcx

import (
	"context"
	"time"
)

// deadlineAround caps how long the work below it may occupy a handler.
//
// It derives the RPC context's deadline rather than replacing it, so a caller
// deadline that is already earlier still wins: context.WithTimeout never extends
// a parent's. That makes this a cap by construction rather than by a comparison
// anyone has to write.
//
// A non-positive timeout disables the cap, which is how the stream bound ships.
// Health RPCs are exempt from a business budget entirely: a Watch is a
// standing subscription rather than work a caller waits on, and [admissionPolicy]
// bounds how many may run at once instead.
func deadlineAround(timeout time.Duration) aroundRPC {
	return func(ctx context.Context, fullMethod string, call func(context.Context) error) error {
		if timeout <= 0 || isHealthMethod(fullMethod) {
			return call(ctx)
		}
		bounded, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return call(bounded)
	}
}
