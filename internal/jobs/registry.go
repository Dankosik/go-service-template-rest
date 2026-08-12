package jobs

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"
)

var (
	ErrDuplicateRevision   = errors.New("duplicate jobs revision")
	ErrUnsupportedRevision = errors.New("unsupported jobs revision")
	ErrPoisonPayload       = errors.New("poison jobs payload")
)

type HandlerInput[A any] struct {
	Arguments          A
	Identity           AcceptanceIdentity
	AttemptGeneration  uint64
	RecoveryGeneration uint64
}

type HandlerResult struct {
	Outcome   OutcomeClass
	Effect    EffectStatus
	RetryHint time.Duration
}

func (r HandlerResult) Validate() error {
	if !r.Outcome.valid() || !r.Effect.valid() || r.RetryHint < 0 {
		return fmt.Errorf("%w: invalid handler result", ErrInvalidTransition)
	}
	return nil
}

type Handler[A any] func(context.Context, HandlerInput[A]) HandlerResult

type DispatchInput struct {
	Payload            []byte
	Identity           AcceptanceIdentity
	AttemptGeneration  uint64
	RecoveryGeneration uint64
}

type Registered struct {
	key                Revision
	maxAttemptDuration time.Duration
	dispatch           func(context.Context, DispatchInput) (HandlerResult, error)
	evaluate           func(AttemptFacts) (Transition, error)
}

func (r Registered) Key() Revision { return r.key }

func (r Registered) MaxAttemptDuration() time.Duration { return r.maxAttemptDuration }

func (r Registered) Dispatch(ctx context.Context, input DispatchInput) (HandlerResult, error) {
	if r.dispatch == nil {
		return HandlerResult{}, fmt.Errorf("%w: registration is empty", ErrUnsupportedRevision)
	}
	return r.dispatch(ctx, input)
}

func (r Registered) Evaluate(facts AttemptFacts) (Transition, error) {
	if r.evaluate == nil {
		return Transition{}, fmt.Errorf("%w: registration is empty", ErrUnsupportedRevision)
	}
	return r.evaluate(facts)
}

type Registry struct {
	entries map[Revision]Registered
}

func NewRegistry() *Registry { return &Registry{} }

func Register[A any](registry *Registry, definition Definition[A], handler Handler[A]) error {
	if registry == nil {
		return fmt.Errorf("%w: registry is nil", ErrInvalidDefinition)
	}
	if err := definition.valid(); err != nil {
		return err
	}
	if handler == nil {
		return fmt.Errorf("%w: handler is required", ErrInvalidDefinition)
	}
	if registry.entries == nil {
		registry.entries = make(map[Revision]Registered)
	}
	key := definition.Key()
	if _, exists := registry.entries[key]; exists {
		return fmt.Errorf("%w: %s/%s/%s", ErrDuplicateRevision, key.Kind, key.ArgsVersion, key.PolicyVersion)
	}
	registry.entries[key] = Registered{
		key:                key,
		maxAttemptDuration: definition.policy.MaxAttemptDuration,
		evaluate:           definition.Evaluate,
		dispatch: func(ctx context.Context, input DispatchInput) (HandlerResult, error) {
			if err := input.Identity.Validate(); err != nil {
				return HandlerResult{}, err
			}
			if input.AttemptGeneration == 0 {
				return HandlerResult{}, fmt.Errorf("%w: attempt_generation must be positive", ErrInvalidTransition)
			}
			args, err := definition.Decode(input.Payload)
			if err != nil {
				return HandlerResult{}, fmt.Errorf("%w: %w", ErrPoisonPayload, err)
			}
			result := handler(ctx, HandlerInput[A]{
				Arguments: args, Identity: input.Identity,
				AttemptGeneration: input.AttemptGeneration, RecoveryGeneration: input.RecoveryGeneration,
			})
			if err := result.Validate(); err != nil {
				return HandlerResult{}, err
			}
			return result, nil
		},
	}
	return nil
}

func (r *Registry) Lookup(key Revision) (Registered, error) {
	if err := key.Validate(); err != nil {
		return Registered{}, err
	}
	if r == nil {
		return Registered{}, fmt.Errorf("%w: registry is nil", ErrUnsupportedRevision)
	}
	registered, ok := r.entries[key]
	if !ok {
		return Registered{}, fmt.Errorf("%w: %s/%s/%s", ErrUnsupportedRevision, key.Kind, key.ArgsVersion, key.PolicyVersion)
	}
	return registered, nil
}

func (r *Registry) Keys() []Revision {
	if r == nil {
		return nil
	}
	return slices.SortedFunc(maps.Keys(r.entries), compareRevision)
}

func (r *Registry) Require(keys []Revision) error {
	for _, key := range keys {
		if _, err := r.Lookup(key); err != nil {
			return err
		}
	}
	return nil
}

func compareRevision(a, b Revision) int {
	if a.Kind != b.Kind {
		return cmp.Compare(a.Kind, b.Kind)
	}
	if a.ArgsVersion != b.ArgsVersion {
		return cmp.Compare(a.ArgsVersion, b.ArgsVersion)
	}
	return cmp.Compare(a.PolicyVersion, b.PolicyVersion)
}
