package postgres

import (
	"context"
	"strings"
	"testing"
)

func TestNewRejectsEmptyDSN(t *testing.T) {
	t.Parallel()

	_, err := New(context.Background(), "   \n\t")
	if err == nil {
		t.Fatal("New() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "postgres dsn is empty") {
		t.Fatalf("New() error = %q, want to contain %q", err.Error(), "postgres dsn is empty")
	}
}

func TestPoolHelpersWithoutConnection(t *testing.T) {
	t.Parallel()

	var nilPool *Pool
	nilPool.Close()

	if err := nilPool.Check(context.Background()); err == nil {
		t.Fatal("(*Pool)(nil).Check() error = nil, want non-nil")
	}

	pool := &Pool{}
	if got := pool.Name(); got != "postgres" {
		t.Fatalf("Name() = %q, want %q", got, "postgres")
	}
	if got := pool.DB(); got != nil {
		t.Fatalf("DB() = %v, want nil", got)
	}

	pool.Close()
	if err := pool.Check(context.Background()); err == nil {
		t.Fatal("Check() error = nil, want non-nil for nil internal pool")
	}
}
