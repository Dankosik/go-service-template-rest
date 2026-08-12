package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/waittest"
)

type lifecycleEngineStub struct {
	mu         sync.Mutex
	drainCalls int
	facts      postgresjobs.EngineFacts
	run        func(context.Context) error
	drain      func(context.Context) postgresjobs.DrainResult
}

func (s *lifecycleEngineStub) Run(ctx context.Context) error {
	if s.run != nil {
		return s.run(ctx)
	}
	<-ctx.Done()
	return ctx.Err()
}
func (s *lifecycleEngineStub) Facts() postgresjobs.EngineFacts { return s.facts }
func (s *lifecycleEngineStub) StartDrain(ctx context.Context) postgresjobs.DrainResult {
	s.mu.Lock()
	s.drainCalls++
	s.mu.Unlock()
	if s.drain != nil {
		return s.drain(ctx)
	}
	return postgresjobs.DrainResult{CleanupSafe: true}
}

func jobsWorkerLifecycleConfig(t *testing.T) config.Config {
	t.Helper()
	cfg := config.Config{}
	cfg.Observability.Metrics.Addr = waittest.FreeTCPAddr(t, "jobs worker diagnostics")
	cfg.HTTP.GracePeriod = time.Second
	cfg.Jobs.PollInterval = time.Millisecond
	cfg.Jobs.DrainTimeout = time.Millisecond
	return cfg
}

func TestJobsWorkerLifecycleDrainsOnceAfterSignal(t *testing.T) {
	signalCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine := &lifecycleEngineStub{facts: postgresjobs.EngineFacts{ClaimAdmissionOpen: true, Compatible: true, ObservationFresh: true}}
	cfg := jobsWorkerLifecycleConfig(t)
	result := make(chan lifecycleResult, 1)
	go func() { result <- runLifecycle(signalCtx, context.Background(), cfg, telemetry.New(), engine) }()
	time.Sleep(10 * time.Millisecond)
	cancel()
	got := <-result
	if got.Err != nil || !got.CleanupSafe {
		t.Fatalf("runLifecycle() = %+v, want safe signal drain", got)
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.drainCalls != 1 {
		t.Fatalf("StartDrain calls = %d, want 1", engine.drainCalls)
	}
}

func TestJobsWorkerLifecycleWithdrawsReadinessBeforeDrain(t *testing.T) {
	signalCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := jobsWorkerLifecycleConfig(t)
	readyStatus := make(chan int, 1)
	engine := &lifecycleEngineStub{facts: postgresjobs.EngineFacts{ClaimAdmissionOpen: true, Compatible: true, ObservationFresh: true}}
	engine.drain = func(context.Context) postgresjobs.DrainResult {
		response, err := http.Get("http://" + cfg.Observability.Metrics.Addr + "/health/ready")
		if err != nil {
			return postgresjobs.DrainResult{CleanupSafe: true, Err: err}
		}
		defer response.Body.Close()
		readyStatus <- response.StatusCode
		return postgresjobs.DrainResult{CleanupSafe: true}
	}
	result := make(chan lifecycleResult, 1)
	go func() { result <- runLifecycle(signalCtx, context.Background(), cfg, telemetry.New(), engine) }()
	client := &http.Client{Timeout: 100 * time.Millisecond}
	waittest.Until(t, time.Second, func() bool {
		response, err := client.Get("http://" + cfg.Observability.Metrics.Addr + "/health/ready")
		if err != nil {
			return false
		}
		defer response.Body.Close()
		return response.StatusCode == http.StatusOK
	}, "jobs worker initial readiness")
	cancel()
	got := <-result
	if got.Err != nil || !got.CleanupSafe {
		t.Fatalf("runLifecycle() = %+v, want safe readiness withdrawal", got)
	}
	if status := <-readyStatus; status != http.StatusServiceUnavailable {
		t.Fatalf("readiness during StartDrain = %d, want %d", status, http.StatusServiceUnavailable)
	}
}

func TestJobsWorkerLifecycleDrainsAfterTerminalEngineFailure(t *testing.T) {
	terminal := errors.New("terminal engine failure")
	engine := &lifecycleEngineStub{
		facts: postgresjobs.EngineFacts{ClaimAdmissionOpen: true, Compatible: true, ObservationFresh: true},
		run:   func(context.Context) error { return terminal },
	}
	got := runLifecycle(context.Background(), context.Background(), jobsWorkerLifecycleConfig(t), telemetry.New(), engine)
	if !errors.Is(got.Err, terminal) || !got.CleanupSafe {
		t.Fatalf("runLifecycle() = %+v, want safe terminal drain", got)
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.drainCalls != 1 {
		t.Fatalf("StartDrain calls = %d, want 1", engine.drainCalls)
	}
}

func TestJobsWorkerLifecycleRetainsDependenciesAfterUnsafeDrain(t *testing.T) {
	unsafe := errors.New("attempt did not join")
	signalCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine := &lifecycleEngineStub{
		facts: postgresjobs.EngineFacts{ClaimAdmissionOpen: true, Compatible: true, ObservationFresh: true},
		drain: func(context.Context) postgresjobs.DrainResult {
			return postgresjobs.DrainResult{Err: unsafe}
		},
	}
	cfg := jobsWorkerLifecycleConfig(t)
	result := make(chan lifecycleResult, 1)
	go func() {
		result <- runLifecycle(signalCtx, t.Context(), cfg, telemetry.New(), engine)
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()
	got := <-result
	if !errors.Is(got.Err, unsafe) || got.CleanupSafe {
		t.Fatalf("runLifecycle() = %+v, want unsafe cleanup result", got)
	}
}
