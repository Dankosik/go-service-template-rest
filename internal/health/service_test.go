package health

import (
	"context"
	"errors"
	"testing"
)

func TestServiceReadySuccess(t *testing.T) {
	t.Parallel()

	db := fakeProbe{name: "db"}
	cache := fakeProbe{name: "cache"}

	svc := New(db, cache)

	if err := svc.Ready(context.Background()); err != nil {
		t.Fatalf("Ready() error = %v", err)
	}
}

func TestServiceReadyFail(t *testing.T) {
	t.Parallel()

	downErr := errors.New("down")
	db := fakeProbe{name: "db", err: downErr}

	svc := New(db)

	err := svc.Ready(context.Background())
	if err == nil {
		t.Fatalf("Ready() expected error")
	}

	if !errors.Is(err, downErr) {
		t.Fatalf("Ready() error = %v, want wrapped %v", err, downErr)
	}
}

func TestServiceReadyDraining(t *testing.T) {
	t.Parallel()

	probe := fakeProbe{name: "db"}

	svc := New(probe)
	svc.StartDrain()

	err := svc.Ready(context.Background())
	if !errors.Is(err, ErrDraining) {
		t.Fatalf("Ready() error = %v, want ErrDraining", err)
	}
}

type fakeProbe struct {
	name string
	err  error
}

func (p fakeProbe) Name() string {
	return p.name
}

func (p fakeProbe) Check(context.Context) error {
	return p.err
}
