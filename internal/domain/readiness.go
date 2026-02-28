package domain

import "context"

type ReadinessProbe interface {
	Name() string
	Check(ctx context.Context) error
}
