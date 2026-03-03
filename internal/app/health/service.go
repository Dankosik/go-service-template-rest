package health

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
)

type Service struct {
	probes   []Probe
	draining atomic.Bool
}

//go:generate -command mockgen go tool mockgen
//go:generate mockgen -source=service.go -destination=zz_probe_mock_test.go -package=health
type Probe interface {
	Name() string
	Check(ctx context.Context) error
}

var ErrDraining = errors.New("service is draining")

func New(probes ...Probe) *Service {
	items := make([]Probe, len(probes))
	copy(items, probes)
	return &Service{probes: items}
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
