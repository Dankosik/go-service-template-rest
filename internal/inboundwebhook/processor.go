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

type decodeHandle struct {
	decode func(json.RawMessage) (any, error)
	handle func(context.Context, VerifiedDelivery, any) error
}

// Registry holds exactly one typed decoder/handler per endpoint.
type Registry struct {
	bindings map[string]decodeHandle
}

// NewRegistry returns an empty typed binding registry.
func NewRegistry() *Registry {
	return &Registry{bindings: make(map[string]decodeHandle)}
}

// Bind registers one non-nil decoder and handler for endpointID.
func Bind[T any](
	reg *Registry,
	endpointID string,
	decode func(json.RawMessage) (T, error),
	handle func(context.Context, VerifiedDelivery, T) error,
) error {
	if reg == nil || endpointID == "" || decode == nil || handle == nil {
		return ErrInvalidBinding
	}
	if _, exists := reg.bindings[endpointID]; exists {
		return ErrDuplicateBinding
	}
	reg.bindings[endpointID] = decodeHandle{
		decode: func(raw json.RawMessage) (any, error) {
			return decode(raw)
		},
		handle: func(ctx context.Context, delivery VerifiedDelivery, value any) error {
			typed, ok := value.(T)
			if !ok {
				return ErrInvalidBinding
			}
			return handle(ctx, delivery, typed)
		},
	}
	return nil
}

// EndpointIDs returns the registered endpoint set.
func (r *Registry) EndpointIDs() []string {
	if r == nil {
		return nil
	}
	ids := make([]string, 0, len(r.bindings))
	for id := range r.bindings {
		ids = append(ids, id)
	}
	return ids
}

// RequireExact requires the registered set to equal configured.
func (r *Registry) RequireExact(configured []string) error {
	if r == nil {
		return ErrInvalidBinding
	}
	if len(configured) != len(r.bindings) {
		return ErrInvalidBinding
	}
	for _, id := range configured {
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
	raw := bytes.Clone(delivery.Body)
	value, err := binding.decode(json.RawMessage(raw))
	if err != nil {
		return decodeError{err: err}
	}
	if err := binding.handle(ctx, delivery, value); err != nil {
		return handleError{err: err}
	}
	return nil
}

type decodeError struct{ err error }

func (e decodeError) Error() string { return e.err.Error() }
func (e decodeError) Unwrap() error { return e.err }

type handleError struct{ err error }

func (e handleError) Error() string { return e.err.Error() }
func (e handleError) Unwrap() error { return e.err }

// IsDecodeError reports a decoder failure, including typed rejection.
func IsDecodeError(err error) bool {
	var decoded decodeError
	return errors.As(err, &decoded)
}

// profile:inbound-webhooks-standard:end
