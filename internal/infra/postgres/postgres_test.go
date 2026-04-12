package postgres

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

func TestProbeAddress(t *testing.T) {
	t.Parallel()

	t.Run("valid dsn", func(t *testing.T) {
		t.Parallel()

		address, err := ProbeAddress("postgres://user:pass@localhost:5432/app?sslmode=disable")
		if err != nil {
			t.Fatalf("ProbeAddress() error = %v", err)
		}
		if address != "localhost:5432" {
			t.Fatalf("ProbeAddress() = %q, want localhost:5432", address)
		}
	})

	t.Run("invalid dsn is redacted", func(t *testing.T) {
		t.Parallel()

		rawDSN := "postgres://user:top-secret%@localhost:5432/app"
		_, err := ProbeAddress(rawDSN)
		if err == nil {
			t.Fatal("ProbeAddress() error = nil, want non-nil")
		}
		if !errors.Is(err, ErrConfig) {
			t.Fatalf("ProbeAddress() error = %v, want ErrConfig", err)
		}
		if !strings.Contains(err.Error(), "parse postgres dsn") || !strings.Contains(err.Error(), "redacted") {
			t.Fatalf("ProbeAddress() error = %v, want redacted parse context", err)
		}
		for _, leaked := range []string{rawDSN, "top-secret", "user"} {
			if strings.Contains(err.Error(), leaked) {
				t.Fatalf("ProbeAddress() error = %v, leaked %q", err, leaked)
			}
		}
	})

	t.Run("invalid probe target shape", func(t *testing.T) {
		t.Parallel()

		_, err := ProbeAddress("user=user password=pass host=/var/run/postgresql dbname=app")
		if err == nil {
			t.Fatal("ProbeAddress() error = nil, want non-nil")
		}
		if !errors.Is(err, ErrConfig) {
			t.Fatalf("ProbeAddress() error = %v, want ErrConfig", err)
		}
		if !strings.Contains(err.Error(), "invalid postgres probe address") {
			t.Fatalf("ProbeAddress() error = %v, want invalid probe target context", err)
		}
	})
}

func TestPoolHelpersWithoutConnection(t *testing.T) {
	t.Parallel()

	var nilPool *Pool
	nilPool.Close()

	if err := nilPool.Check(context.Background()); err == nil {
		t.Fatal("(*Pool)(nil).Check() error = nil, want non-nil")
	} else if !errors.Is(err, ErrHealthcheck) {
		t.Fatalf("(*Pool)(nil).Check() error = %v, want ErrHealthcheck", err)
	}

	pool := &Pool{}
	if got := pool.Name(); got != "postgres" {
		t.Fatalf("Name() = %q, want %q", got, "postgres")
	}

	pool.Close()
	if err := pool.Check(context.Background()); err == nil {
		t.Fatal("Check() error = nil, want non-nil for nil internal pool")
	} else if !errors.Is(err, ErrHealthcheck) {
		t.Fatalf("Check() error = %v, want ErrHealthcheck", err)
	}
}

func TestMaxIdleConnLimiter(t *testing.T) {
	t.Parallel()

	limiter := newMaxIdleConnLimiter(2)
	first := &pgx.Conn{}
	second := &pgx.Conn{}
	third := &pgx.Conn{}

	if !limiter.afterRelease(first) {
		t.Fatal("afterRelease(first) = false, want true")
	}
	if !limiter.afterRelease(second) {
		t.Fatal("afterRelease(second) = false, want true")
	}
	if limiter.afterRelease(third) {
		t.Fatal("afterRelease(third) = true, want false when max idle is full")
	}

	limiter.beforeAcquire(first)
	if !limiter.afterRelease(third) {
		t.Fatal("afterRelease(third) after first acquire = false, want true")
	}

	limiter.beforeClose(second)
	if !limiter.afterRelease(first) {
		t.Fatal("afterRelease(first) after second close = false, want true")
	}

	disabled := newMaxIdleConnLimiter(0)
	if disabled.afterRelease(&pgx.Conn{}) {
		t.Fatal("afterRelease() with max idle 0 = true, want false")
	}
}

func TestMaxIdleConnLimiterConcurrentReleases(t *testing.T) {
	t.Parallel()

	limiter := newMaxIdleConnLimiter(2)
	conns := make([]*pgx.Conn, 10)
	for i := range conns {
		conns[i] = &pgx.Conn{}
	}

	var wg sync.WaitGroup
	kept := make(chan bool, len(conns))
	for _, conn := range conns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			kept <- limiter.afterRelease(conn)
		}()
	}
	wg.Wait()
	close(kept)

	var keepCount int
	for keep := range kept {
		if keep {
			keepCount++
		}
	}
	if keepCount != 2 {
		t.Fatalf("kept releases = %d, want 2", keepCount)
	}
}

func TestInstallMaxIdleConnLimiterComposesPoolHooks(t *testing.T) {
	t.Parallel()

	var beforeAcquireCalled bool
	var afterReleaseCalled bool
	var beforeCloseCalled bool
	poolConfig := &pgxpool.Config{
		BeforeAcquire: func(context.Context, *pgx.Conn) bool {
			beforeAcquireCalled = true
			return true
		},
		AfterRelease: func(*pgx.Conn) bool {
			afterReleaseCalled = true
			return true
		},
		BeforeClose: func(*pgx.Conn) {
			beforeCloseCalled = true
		},
	}
	first := &pgx.Conn{}
	second := &pgx.Conn{}

	installMaxIdleConnLimiter(poolConfig, 1)

	if !poolConfig.AfterRelease(first) {
		t.Fatal("AfterRelease(first) = false, want true")
	}
	if !afterReleaseCalled {
		t.Fatal("original AfterRelease was not called")
	}
	if poolConfig.AfterRelease(second) {
		t.Fatal("AfterRelease(second) = true, want false when max idle is full")
	}

	if !poolConfig.BeforeAcquire(context.Background(), first) {
		t.Fatal("BeforeAcquire(first) = false, want true")
	}
	if !beforeAcquireCalled {
		t.Fatal("original BeforeAcquire was not called")
	}
	if !poolConfig.AfterRelease(second) {
		t.Fatal("AfterRelease(second) after first acquire = false, want true")
	}

	poolConfig.BeforeClose(second)
	if !beforeCloseCalled {
		t.Fatal("original BeforeClose was not called")
	}
}
