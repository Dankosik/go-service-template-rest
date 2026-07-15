package evalfixture

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrRecordNotFound = errors.New("record not found")

type recordRepository interface {
	Load(context.Context, string) ([]byte, error)
}

type Service struct {
	repo recordRepository
}

func (s *Service) Load(ctx context.Context, id string) ([]byte, error) {
	raw, err := s.loadRaw(ctx, id)
	if err != nil {
		return nil, normalizeLoadError(err)
	}
	return cloneForCaller(raw), nil
}

func (s *Service) loadRaw(ctx context.Context, id string) ([]byte, error) {
	return s.repo.Load(ctx, id)
}

func normalizeLoadError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRecordNotFound
	}
	return fmt.Errorf("load record: %w", err)
}

func cloneForCaller(raw []byte) []byte {
	return bytes.Clone(raw)
}
