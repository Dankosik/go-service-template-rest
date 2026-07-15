package evalfixture

import (
	"context"
	"errors"
)

type rawRepository interface {
	Fetch(context.Context, string) ([]byte, error)
}

type decoder interface {
	Decode([]byte) (*Order, error)
}

type Loader struct {
	repo    rawRepository
	decoder decoder
}

func (l *Loader) Load(ctx context.Context, id string) (*Order, error) {
	raw, err := l.repo.Fetch(ctx, id)
	if err != nil {
		return nil, simplifiedError(err)
	}
	order, err := l.decoder.Decode(raw)
	if err != nil {
		return nil, simplifiedError(err)
	}
	return order, nil
}

func simplifiedError(err error) error {
	if err == nil {
		return nil
	}
	return errors.New("load failed")
}
