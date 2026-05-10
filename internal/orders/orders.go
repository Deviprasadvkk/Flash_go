package orders

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Order struct {
	ItemID    string
	UserID    string
	CreatedAt time.Time
}

type Store struct {
	pool *pgxpool.Pool
}

// NewPostgres creates a new Store connected to the given DSN (e.g. "postgres://user:pass@host:5432/db").
func NewPostgres(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx connect: %w", err)
	}
	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS orders (
  id BIGSERIAL PRIMARY KEY,
  item_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`)
	return err
}

// Save inserts an order record.
func (s *Store) Save(ctx context.Context, o Order) error {
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO orders (item_id, user_id, created_at) VALUES ($1, $2, $3)`, o.ItemID, o.UserID, o.CreatedAt)
	return err
}

// Close releases DB resources.
func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}
