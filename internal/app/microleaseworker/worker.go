package microleaseworker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrInvalidWorker = errors.New("microlease worker invalid")
	ErrNotReady      = errors.New("microlease worker not ready")
)

const (
	RoleTerminalConsumer      = "terminal_consumer"
	RoleCheckpointConsumer    = "checkpoint_consumer"
	RoleCloseConsumer         = "close_consumer"
	RoleInboxRetry            = "inbox_retry"
	RoleOutboxRelay           = "outbox_relay"
	RoleStaleReconciliation   = "stale_reconciliation"
	RoleAdmissionControlRenew = "admission_control_renewal"
)

var requiredRoles = []string{
	RoleTerminalConsumer,
	RoleCheckpointConsumer,
	RoleCloseConsumer,
	RoleInboxRetry,
	RoleOutboxRelay,
	RoleStaleReconciliation,
	RoleAdmissionControlRenew,
}

type Probe interface {
	Name() string
	Check(context.Context) error
}

type Task struct {
	Role           string
	Interval       time.Duration
	MaxConcurrency int
	Run            func(context.Context) error
}

type Observer interface {
	ObserveWorkerTask(role, result, reasonClass string)
}

type Config struct {
	ReadinessTimeout time.Duration
	DefaultInterval  time.Duration
	ShutdownTimeout  time.Duration
}

type Worker struct {
	cfg      Config
	probes   []Probe
	tasks    []Task
	observer Observer

	mu    sync.RWMutex
	ready bool
}

func New(cfg Config, probes []Probe, tasks []Task, observer Observer) (*Worker, error) {
	cfg = normalizeConfig(cfg)
	if err := validateTasks(tasks); err != nil {
		return nil, err
	}
	return &Worker{
		cfg:      cfg,
		probes:   append([]Probe(nil), probes...),
		tasks:    normalizeTasks(cfg, tasks),
		observer: observer,
	}, nil
}

func (w *Worker) Ready(ctx context.Context) error {
	if w == nil {
		return fmt.Errorf("%w: worker is nil", ErrInvalidWorker)
	}
	w.mu.RLock()
	ready := w.ready
	w.mu.RUnlock()
	if !ready {
		return ErrNotReady
	}
	for _, probe := range w.probes {
		probeCtx, cancel := context.WithTimeout(ctx, w.cfg.ReadinessTimeout)
		err := probe.Check(probeCtx)
		cancel()
		if err != nil {
			return fmt.Errorf("%w: probe %s: %w", ErrNotReady, safeRoleLabel(probe.Name()), err)
		}
	}
	return nil
}

func (w *Worker) Run(ctx context.Context) error {
	if w == nil {
		return fmt.Errorf("%w: worker is nil", ErrInvalidWorker)
	}
	if err := w.checkProbes(ctx); err != nil {
		return err
	}
	w.setReady(true)
	defer w.setReady(false)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	for _, task := range w.tasks {
		wg.Go(func() {
			w.runTaskLoop(runCtx, task)
		})
	}

	<-ctx.Done()
	cancel()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(w.cfg.ShutdownTimeout):
		return fmt.Errorf("%w: shutdown timeout", ErrInvalidWorker)
	}
}

func (w *Worker) checkProbes(ctx context.Context) error {
	for _, probe := range w.probes {
		probeCtx, cancel := context.WithTimeout(ctx, w.cfg.ReadinessTimeout)
		err := probe.Check(probeCtx)
		cancel()
		if err != nil {
			return fmt.Errorf("%w: dependency %s: %w", ErrNotReady, safeRoleLabel(probe.Name()), err)
		}
	}
	return nil
}

func (w *Worker) runTaskLoop(ctx context.Context, task Task) {
	sem := make(chan struct{}, task.MaxConcurrency)
	var taskWG sync.WaitGroup
	start := func() {
		select {
		case sem <- struct{}{}:
		default:
			w.observe(task.Role, "skipped", "concurrency_limit")
			return
		}
		taskWG.Go(func() {
			defer func() { <-sem }()
			if err := task.Run(ctx); err != nil {
				w.observe(task.Role, "error", "task_error")
				return
			}
			w.observe(task.Role, "success", "")
		})
	}

	start()
	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			taskWG.Wait()
			return
		case <-ticker.C:
			start()
		}
	}
}

func (w *Worker) observe(role, result, reasonClass string) {
	if w.observer == nil {
		return
	}
	w.observer.ObserveWorkerTask(safeRoleLabel(role), safeResultLabel(result), safeReasonLabel(reasonClass))
}

func (w *Worker) setReady(ready bool) {
	w.mu.Lock()
	w.ready = ready
	w.mu.Unlock()
}

func normalizeConfig(cfg Config) Config {
	if cfg.ReadinessTimeout <= 0 {
		cfg.ReadinessTimeout = time.Second
	}
	if cfg.DefaultInterval <= 0 {
		cfg.DefaultInterval = time.Second
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 5 * time.Second
	}
	return cfg
}

func normalizeTasks(cfg Config, tasks []Task) []Task {
	normalized := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if task.Interval <= 0 {
			task.Interval = cfg.DefaultInterval
		}
		if task.MaxConcurrency <= 0 {
			task.MaxConcurrency = 1
		}
		normalized = append(normalized, task)
	}
	return normalized
}

func validateTasks(tasks []Task) error {
	seen := make(map[string]bool, len(tasks))
	for _, task := range tasks {
		if task.Role == "" {
			return fmt.Errorf("%w: task role is required", ErrInvalidWorker)
		}
		if task.Run == nil {
			return fmt.Errorf("%w: task %s runner is required", ErrInvalidWorker, safeRoleLabel(task.Role))
		}
		seen[task.Role] = true
	}
	for _, role := range requiredRoles {
		if !seen[role] {
			return fmt.Errorf("%w: required task %s is missing", ErrInvalidWorker, role)
		}
	}
	return nil
}

func safeRoleLabel(role string) string {
	switch role {
	case RoleTerminalConsumer, RoleCheckpointConsumer, RoleCloseConsumer, RoleInboxRetry, RoleOutboxRelay, RoleStaleReconciliation, RoleAdmissionControlRenew, "postgres", "redpanda":
		return role
	default:
		return "other"
	}
}

func safeResultLabel(result string) string {
	switch result {
	case "success", "error", "skipped":
		return result
	default:
		return "other"
	}
}

func safeReasonLabel(reason string) string {
	switch reason {
	case "", "task_error", "concurrency_limit":
		return reason
	default:
		return "other"
	}
}
