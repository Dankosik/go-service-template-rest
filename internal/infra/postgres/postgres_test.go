package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewRejectsEmptyDSN(t *testing.T) {
	t.Parallel()

	_, err := New(context.Background(), Options{
		DSN:                "   \n\t",
		ConnectTimeout:     time.Second,
		HealthcheckTimeout: time.Second,
		MaxOpenConns:       10,
		MaxIdleConns:       5,
		ConnMaxLifetime:    time.Minute,
	})
	if err == nil {
		t.Fatal("New() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "postgres dsn is empty") {
		t.Fatalf("New() error = %q, want to contain %q", err.Error(), "postgres dsn is empty")
	}
	if !errors.Is(err, ErrConfig) {
		t.Fatalf("New() error = %v, want ErrConfig", err)
	}
}

func TestNewRejectsInvalidOptions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		opts Options
	}{
		{
			name: "connect timeout",
			opts: Options{
				DSN:                "postgres://user:pass@localhost:5432/db?sslmode=disable",
				HealthcheckTimeout: time.Second,
				MaxOpenConns:       10,
				MaxIdleConns:       5,
				ConnMaxLifetime:    time.Minute,
			},
		},
		{
			name: "healthcheck timeout",
			opts: Options{
				DSN:             "postgres://user:pass@localhost:5432/db?sslmode=disable",
				ConnectTimeout:  time.Second,
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Minute,
			},
		},
		{
			name: "max open conns",
			opts: Options{
				DSN:                "postgres://user:pass@localhost:5432/db?sslmode=disable",
				ConnectTimeout:     time.Second,
				HealthcheckTimeout: time.Second,
				MaxIdleConns:       5,
				ConnMaxLifetime:    time.Minute,
			},
		},
		{
			name: "max idle conns",
			opts: Options{
				DSN:                "postgres://user:pass@localhost:5432/db?sslmode=disable",
				ConnectTimeout:     time.Second,
				HealthcheckTimeout: time.Second,
				MaxOpenConns:       10,
				MaxIdleConns:       11,
				ConnMaxLifetime:    time.Minute,
			},
		},
		{
			name: "conn max lifetime",
			opts: Options{
				DSN:                "postgres://user:pass@localhost:5432/db?sslmode=disable",
				ConnectTimeout:     time.Second,
				HealthcheckTimeout: time.Second,
				MaxOpenConns:       10,
				MaxIdleConns:       5,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(context.Background(), tc.opts)
			if err == nil {
				t.Fatal("New() error = nil, want non-nil")
			}
			if !errors.Is(err, ErrConfig) {
				t.Fatalf("New() error = %v, want ErrConfig", err)
			}
		})
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

func TestShouldKeepReleasedConn(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                  string
		maxIdleConns          int32
		idleConnsBeforeReturn int32
		wantKeep              bool
	}{
		{
			name:                  "zero max idle closes released conn",
			maxIdleConns:          0,
			idleConnsBeforeReturn: 0,
			wantKeep:              false,
		},
		{
			name:                  "keep when below max",
			maxIdleConns:          2,
			idleConnsBeforeReturn: 1,
			wantKeep:              true,
		},
		{
			name:                  "close when at max",
			maxIdleConns:          2,
			idleConnsBeforeReturn: 2,
			wantKeep:              false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldKeepReleasedConn(tc.maxIdleConns, tc.idleConnsBeforeReturn)
			if got != tc.wantKeep {
				t.Fatalf("shouldKeepReleasedConn(%d, %d) = %v, want %v", tc.maxIdleConns, tc.idleConnsBeforeReturn, got, tc.wantKeep)
			}
		})
	}
}
