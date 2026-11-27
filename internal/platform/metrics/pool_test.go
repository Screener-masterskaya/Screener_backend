package metrics

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolCollectorExportsStats(t *testing.T) {
	// Пул, указывающий на недоступную базу, всё равно отдаёт статистику:
	// pgxpool не требует живых соединений для работы Stat().
	pool, err := pgxpool.New(context.Background(), "postgres://screener:example@127.0.0.1:1/screener")
	require.NoError(t, err)
	defer pool.Close()

	reg := prometheus.NewRegistry()
	require.NoError(t, RegisterPool(reg, pool))

	body := gatherText(t, reg)

	assert.Contains(t, body, "screener_pgxpool_max_conns")
	assert.Contains(t, body, "screener_pgxpool_total_conns")
	assert.Contains(t, body, "screener_pgxpool_idle_conns")
	assert.Contains(t, body, "screener_pgxpool_acquired_conns")
	assert.Contains(t, body, "screener_pgxpool_acquire_count_total")
	assert.Contains(t, body, "screener_pgxpool_acquire_duration_seconds_total")
}

func TestPoolCollectorRegisteredWithNilRegistryUsesDefault(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://screener:example@127.0.0.1:1/screener")
	require.NoError(t, err)
	defer pool.Close()

	collector := NewPoolCollector(pool)
	prometheus.DefaultRegisterer.MustRegister(collector)
	defer prometheus.DefaultRegisterer.Unregister(collector)

	body := gatherText(t, prometheus.DefaultGatherer)

	assert.Contains(t, body, "screener_pgxpool_max_conns")
	// Дефолтный gatherer также содержит коллекторы Go runtime и процесса.
	assert.Contains(t, body, "go_goroutines")
	assert.Contains(t, body, "process_cpu_seconds_total")
}
