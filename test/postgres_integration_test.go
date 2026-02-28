//go:build integration

package integration_test

import (
	"context"
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

	container, err := tcpostgres.Run(
		ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("app"),
		tcpostgres.WithUsername("app"),
		tcpostgres.WithPassword("app"),
	)
	if err != nil {
		if isDockerUnavailable(err) {
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

	pool, err := postgres.New(ctx, dsn)
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

func isDockerUnavailable(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "cannot connect to the docker daemon") ||
		strings.Contains(msg, "docker socket") ||
		strings.Contains(msg, "no such host")
}
