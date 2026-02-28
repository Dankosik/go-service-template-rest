package health

import (
	"context"
	"fmt"

	"github.com/example/go-service-template-rest/internal/domain"
)

type Service struct {
	probes []domain.ReadinessProbe
}

func New(probes ...domain.ReadinessProbe) *Service {
	items := make([]domain.ReadinessProbe, len(probes))
	copy(items, probes)
	return &Service{probes: items}
}

func (s *Service) Ready(ctx context.Context) error {
	for _, probe := range s.probes {
		if err := probe.Check(ctx); err != nil {
			return fmt.Errorf("%s probe failed: %w", probe.Name(), err)
		}
	}
	return nil
}
