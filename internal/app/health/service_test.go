package health

import (
	"context"
	"errors"
	"testing"
)

type probeStub struct {
	name string
	err  error
}

func (p probeStub) Name() string {
	return p.name
}

func (p probeStub) Check(context.Context) error {
	return p.err
}

func TestServiceReadySuccess(t *testing.T) {
	svc := New(probeStub{name: "db"}, probeStub{name: "cache"})

	if err := svc.Ready(context.Background()); err != nil {
		t.Fatalf("Ready() error = %v", err)
	}
}

func TestServiceReadyFail(t *testing.T) {
	svc := New(probeStub{name: "db", err: errors.New("down")})

	if err := svc.Ready(context.Background()); err == nil {
		t.Fatalf("Ready() expected error")
	}
}

func TestServiceReadyDraining(t *testing.T) {
	svc := New()
	svc.StartDrain()

	err := svc.Ready(context.Background())
	if !errors.Is(err, ErrDraining) {
		t.Fatalf("Ready() error = %v, want ErrDraining", err)
	}
}
