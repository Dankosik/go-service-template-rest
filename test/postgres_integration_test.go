//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgresReadinessProbe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := runPostgresContainer(ctx)
	if err != nil {
		if isDockerUnavailable(err) {
			if requireDockerForIntegration() {
				t.Fatalf("docker is required for integration tests: %v", err)
			}
			t.Skipf("docker is unavailable: %v", err)
		}
		t.Fatalf("start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if termErr := testcontainers.TerminateContainer(container); termErr != nil {
			t.Errorf("terminate postgres container: %v", termErr)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("build postgres dsn: %v", err)
	}

	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       10,
		MaxIdleConns:       5,
		ConnMaxLifetime:    time.Hour,
	})
	if err != nil {
		t.Fatalf("create postgres pool: %v", err)
	}
	t.Cleanup(pool.Close)

	checkCtx, checkCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer checkCancel()

	if err := pool.Check(checkCtx); err != nil {
		t.Fatalf("readiness check failed: %v", err)
	}
}

func runPostgresContainer(ctx context.Context) (container *tcpostgres.PostgresContainer, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("testcontainers panic: %v", recovered)
		}
	}()

	container, err = tcpostgres.Run(
		ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("app"),
		tcpostgres.WithUsername("app"),
		tcpostgres.WithPassword("app"),
		tcpostgres.BasicWaitStrategies(),
	)
	return container, err
}

func isDockerUnavailable(err error) bool {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "checked path: $xdg_runtime_dir") {
		return true
	}

	return strings.Contains(msg, "cannot connect to the docker daemon") ||
		strings.Contains(msg, "is the docker daemon running") ||
		strings.Contains(msg, "error during connect") ||
		strings.Contains(msg, "docker socket") ||
		strings.Contains(msg, "no such host")
}

func requireDockerForIntegration() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("REQUIRE_DOCKER")))
	return v == "1" || v == "true" || v == "yes"
}
