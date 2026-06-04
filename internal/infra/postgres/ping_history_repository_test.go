package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakePingHistoryQuerier struct {
	create func(ctx context.Context, payload string) (sqlcgen.PingHistory, error)
	list   func(ctx context.Context, limit int32) ([]sqlcgen.PingHistory, error)
}

type pingHistoryContextKey struct{}

func newPingHistoryRepositoryWithQuerier(t *testing.T, queries pingHistoryQuerier) *PingHistoryRepository {
	t.Helper()

	repo, err := newPingHistoryRepository(queries)
	if err != nil {
		t.Fatalf("newPingHistoryRepository() error = %v, want nil", err)
	}
	return repo
}

func (f fakePingHistoryQuerier) CreatePingHistory(ctx context.Context, payload string) (sqlcgen.PingHistory, error) {
	if f.create == nil {
		return sqlcgen.PingHistory{}, errors.New("unexpected CreatePingHistory call")
	}
	return f.create(ctx, payload)
}

func (f fakePingHistoryQuerier) ListRecentPingHistory(ctx context.Context, limit int32) ([]sqlcgen.PingHistory, error) {
	if f.list == nil {
		return nil, errors.New("unexpected ListRecentPingHistory call")
	}
	return f.list(ctx, limit)
}

func TestNewPingHistoryRepositoryRejectsNilPool(t *testing.T) {
	t.Parallel()

	repo, err := NewPingHistoryRepository(nil)
	if err == nil {
		t.Fatal("NewPingHistoryRepository(nil) error = nil, want non-nil")
	}
	if repo != nil {
		t.Fatalf("NewPingHistoryRepository(nil) repo = %#v, want nil", repo)
	}
	if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("NewPingHistoryRepository(nil) error = %v, want ErrPingHistoryRepository", err)
	}

	repo, err = NewPingHistoryRepository(&Pool{})
	if err == nil {
		t.Fatal("NewPingHistoryRepository(empty pool) error = nil, want non-nil")
	}
	if repo != nil {
		t.Fatalf("NewPingHistoryRepository(empty pool) repo = %#v, want nil", repo)
	}
	if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("NewPingHistoryRepository(empty pool) error = %v, want ErrPingHistoryRepository", err)
	}
}

func TestNewPingHistoryRepositoryRejectsNilQuerier(t *testing.T) {
	t.Parallel()

	repo, err := newPingHistoryRepository(nil)
	if err == nil {
		t.Fatal("newPingHistoryRepository(nil) error = nil, want non-nil")
	}
	if repo != nil {
		t.Fatalf("newPingHistoryRepository(nil) repo = %#v, want nil", repo)
	}
	if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("newPingHistoryRepository(nil) error = %v, want ErrPingHistoryRepository", err)
	}
}

func TestPingHistoryRepositoryRejectsNilAndZeroValueUse(t *testing.T) {
	t.Parallel()

	var nilRepo *PingHistoryRepository
	if _, err := nilRepo.Create(context.Background(), "payload"); err == nil {
		t.Fatal("(*PingHistoryRepository)(nil).Create() error = nil, want non-nil")
	} else if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("(*PingHistoryRepository)(nil).Create() error = %v, want ErrPingHistoryRepository", err)
	}
	if _, err := nilRepo.ListRecent(context.Background(), 1); err == nil {
		t.Fatal("(*PingHistoryRepository)(nil).ListRecent() error = nil, want non-nil")
	} else if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("(*PingHistoryRepository)(nil).ListRecent() error = %v, want ErrPingHistoryRepository", err)
	}

	zeroRepo := &PingHistoryRepository{}
	if _, err := zeroRepo.Create(context.Background(), "payload"); err == nil {
		t.Fatal("zero-value Create() error = nil, want non-nil")
	} else if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("zero-value Create() error = %v, want ErrPingHistoryRepository", err)
	}
	if _, err := zeroRepo.ListRecent(context.Background(), 1); err == nil {
		t.Fatal("zero-value ListRecent() error = nil, want non-nil")
	} else if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("zero-value ListRecent() error = %v, want ErrPingHistoryRepository", err)
	}
}

func TestPingHistoryRepositoryCreate(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	const contextMarker = "create-context"
	repo := newPingHistoryRepositoryWithQuerier(t, fakePingHistoryQuerier{
		create: func(ctx context.Context, payload string) (sqlcgen.PingHistory, error) {
			if got := ctx.Value(pingHistoryContextKey{}); got != contextMarker {
				t.Fatalf("CreatePingHistory() context marker = %v, want %q", got, contextMarker)
			}
			if payload != "ok" {
				t.Fatalf("CreatePingHistory() payload = %q, want %q", payload, "ok")
			}
			return sqlcgen.PingHistory{
				ID:      7,
				Payload: "ok",
				CreatedAt: pgtype.Timestamptz{
					Time:  createdAt,
					Valid: true,
				},
			}, nil
		},
	})

	ctx := context.WithValue(context.Background(), pingHistoryContextKey{}, contextMarker)
	got, err := repo.Create(ctx, "ok")
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}
	if got.ID != 7 || got.Payload != "ok" || !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("Create() = %#v, want ID=7 Payload=ok CreatedAt=%v", got, createdAt)
	}
}

func TestPingHistoryRepositoryCreateWrapsQueryError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("write failed")
	repo := newPingHistoryRepositoryWithQuerier(t, fakePingHistoryQuerier{
		create: func(context.Context, string) (sqlcgen.PingHistory, error) {
			return sqlcgen.PingHistory{}, sentinel
		},
	})

	_, err := repo.Create(context.Background(), "boom")
	if err == nil {
		t.Fatal("Create() error = nil, want non-nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("Create() error = %v, want wrapped %v", err, sentinel)
	}
	if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("Create() error = %v, want ErrPingHistoryRepository", err)
	}
	if !strings.Contains(err.Error(), "create ping history") {
		t.Fatalf("Create() error = %q, want context prefix", err.Error())
	}
}

func TestPingHistoryRepositoryCreateRejectsNullCreatedAt(t *testing.T) {
	t.Parallel()

	repo := newPingHistoryRepositoryWithQuerier(t, fakePingHistoryQuerier{
		create: func(context.Context, string) (sqlcgen.PingHistory, error) {
			return sqlcgen.PingHistory{ID: 9, Payload: "x"}, nil
		},
	})

	_, err := repo.Create(context.Background(), "x")
	if err == nil {
		t.Fatal("Create() error = nil, want non-nil")
	}
	if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("Create() error = %v, want ErrPingHistoryRepository", err)
	}
	if !strings.Contains(err.Error(), "created_at is null") {
		t.Fatalf("Create() error = %q, want null created_at detail", err.Error())
	}
}

func TestPingHistoryRepositoryListRecent(t *testing.T) {
	t.Parallel()

	firstAt := time.Date(2026, time.February, 1, 10, 0, 0, 0, time.UTC)
	secondAt := firstAt.Add(-time.Minute)
	const contextMarker = "list-context"
	repo := newPingHistoryRepositoryWithQuerier(t, fakePingHistoryQuerier{
		list: func(ctx context.Context, limit int32) ([]sqlcgen.PingHistory, error) {
			if got := ctx.Value(pingHistoryContextKey{}); got != contextMarker {
				t.Fatalf("ListRecentPingHistory() context marker = %v, want %q", got, contextMarker)
			}
			if limit != 2 {
				t.Fatalf("ListRecentPingHistory() limit = %d, want 2", limit)
			}
			return []sqlcgen.PingHistory{
				{
					ID:      10,
					Payload: "a",
					CreatedAt: pgtype.Timestamptz{
						Time:  firstAt,
						Valid: true,
					},
				},
				{
					ID:      9,
					Payload: "b",
					CreatedAt: pgtype.Timestamptz{
						Time:  secondAt,
						Valid: true,
					},
				},
			}, nil
		},
	})

	ctx := context.WithValue(context.Background(), pingHistoryContextKey{}, contextMarker)
	got, err := repo.ListRecent(ctx, 2)
	if err != nil {
		t.Fatalf("ListRecent() error = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListRecent() len = %d, want 2", len(got))
	}
	if got[0].ID != 10 || got[1].ID != 9 {
		t.Fatalf("ListRecent() IDs = [%d,%d], want [10,9]", got[0].ID, got[1].ID)
	}
	if got[0].Payload != "a" || got[1].Payload != "b" {
		t.Fatalf("ListRecent() payloads = [%q,%q], want [a,b]", got[0].Payload, got[1].Payload)
	}
	if !got[0].CreatedAt.Equal(firstAt) || !got[1].CreatedAt.Equal(secondAt) {
		t.Fatalf("ListRecent() CreatedAt = [%v,%v], want [%v,%v]", got[0].CreatedAt, got[1].CreatedAt, firstAt, secondAt)
	}
}

func TestPingHistoryRepositoryListRecentErrors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("read failed")
	repo := newPingHistoryRepositoryWithQuerier(t, fakePingHistoryQuerier{
		list: func(context.Context, int32) ([]sqlcgen.PingHistory, error) {
			return nil, sentinel
		},
	})

	_, err := repo.ListRecent(context.Background(), 1)
	if err == nil {
		t.Fatal("ListRecent() error = nil, want non-nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("ListRecent() error = %v, want wrapped %v", err, sentinel)
	}
	if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("ListRecent() error = %v, want ErrPingHistoryRepository", err)
	}
	if !strings.Contains(err.Error(), "list recent ping history") {
		t.Fatalf("ListRecent() error = %q, want context prefix", err.Error())
	}

	_, err = repo.ListRecent(context.Background(), 0)
	if err == nil {
		t.Fatal("ListRecent(limit=0) error = nil, want non-nil")
	}
	if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("ListRecent(limit=0) error = %v, want ErrPingHistoryRepository", err)
	}
}

func TestPingHistoryRepositoryListRecentRejectsOverSampleLimit(t *testing.T) {
	t.Parallel()

	var listCalled bool
	repo := newPingHistoryRepositoryWithQuerier(t, fakePingHistoryQuerier{
		list: func(context.Context, int32) ([]sqlcgen.PingHistory, error) {
			listCalled = true
			return nil, nil
		},
	})

	_, err := repo.ListRecent(context.Background(), pingHistorySampleMaxListLimit+1)
	if err == nil {
		t.Fatal("ListRecent(over sample max) error = nil, want non-nil")
	}
	if !errors.Is(err, ErrPingHistoryRepository) {
		t.Fatalf("ListRecent(over sample max) error = %v, want ErrPingHistoryRepository", err)
	}
	if !strings.Contains(err.Error(), "limit must be <=") {
		t.Fatalf("ListRecent(over sample max) error = %q, want limit max detail", err.Error())
	}
	if listCalled {
		t.Fatal("ListRecent(over sample max) called query, want rejection before SQL")
	}
}
