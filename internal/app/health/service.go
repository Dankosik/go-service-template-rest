package health

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/example/go-service-template-rest/internal/domain"
)

type Service struct {
	probes   []domain.ReadinessProbe
	draining atomic.Bool
}

var ErrDraining = errors.New("service is draining")

func New(probes ...domain.ReadinessProbe) *Service {
	items := make([]domain.ReadinessProbe, len(probes))
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
