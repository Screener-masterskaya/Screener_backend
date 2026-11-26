package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/Screener-masterskaya/Screener_backend/internal/config"
	"github.com/Screener-masterskaya/Screener_backend/internal/platform/postgres"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

var cfg = config.PostgresPool{
	MaxConns:        10,
	MaxConnLifetime: time.Hour,
	ConnectTimeout:  5 * time.Second,
}

const password = "testSecret123"

func TestNewPoolSuccess(t *testing.T) {
	ctx := context.Background()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:18.7-alpine3.24",
		tcpostgres.WithDatabase("screener"),
		tcpostgres.WithUsername("user"),
		tcpostgres.WithPassword(password),
		tcpostgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err, "failed to start container")

	databaseURL, err := container.ConnectionString(
		ctx,
		"sslmode=disable",
	)
	require.NoError(t, err, "failed to get connection string")

	pool, err := postgres.NewPool(ctx, databaseURL, cfg)
	require.NoError(t, err, "failed to initialize PostgreSQL pool")
	require.NotNil(t, pool)
	t.Cleanup(pool.Close)

	queryCtx, queryCancel := context.WithTimeout(ctx, 5*time.Second)
	defer queryCancel()

	var result int

	err = pool.QueryRow(queryCtx, "SELECT 1").Scan(&result)
	require.NoError(t, err, "failed to execute SQL query")
	require.Equal(t, 1, result)
}

func TestNewPoolInvalidConfig(t *testing.T) {
	ctx := context.Background()

	databaseURL := "postgres://user:" + password + "@localhost:abc/screener?sslmode=disable"

	pool, err := postgres.NewPool(ctx, databaseURL, cfg)
	require.EqualError(t, err, "invalid PostgreSQL connection configuration")
	require.Nil(t, pool)
	require.NotContains(t, err.Error(), password)
}

func TestNewPoolUnavailableDB(t *testing.T) {
	ctx := context.Background()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:18.7-alpine3.24",
		tcpostgres.WithDatabase("screener"),
		tcpostgres.WithUsername("user"),
		tcpostgres.WithPassword(password),
		tcpostgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err, "failed to start container")

	databaseURL, err := container.ConnectionString(
		ctx,
		"sslmode=disable",
	)
	require.NoError(t, err, "failed to get connection string")

	err = container.Stop(ctx, nil)
	require.NoError(t, err, "failed to stop container")

	pool, err := postgres.NewPool(ctx, databaseURL, cfg)
	require.EqualError(t, err, "failed to establish PostgreSQL connection")
	require.Nil(t, pool)
	require.NotContains(t, err.Error(), password)
}
