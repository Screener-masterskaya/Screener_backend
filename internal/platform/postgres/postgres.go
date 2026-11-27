package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Screener-masterskaya/Screener_backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates a PostgreSQL connection pool and verifies connectivity.
// The caller is responsible for closing the returned pool.
func NewPool(
	ctx context.Context,
	databaseURL string,
	cfg config.PostgresPool,
) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, errors.New("invalid PostgreSQL connection configuration")
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL connection pool: %w", err)
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer pingCancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to establish PostgreSQL connection: %w", err)
	}

	return pool, nil
}
