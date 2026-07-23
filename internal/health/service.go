package health

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
)

type Service struct {
	probes   []Probe
	draining atomic.Bool
}

type Probe interface {
	Name() string
	Check(ctx context.Context) error
}

var ErrDraining = errors.New("service is draining")

func New(probes ...Probe) *Service {
	return &Service{probes: slices.Clone(probes)}
}

func (s *Service) Ready(ctx context.Context) error {
	if s.draining.Load() {
		return ErrDraining
	}
	for _, probe := range s.probes {
		if err := probe.Check(ctx); err != nil {
			return fmt.Errorf("%s probe failed: %w", probe.Name(), err)
		}
	}
	return nil
}

func (s *Service) StartDrain() {
	s.draining.Store(true)
}
