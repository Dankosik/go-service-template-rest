package jobs

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestJobsRegistry(t *testing.T) {
	v1 := testDefinition(t, Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"})
	v2Input := testDefinitionInput(Revision{Kind: "email", ArgsVersion: "v2", PolicyVersion: "p1"})
	v2Input.Policy.MaxAttemptDuration = 90 * time.Second
	v2, err := NewDefinition(v2Input)
	requireError(t, err)
	registry := NewRegistry()
	called := 0
	handler := func(_ context.Context, input HandlerInput[testArgs]) HandlerResult {
		called++
		if input.Arguments.Task != "send" {
			t.Fatalf("handler arguments = %+v", input.Arguments)
		}
		return HandlerResult{Outcome: OutcomeSuccess, Effect: EffectCompleted}
	}
	if err := Register(registry, v1, handler); err != nil {
		t.Fatalf("Register(v1) error = %v", err)
	}
	if err := Register(registry, v2, handler); err != nil {
		t.Fatalf("Register(v2) error = %v", err)
	}
	if err := Register(registry, v1, func(context.Context, HandlerInput[testArgs]) HandlerResult {
		called += 100
		return HandlerResult{Outcome: OutcomePermanent, Effect: EffectNone}
	}); !errors.Is(err, ErrDuplicateRevision) {
		t.Fatalf("duplicate Register error = %v", err)
	}

	wantKeys := []Revision{v1.Key(), v2.Key()}
	if got := registry.Keys(); !reflect.DeepEqual(got, wantKeys) {
		t.Fatalf("Keys() = %+v, want %+v", got, wantKeys)
	}
	for _, omitted := range wantKeys {
		partial := NewRegistry()
		for _, definition := range []Definition[testArgs]{v1, v2} {
			if definition.Key() != omitted {
				requireError(t, Register(partial, definition, handler))
			}
		}
		if err := partial.Require(wantKeys); !errors.Is(err, ErrUnsupportedRevision) {
			t.Fatalf("Require() with omitted %v error = %v", omitted, err)
		}
	}

	registered, err := registry.Lookup(v1.Key())
	requireError(t, err)
	if registered.Key() != v1.Key() {
		t.Fatalf("Lookup().Key() = %+v, want %+v", registered.Key(), v1.Key())
	}
	if registered.MaxAttemptDuration() != time.Minute {
		t.Fatalf("Lookup().MaxAttemptDuration() = %s, want %s", registered.MaxAttemptDuration(), time.Minute)
	}
	registeredV2, err := registry.Lookup(v2.Key())
	requireError(t, err)
	if registeredV2.MaxAttemptDuration() != 90*time.Second {
		t.Fatalf("Lookup(v2).MaxAttemptDuration() = %s, want %s", registeredV2.MaxAttemptDuration(), 90*time.Second)
	}
	input := DispatchInput{
		Payload:  []byte(`{"task":"send","count":1,"metadata":{"a":"1"}}`),
		Identity: testIdentity(), AttemptGeneration: 1,
	}
	result, err := registered.Dispatch(context.Background(), input)
	requireError(t, err)
	if result.Outcome != OutcomeSuccess || called != 1 {
		t.Fatalf("valid dispatch result=%+v called=%d", result, called)
	}

	poison := [][]byte{
		[]byte(`{"count":1,"metadata":{}}`),
		[]byte(`{"task":"send","count":1,"metadata":{},"future":true}`),
		[]byte(`{"task":"send","count":1,"metadata":{}} {}`),
		[]byte(`{"task":`),
	}
	for _, payload := range poison {
		candidate := input
		candidate.Payload = payload
		if _, err := registered.Dispatch(context.Background(), candidate); !errors.Is(err, ErrPoisonPayload) {
			t.Fatalf("Dispatch(%q) error = %v, want ErrPoisonPayload", payload, err)
		}
	}
	if called != 1 {
		t.Fatalf("poison payload invoked handler; called=%d", called)
	}

	unknown := Revision{Kind: "email", ArgsVersion: "v3", PolicyVersion: "p1"}
	if _, err := registry.Lookup(unknown); !errors.Is(err, ErrUnsupportedRevision) {
		t.Fatalf("Lookup(unknown) error = %v", err)
	}
}
