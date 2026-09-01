// profile:inbound-webhooks-standard:start
package inboundwebhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrInvalidBinding   = errors.New("inbound webhook binding is invalid")
	ErrDuplicateBinding = errors.New("inbound webhook binding is duplicate")
	ErrUnknownBinding   = errors.New("inbound webhook binding is unknown")
	ErrDecodeRejected   = errors.New("inbound webhook decode rejected")
)

// VerifiedDelivery is the verified durable receipt a feature handler sees.
type VerifiedDelivery struct {
	EndpointID string
	DeliveryID string
	SignedAt   time.Time
	Body       json.RawMessage
	ReceivedAt time.Time
}

type dispatchHandle func(context.Context, VerifiedDelivery) error

// Registry holds exactly one typed decoder/handler per endpoint.
type Registry struct {
	bindings map[string]dispatchHandle
}

// NewRegistry returns an empty typed binding registry.
func NewRegistry() *Registry {
	return &Registry{bindings: make(map[string]dispatchHandle)}
}

// Bind registers one non-nil decoder and handler for endpointID.
func (r *Registry) Bind[T any](
	endpointID string,
	decode func(json.RawMessage) (T, error),
	handle func(context.Context, VerifiedDelivery, T) error,
) error {
	if r == nil || endpointID == "" || decode == nil || handle == nil {
		return ErrInvalidBinding
	}
	if r.bindings == nil {
		r.bindings = make(map[string]dispatchHandle)
	}
	if _, exists := r.bindings[endpointID]; exists {
		return ErrDuplicateBinding
	}
	r.bindings[endpointID] = func(ctx context.Context, delivery VerifiedDelivery) error {
		value, err := decode(json.RawMessage(bytes.Clone(delivery.Body)))
		if err != nil {
			return decodeError{err: err}
		}
		return handle(ctx, delivery, value)
	}
	return nil
}

// HasBinding reports whether endpointID has a decoder and handler.
func (r *Registry) HasBinding(endpointID string) bool {
	if r == nil {
		return false
	}
	_, ok := r.bindings[endpointID]
	return ok
}

// RequireExact requires the registered set to equal configured.
func (r *Registry) RequireExact(configured []string) error {
	if r == nil {
		return ErrInvalidBinding
	}
	if len(configured) != len(r.bindings) {
		return ErrInvalidBinding
	}
	seen := make(map[string]struct{}, len(configured))
	for _, id := range configured {
		if _, duplicate := seen[id]; duplicate {
			return ErrInvalidBinding
		}
		seen[id] = struct{}{}
		if _, ok := r.bindings[id]; !ok {
			return ErrInvalidBinding
		}
	}
	return nil
}

// Dispatch decodes and handles one verified delivery for its endpoint.
func (r *Registry) Dispatch(ctx context.Context, delivery VerifiedDelivery) error {
	if r == nil {
		return ErrUnknownBinding
	}
	binding, ok := r.bindings[delivery.EndpointID]
	if !ok {
		return ErrUnknownBinding
	}
	return binding(ctx, delivery)
}

type decodeError struct{ err error }

func (e decodeError) Error() string { return e.err.Error() }
func (e decodeError) Unwrap() error { return e.err }

// IsDecodeError reports a decoder failure, including typed rejection.
func IsDecodeError(err error) bool {
	_, ok := errors.AsType[decodeError](err)
	return ok
}

// profile:inbound-webhooks-standard:end
