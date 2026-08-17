//go:build integration

package referenceservice_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresinbox"
	"github.com/jackc/pgx/v5"
)

// inboxArticleAdapter is the composition-root shape for a real consumer. It
// owns the concrete transaction and claim; article.Service still sees only its
// feature-owned repository and Atomically ports.
type inboxArticleAdapter struct {
	pool *postgres.Pool
}

func (adapter inboxArticleAdapter) consume(
	ctx context.Context,
	consumerIdentity, messageID string,
	draft article.Draft,
) (bool, error) {
	applied := false
	err := adapter.pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		claimed, err := postgresinbox.Claim(ctx, tx, consumerIdentity, messageID)
		if err != nil || !claimed {
			return err
		}
		repository := pgRepository{querier: tx}
		service, err := article.NewService(repository, boundArticleTransaction{repository: repository})
		if err != nil {
			return err
		}
		if _, err := service.Create(ctx, draft); err != nil {
			return err
		}
		applied = true
		return nil
	})
	return applied, err
}

type boundArticleTransaction struct {
	repository article.Repository
}

func (transaction boundArticleTransaction) Do(_ context.Context, fn func(article.Repository) error) error {
	return fn(transaction.repository)
}

func TestPostgresInboxAdapterPlacement(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	dsn := pgtest.Migrated(t, os.DirFS("../.."), "migrations")
	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       4,
		AcquireTimeout:     time.Second,
		ConnMaxLifetime:    time.Hour,
		StatementTimeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatalf("postgres.New(): %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.PGX().Exec(ctx, unitOfWorkSchema); err != nil {
		t.Fatalf("create article schema: %v", err)
	}
	adapter := inboxArticleAdapter{pool: pool}
	draft := article.Draft{Slug: "deduplicated", Title: "Deduplicated", Summary: "Claim and effect share one transaction."}

	applied, err := adapter.consume(ctx, "EVENTS/articles", "logical-message", draft)
	if err != nil || !applied {
		t.Fatalf("first consume = applied %t, error %v", applied, err)
	}
	applied, err = adapter.consume(ctx, "EVENTS/articles", "logical-message", draft)
	if err != nil || applied {
		t.Fatalf("duplicate consume = applied %t, error %v", applied, err)
	}

	var articles, events, claims int
	if err := pool.PGX().QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM articles WHERE slug = $1),
			(SELECT count(*) FROM article_events WHERE slug = $1),
			(SELECT count(*) FROM postgres_inbox_claims
			 WHERE consumer_identity = 'EVENTS/articles' AND message_id = 'logical-message')
	`, draft.Slug).Scan(&articles, &events, &claims); err != nil {
		t.Fatalf("read adapter state: %v", err)
	}
	if articles != 1 || events != 1 || claims != 1 {
		t.Fatalf("adapter state = articles %d events %d claims %d, want 1/1/1", articles, events, claims)
	}
}
