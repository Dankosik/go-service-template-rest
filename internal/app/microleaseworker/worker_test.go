package microleaseworker

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeProbe struct {
	name string
	err  error
}

func (p fakeProbe) Name() string { return p.name }

func (p fakeProbe) Check(context.Context) error { return p.err }

type workerObserver struct {
	mu     sync.Mutex
	labels []string
}

func (o *workerObserver) ObserveWorkerTask(role, result, reasonClass string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.labels = append(o.labels, role+"|"+result+"|"+reasonClass)
}

func (o *workerObserver) snapshot() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]string(nil), o.labels...)
}

func TestWorkerRequiresEveryDurableMicroleaseRole(t *testing.T) {
	t.Parallel()

	_, err := New(Config{}, nil, []Task{
		taskForRole(RoleTerminalConsumer, nil),
		taskForRole(RoleCheckpointConsumer, nil),
		taskForRole(RoleCloseConsumer, nil),
		taskForRole(RoleInboxRetry, nil),
		taskForRole(RoleOutboxRelay, nil),
		taskForRole(RoleStaleReconciliation, nil),
	}, nil)
	if err == nil {
		t.Fatalf("New() error = nil, want missing admission renewal")
	}
	if !strings.Contains(err.Error(), RoleAdmissionControlRenew) {
		t.Fatalf("error = %v, want missing role %s", err, RoleAdmissionControlRenew)
	}
}

func TestWorkerReadinessDependsOnStartupDependencyProbes(t *testing.T) {
	t.Parallel()

	worker, err := New(Config{ReadinessTimeout: 20 * time.Millisecond}, []Probe{fakeProbe{name: "redpanda", err: errors.New("down")}}, allTasks(nil), nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := worker.Ready(context.Background()); !errors.Is(err, ErrNotReady) {
		t.Fatalf("Ready() error = %v, want ErrNotReady", err)
	}
	if err := worker.Run(context.Background()); !errors.Is(err, ErrNotReady) {
		t.Fatalf("Run() error = %v, want ErrNotReady", err)
	}
}

func TestWorkerRunsEveryRoleAndStopsOnContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	seen := make(map[string]bool)
	var seenMu sync.Mutex
	tasks := allTasks(func(role string) func(context.Context) error {
		return func(context.Context) error {
			seenMu.Lock()
			seen[role] = true
			if len(seen) == len(requiredRoles) {
				cancel()
			}
			seenMu.Unlock()
			return nil
		}
	})
	observer := &workerObserver{}
	worker, err := New(Config{DefaultInterval: time.Hour, ShutdownTimeout: time.Second}, []Probe{fakeProbe{name: "postgres"}, fakeProbe{name: "redpanda"}}, tasks, observer)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := worker.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := worker.Ready(context.Background()); !errors.Is(err, ErrNotReady) {
		t.Fatalf("Ready() after stop = %v, want ErrNotReady", err)
	}
	seenMu.Lock()
	defer seenMu.Unlock()
	for _, role := range requiredRoles {
		if !seen[role] {
			t.Fatalf("role %s did not run", role)
		}
	}
	assertWorkerObserverLabelsSafe(t, observer.snapshot())
}

func TestWorkerBoundsTerminalConcurrencyWhereRowLocksMatter(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{}, 8)
	release := make(chan struct{})
	var mu sync.Mutex
	active := 0
	maxActive := 0

	tasks := allTasks(func(role string) func(context.Context) error {
		if role != RoleTerminalConsumer {
			return func(context.Context) error { return nil }
		}
		return func(ctx context.Context) error {
			mu.Lock()
			active++
			if active > maxActive {
				maxActive = active
			}
			mu.Unlock()
			started <- struct{}{}
			select {
			case <-release:
			case <-ctx.Done():
			}
			mu.Lock()
			active--
			mu.Unlock()
			return nil
		}
	})
	for i := range tasks {
		if tasks[i].Role == RoleTerminalConsumer {
			tasks[i].Interval = time.Millisecond
			tasks[i].MaxConcurrency = 1
		}
	}

	worker, err := New(Config{DefaultInterval: 5 * time.Millisecond, ShutdownTimeout: time.Second}, nil, tasks, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- worker.Run(ctx)
	}()

	<-started
	time.Sleep(15 * time.Millisecond)
	close(release)
	cancel()

	if err := <-done; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if maxActive != 1 {
		t.Fatalf("max terminal concurrency = %d, want 1", maxActive)
	}
}

func taskForRole(role string, run func(context.Context) error) Task {
	if run == nil {
		run = func(context.Context) error { return nil }
	}
	return Task{Role: role, Interval: time.Hour, MaxConcurrency: 1, Run: run}
}

func allTasks(factory func(string) func(context.Context) error) []Task {
	tasks := make([]Task, 0, len(requiredRoles))
	for _, role := range requiredRoles {
		var run func(context.Context) error
		if factory != nil {
			run = factory(role)
		}
		tasks = append(tasks, taskForRole(role, run))
	}
	return tasks
}

func assertWorkerObserverLabelsSafe(t *testing.T, labels []string) {
	t.Helper()
	text := strings.Join(labels, "\n")
	for _, forbidden := range []string{"acct_", "request-", "trace-", "microlease-", "debit-"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("observer labels leaked forbidden value %q: %s", forbidden, text)
		}
	}
}
