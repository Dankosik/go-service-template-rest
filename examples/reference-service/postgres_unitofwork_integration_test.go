//go:build integration

package referenceservice_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}

// This file is the proof that article.Store binds to a real transaction
// without the feature package ever seeing pgx.
//
// The memory adapter proves the use case rolls back; it cannot prove the port has
// a PostgreSQL implementation, and a port that turns out not to fit the datastore
// is worse than no port. The adapter below is what a service writes once, in its
// own infra package, and is the whole answer to "how do I compose two repository
// calls in one transaction here".

const unitOfWorkSchema = `
CREATE TABLE articles (
	slug      text PRIMARY KEY,
	title     text NOT NULL,
	summary   text NOT NULL,
	published boolean NOT NULL
);
CREATE TABLE article_events (
	id   bigserial PRIMARY KEY,
	slug text NOT NULL,
	-- The CHECK is what the rollback test trips. NOT NULL alone would not: an
	-- empty kind is a value, so the insert would succeed and the test would prove
	-- nothing while still reporting green.
	kind text NOT NULL CHECK (kind <> '')
);`

// pgAdapter binds the feature's ports to a PostgreSQL pool.
//
// Do is the only interesting method: it takes the pool's transaction and hands fn
// a repository built over the pgx.Tx. Because repository statements are written
// against the same DBTX shape sqlc generates, so the same code serves inside
// and outside a transaction.
type pgAdapter struct {
	pool *pgxpool.Pool
}

func (a pgAdapter) Do(ctx context.Context, fn func(article.Writer) error) error {
	return postgres.InTx(ctx, a.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return fn(pgRepository{tx: tx})
	})
}

func (a pgAdapter) FindBySlug(ctx context.Context, slug string) (article.Article, error) {
	var found article.Article
	err := a.pool.QueryRow(
		ctx,
		"SELECT slug, title, summary, published FROM articles WHERE slug = $1",
		slug,
	).Scan(&found.Slug, &found.Title, &found.Summary, &found.Published)
	if errors.Is(err, pgx.ErrNoRows) {
		return article.Article{}, article.ErrNotFound
	}
	if err != nil {
		return article.Article{}, err
	}
	return found, nil
}

type pgRepository struct {
	tx pgx.Tx
}

func (r pgRepository) Create(ctx context.Context, created article.Article) error {
	_, err := r.tx.Exec(
		ctx,
		"INSERT INTO articles (slug, title, summary, published) VALUES ($1, $2, $3, $4)",
		created.Slug, created.Title, created.Summary, created.Published,
	)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == pgerrcode.UniqueViolation {
		return fmt.Errorf("%w: %w", article.ErrAlreadyExists, err)
	}
	return err
}

func (r pgRepository) AppendEvent(ctx context.Context, event article.Event) error {
	_, err := r.tx.Exec(
		ctx,
		"INSERT INTO article_events (slug, kind) VALUES ($1, $2)",
		event.Slug, string(event.Kind),
	)
	return err
}

func TestUnitOfWorkMapsDuplicateSlug(t *testing.T) {
	_, service := newUnitOfWorkService(t)
	draft := article.Draft{Slug: "same", Title: "Same", Summary: "Created once."}

	if _, err := service.Create(t.Context(), draft); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	if _, err := service.Create(t.Context(), draft); !errors.Is(err, article.ErrAlreadyExists) {
		t.Fatalf("second Create() error = %v, want %v", err, article.ErrAlreadyExists)
	}
}

// TestUnitOfWorkCommitsBothWrites is the ordinary path: the article and the event
// that announces it land together.
func TestUnitOfWorkCommitsBothWrites(t *testing.T) {
	adapter, service := newUnitOfWorkService(t)

	created, err := service.Create(t.Context(), article.Draft{
		Slug: "clear-owners", Title: "Clear owners", Summary: "Keep behavior with its owner.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := adapter.FindBySlug(t.Context(), created.Slug)
	if err != nil {
		t.Fatalf("FindBySlug() error = %v", err)
	}
	if found != created {
		t.Fatalf("stored article = %+v, want %+v", found, created)
	}
	if got := countEvents(t, adapter, created.Slug); got != 1 {
		t.Fatalf("events for %q = %d, want 1", created.Slug, got)
	}
}

// TestUnitOfWorkRollsBackTheArticleWhenTheEventFails is the property no unit test
// can make: the rollback has to come from the database.
//
// The event insert is made to fail by a statement the server rejects, so the
// failure arrives the way a real one does — from PostgreSQL, mid-transaction —
// rather than from a stub that decided to return an error.
func TestUnitOfWorkRollsBackTheArticleWhenTheEventFails(t *testing.T) {
	adapter, _ := newUnitOfWorkService(t)

	const slug = "rolled-back"
	err := adapter.Do(t.Context(), func(writer article.Writer) error {
		if createErr := writer.Create(t.Context(), article.Article{
			Slug: slug, Title: "t", Summary: "s", Published: true,
		}); createErr != nil {
			return createErr
		}
		// An empty kind violates the CHECK constraint, so PostgreSQL refuses the
		// insert and aborts the transaction that already holds the article.
		return writer.AppendEvent(t.Context(), article.Event{Slug: slug, Kind: ""})
	})
	if err == nil {
		t.Fatal("the event insert succeeded, so this run proves nothing about rollback")
	}

	if _, findErr := adapter.FindBySlug(t.Context(), slug); !errors.Is(findErr, article.ErrNotFound) {
		t.Fatalf("FindBySlug() error = %v, want the article rolled back with the event", findErr)
	}
	if got := countEvents(t, adapter, slug); got != 0 {
		t.Fatalf("events for %q = %d, want 0", slug, got)
	}
}

func newUnitOfWorkService(t *testing.T) (pgAdapter, *article.Service) {
	t.Helper()

	dsn := pgtest.DSN(t)
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	pool, err := postgres.Open(ctx, postgres.Options{
		DSN: dsn,

		MaxOpenConns: 4,
	})
	if err != nil {
		t.Fatalf("postgres.Open() error = %v", err)
	}
	t.Cleanup(pool.Close)

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if _, err := conn.Exec(ctx, unitOfWorkSchema); err != nil {
		conn.Release()
		t.Fatalf("create unit-of-work schema: %v", err)
	}
	conn.Release()

	adapter := pgAdapter{pool: pool}
	service, err := article.NewService(adapter)
	if err != nil {
		t.Fatalf("article.NewService() error = %v", err)
	}
	return adapter, service
}

func countEvents(t *testing.T, adapter pgAdapter, slug string) int {
	t.Helper()

	var count int
	if err := adapter.pool.QueryRow(t.Context(), "SELECT count(*) FROM article_events WHERE slug = $1", slug).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	return count
}
