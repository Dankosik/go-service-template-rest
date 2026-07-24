package postgres

import (
	"context"
	"errors"
	"testing"
)

func TestPoolNilAndZeroValueSafety(t *testing.T) {
	t.Parallel()

	var nilPool *Pool
	nilPool.Close()
	if nilPool.PGX() != nil {
		t.Fatal("nil Pool PGX() != nil")
	}
	if err := nilPool.Check(context.Background()); err == nil {
		t.Fatal("nil Pool Check() error = nil, want non-nil")
	} else if !errors.Is(err, ErrHealthcheck) {
		t.Fatalf("nil Pool Check() error = %v, want ErrHealthcheck", err)
	}

	zeroPool := &Pool{}
	zeroPool.Close()
	if zeroPool.PGX() != nil {
		t.Fatal("zero Pool PGX() != nil")
	}
	if err := zeroPool.Check(context.Background()); err == nil {
		t.Fatal("zero Pool Check() error = nil, want non-nil")
	} else if !errors.Is(err, ErrHealthcheck) {
		t.Fatalf("zero Pool Check() error = %v, want ErrHealthcheck", err)
	}
}
