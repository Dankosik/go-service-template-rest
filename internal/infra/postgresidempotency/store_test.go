package postgresidempotency

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
)

func TestStoreOptionsValidation(t *testing.T) {
	valid := StoreOptions{
		OwnerRecoveryDelay:     time.Second,
		CleanupBatchSize:       1,
		MaxMaintenanceLag:      time.Minute,
		MaxRelationBytes:       1024,
		AdmissionHeadroomBytes: 1,
	}
	for _, tc := range []struct {
		name    string
		mutate  func(*StoreOptions)
		wantErr bool
	}{
		{name: "valid"},
		{name: "owner recovery delay", mutate: func(o *StoreOptions) { o.OwnerRecoveryDelay = 0 }, wantErr: true},
		{name: "cleanup batch", mutate: func(o *StoreOptions) { o.CleanupBatchSize = 0 }, wantErr: true},
		{name: "maintenance lag", mutate: func(o *StoreOptions) { o.MaxMaintenanceLag = 0 }, wantErr: true},
		{name: "relation ceiling", mutate: func(o *StoreOptions) { o.MaxRelationBytes = 0 }, wantErr: true},
		{name: "headroom", mutate: func(o *StoreOptions) { o.AdmissionHeadroomBytes = 0 }, wantErr: true},
		{name: "headroom equals ceiling", mutate: func(o *StoreOptions) { o.AdmissionHeadroomBytes = o.MaxRelationBytes }, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options := valid
			if tc.mutate != nil {
				tc.mutate(&options)
			}
			err := options.validate()
			if tc.wantErr != errors.Is(err, ErrConfig) {
				t.Fatalf("StoreOptions.validate() error = %v, ErrConfig = %t", err, tc.wantErr)
			}
		})
	}
}

func TestClassificationBudgetDoesNotReset(t *testing.T) {
	deadline := time.Now().Add(time.Second)
	parent, cancel := context.WithDeadline(t.Context(), deadline)
	defer cancel()
	first, firstCancel, firstOwn := classificationContext(parent, 2*time.Second)
	defer firstCancel()
	if first == nil {
		t.Fatal("first classification context = nil")
	}
	second, secondCancel, secondOwn := classificationContext(first, 2*time.Second)
	defer secondCancel()
	if second == nil {
		t.Fatal("second classification context = nil")
	}

	if firstOwn || secondOwn {
		t.Fatalf("classification contexts own budget = %t/%t, want false/false", firstOwn, secondOwn)
	}
	firstDeadline, firstOK := first.Deadline()
	secondDeadline, secondOK := second.Deadline()
	if !firstOK || !secondOK || !firstDeadline.Equal(deadline) || !secondDeadline.Equal(deadline) {
		t.Fatalf("classification deadlines = %v/%v, want parent deadline %v", firstDeadline, secondDeadline, deadline)
	}
}

func TestStoreClassificationRequiresContext(t *testing.T) {
	var store Store
	var parent context.Context
	for _, call := range []struct {
		name string
		call func() error
	}{
		{
			name: "reserve",
			call: func() error {
				_, _, err := store.Reserve(parent, httpidempotency.Contract{}, httpidempotency.Attempt{}, nil)
				return err
			},
		},
		{
			name: "reconcile",
			call: func() error {
				_, _, err := store.Reconcile(parent, httpidempotency.Contract{}, httpidempotency.Attempt{}, nil)
				return err
			},
		},
	} {
		t.Run(call.name, func(t *testing.T) {
			if err := call.call(); !errors.Is(err, ErrConfig) {
				t.Fatalf("classification with nil context error = %v, want ErrConfig", err)
			}
		})
	}
}

func TestStoreConstructionAndFailureHelpers(t *testing.T) {
	options := StoreOptions{
		OwnerRecoveryDelay:     time.Second,
		CleanupBatchSize:       1,
		MaxMaintenanceLag:      time.Minute,
		MaxRelationBytes:       1024,
		AdmissionHeadroomBytes: 1,
	}
	if store, err := NewStore(nil, options); store != nil || !errors.Is(err, ErrConfig) {
		t.Fatalf("NewStore(nil) = %v, %v, want nil ErrConfig", store, err)
	}

	_, attempt, _ := testIdempotencyInputs(t)
	if !bytes.Equal(identityBytes(attempt), attempt.Identity[:]) {
		t.Fatal("identityBytes() did not preserve the request identity")
	}
	if !bytes.Equal(fingerprintBytes(attempt.Fingerprint), attempt.Fingerprint.Digest[:]) {
		t.Fatal("fingerprintBytes() did not preserve the request fingerprint")
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := unavailable(cancelled, "reserve"); !errors.Is(err, context.Canceled) {
		t.Fatalf("unavailable(cancelled) = %v, want context.Canceled", err)
	}
	if err := unavailable(t.Context(), "reserve"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unavailable(active) = %v, want ErrUnavailable", err)
	}
	if err := (&Store{}).Check(t.Context()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Check(unbound) = %v, want ErrUnavailable", err)
	}

	var nilStore *Store
	if nilStore.TerminalErrors() != nil {
		t.Fatal("nil Store TerminalErrors() must be nil")
	}
	nilStore.ObserveTerminal(t.Context(), httpidempotency.Decision{}, ErrIntegrityConflict)
}
