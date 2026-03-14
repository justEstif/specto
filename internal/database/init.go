package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InitDB creates a connection pool and returns the query handle and pool.
// The caller owns the lifecycle and must call pool.Close() when done.
func InitDB(databaseURL string) (*Queries, *pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, nil, fmt.Errorf("database URL must not be empty")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return New(pool), pool, nil
}
