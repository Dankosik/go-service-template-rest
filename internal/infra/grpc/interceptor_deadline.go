package grpcx

import (
	"context"
	"time"
)

// deadlineAround caps how long a unary handler may run. Only the unary chain
// installs it.
//
// It derives the RPC context's deadline rather than replacing it, so a caller
// deadline that is already earlier still wins: context.WithTimeout never extends
// a parent's. That makes this a cap by construction rather than by a comparison
// anyone has to write.
//
// A non-positive timeout disables the cap. Standard health RPCs are exempt from
// this business budget; their own budget in [admissionPolicy] bounds them.
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
