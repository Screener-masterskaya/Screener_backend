package metrics

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// PoolCollector экспортирует статистику pgxpool в виде метрик Prometheus.
// На каждый scrape считывается свежий снимок из pool.Stat().
type PoolCollector struct {
	pool *pgxpool.Pool

	acquireCount         *prometheus.Desc
	acquireDuration      *prometheus.Desc
	acquiredConns        *prometheus.Desc
	canceledAcquireCount *prometheus.Desc
	constructingConns    *prometheus.Desc
	emptyAcquireCount    *prometheus.Desc
	idleConns            *prometheus.Desc
	maxConns             *prometheus.Desc
	totalConns           *prometheus.Desc
	newConnsCount        *prometheus.Desc
	maxLifetimeDestroy   *prometheus.Desc
	maxIdleDestroy       *prometheus.Desc
}

// NewPoolCollector создаёт коллектор для переданного пула. Коллектор
// безопасен для конкурентного использования и не выполняет никакого I/O,
// кроме чтения статистики пула.
func NewPoolCollector(pool *pgxpool.Pool) *PoolCollector {
	const subsystem = "pgxpool"

	desc := func(name, help string) *prometheus.Desc {
		return prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, name),
			help, nil, nil,
		)
	}

	return &PoolCollector{
		pool: pool,

		acquireCount:         desc("acquire_count_total", "Cumulative count of successful acquires from the pool."),
		acquireDuration:      desc("acquire_duration_seconds_total", "Total duration of all successful acquires from the pool in seconds."),
		acquiredConns:        desc("acquired_conns", "Number of currently acquired connections in the pool."),
		canceledAcquireCount: desc("canceled_acquire_count_total", "Cumulative count of acquires from the pool that were canceled by a context."),
		constructingConns:    desc("constructing_conns", "Number of conns with construction in progress in the pool."),
		emptyAcquireCount:    desc("empty_acquire_count_total", "Cumulative count of successful acquires from the pool that waited for a resource to be released or constructed."),
		idleConns:            desc("idle_conns", "Number of currently idle conns in the pool."),
		maxConns:             desc("max_conns", "Maximum size of the pool."),
		totalConns:           desc("total_conns", "Total number of resources currently in the pool."),
		newConnsCount:        desc("new_conns_count_total", "Cumulative count of new connections opened."),
		maxLifetimeDestroy:   desc("max_lifetime_destroy_count_total", "Cumulative count of connections destroyed due to MaxConnLifetime."),
		maxIdleDestroy:       desc("max_idle_destroy_count_total", "Cumulative count of connections destroyed due to MaxConnIdleTime."),
	}
}

// Describe реализует интерфейс prometheus.Collector.
func (c *PoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.acquireCount
	ch <- c.acquireDuration
	ch <- c.acquiredConns
	ch <- c.canceledAcquireCount
	ch <- c.constructingConns
	ch <- c.emptyAcquireCount
	ch <- c.idleConns
	ch <- c.maxConns
	ch <- c.totalConns
	ch <- c.newConnsCount
	ch <- c.maxLifetimeDestroy
	ch <- c.maxIdleDestroy
}

// Collect реализует интерфейс prometheus.Collector.
func (c *PoolCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.pool.Stat()

	gauge := func(desc *prometheus.Desc, value float64) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value)
	}
	counter := func(desc *prometheus.Desc, value float64) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, value)
	}

	counter(c.acquireCount, float64(s.AcquireCount()))
	counter(c.acquireDuration, s.AcquireDuration().Seconds())
	gauge(c.acquiredConns, float64(s.AcquiredConns()))
	counter(c.canceledAcquireCount, float64(s.CanceledAcquireCount()))
	gauge(c.constructingConns, float64(s.ConstructingConns()))
	counter(c.emptyAcquireCount, float64(s.EmptyAcquireCount()))
	gauge(c.idleConns, float64(s.IdleConns()))
	gauge(c.maxConns, float64(s.MaxConns()))
	gauge(c.totalConns, float64(s.TotalConns()))
	counter(c.newConnsCount, float64(s.NewConnsCount()))
	counter(c.maxLifetimeDestroy, float64(s.MaxLifetimeDestroyCount()))
	counter(c.maxIdleDestroy, float64(s.MaxIdleDestroyCount()))
}

// RegisterPool регистрирует PoolCollector для pool в reg.
// Если reg равен nil, используется prometheus.DefaultRegisterer.
func RegisterPool(reg prometheus.Registerer, pool *pgxpool.Pool) error {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	return reg.Register(NewPoolCollector(pool))
}
